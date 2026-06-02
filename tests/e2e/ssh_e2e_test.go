//go:build integration && !windows

package e2e_test

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

const (
	sshAlias      = "remote-monitor-e2e"
	containerUser = "monitor"
)

var liveSummaryPattern = regexp.MustCompile(`\| state live \| CPU [0-9]+% \| RAM [0-9]+ / [1-9][0-9]* MiB \|`)

func TestRemoteMonitorStreamsSamplesOverSSH(t *testing.T) {
	requireCommand(t, "docker")
	realSSH := requireCommand(t, "ssh")
	requireCommand(t, "ssh-keygen")
	requireCommand(t, "ssh-keyscan")

	repo := repoRoot(t)
	tmp := t.TempDir()
	containerName := fmt.Sprintf("remote-monitor-e2e-%d", time.Now().UnixNano())
	imageTag := strings.ToLower(containerName + ":latest")

	dockerCtx, dockerCancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer dockerCancel()

	commandOutput(t, dockerCtx, repo, nil, "docker", "build", "-t", imageTag, filepath.Join(repo, "tests/e2e/ssh-target"))
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = exec.CommandContext(cleanupCtx, "docker", "rm", "-f", containerName).Run()
		_ = exec.CommandContext(cleanupCtx, "docker", "image", "rm", "-f", imageTag).Run()
	})

	keyPath := filepath.Join(tmp, "id_ed25519")
	commandOutput(t, dockerCtx, repo, nil, "ssh-keygen", "-t", "ed25519", "-N", "", "-f", keyPath, "-C", "remote-monitor-e2e")
	publicKey := strings.TrimSpace(readFile(t, keyPath+".pub"))

	containerID := strings.TrimSpace(commandOutput(t, dockerCtx, repo, nil, "docker", "run", "-d", "--rm",
		"--name", containerName,
		"-e", "AUTHORIZED_KEY="+publicKey,
		"-p", "127.0.0.1::22",
		imageTag,
	))
	if containerID == "" {
		t.Fatal("docker run returned an empty container id")
	}

	host, port := dockerHostPort(t, dockerCtx, repo, containerName)
	sshDir := filepath.Join(tmp, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatalf("create ssh dir: %v", err)
	}
	knownHosts := filepath.Join(sshDir, "known_hosts")
	writeFile(t, knownHosts, waitForHostKey(t, repo, host, port))

	sshConfig := filepath.Join(sshDir, "config")
	writeFile(t, sshConfig, fmt.Sprintf(`Host %s
  HostName %s
  Port %s
  User %s
  IdentityFile %s
  IdentitiesOnly yes
  BatchMode yes
  StrictHostKeyChecking yes
  UserKnownHostsFile %s
  GlobalKnownHostsFile /dev/null
  LogLevel ERROR
`, sshAlias, host, port, containerUser, keyPath, knownHosts))

	waitForSSHLogin(t, repo, sshConfig)
	wrapperBin := writeSSHWrapper(t, tmp)

	binaryPath := filepath.Join(tmp, "remote-monitor")
	commandOutput(t, dockerCtx, repo, nil, "go", "build", "-o", binaryPath, "./cmd/remote-monitor")

	output, stderr, ok := waitForLiveMonitorLine(t, repo, binaryPath, tmp, wrapperBin, realSSH, sshConfig)
	if !ok {
		t.Fatalf("monitor did not render a live sample\nstdout:\n%s\nstderr:\n%s\ncontainer logs:\n%s\nssh config:\n%s",
			output,
			stderr,
			commandOutputAllowError(repo, "docker", "logs", containerName),
			redactHome(readFile(t, sshConfig), tmp),
		)
	}
}

func requireCommand(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		if os.Getenv("CI") == "" {
			t.Skipf("%s is required for SSH e2e integration tests", name)
		}
		t.Fatalf("%s is required for SSH e2e integration tests: %v", name, err)
	}

	return path
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func dockerHostPort(t *testing.T, ctx context.Context, repo, containerName string) (host string, port string) {
	t.Helper()
	output := strings.TrimSpace(commandOutput(t, ctx, repo, nil, "docker", "port", containerName, "22/tcp"))
	host, port, err := net.SplitHostPort(output)
	if err != nil {
		t.Fatalf("parse docker port %q: %v", output, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}

	return host, port
}

func waitForHostKey(t *testing.T, repo, host, port string) string {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		output, err := commandOutputErr(ctx, repo, nil, "ssh-keyscan", "-T", "5", "-p", port, host)
		cancel()
		if err == nil && strings.TrimSpace(output) != "" {
			return output
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for ssh host key from %s:%s", host, port)
	return ""
}

func waitForSSHLogin(t *testing.T, repo, sshConfig string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := commandOutputErr(ctx, repo, []string{"HOME=" + filepath.Dir(filepath.Dir(sshConfig))}, "ssh", "-F", sshConfig, sshAlias, "true")
		cancel()
		if err == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for ssh login to %s", sshAlias)
}

func waitForLiveMonitorLine(
	t *testing.T,
	repo string,
	binaryPath string,
	home string,
	wrapperBin string,
	realSSH string,
	sshConfig string,
) (stdoutText string, stderrText string, ok bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath,
		"-interval", "1",
		"-history", "30",
		"-stale-after", "3",
		"-reconnect-delay", "1",
		"-ssh-connect-timeout", "3",
		"-ssh-server-alive", "1",
		"-ssh-server-alive-count", "1",
		"-ssh-control-persist", "0",
		"-theme", "basic",
		"-no-banner",
		sshAlias,
	)
	cmd.Dir = repo
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}

		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = 5 * time.Second
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"PATH="+wrapperBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"REMOTE_MONITOR_E2E_REAL_SSH="+realSSH,
		"REMOTE_MONITOR_E2E_SSH_CONFIG="+sshConfig,
		"TERM=dumb",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("open monitor stdout: %v", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start monitor: %v", err)
	}

	var lines []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		if liveSummaryPattern.MatchString(line) {
			cancel()
			_ = cmd.Wait()
			return strings.Join(lines, "\n"), stderr.String(), true
		}
	}
	if err := scanner.Err(); err != nil {
		lines = append(lines, "scanner error: "+err.Error())
	}
	_ = cmd.Wait()

	return strings.Join(lines, "\n"), stderr.String(), false
}

func commandOutput(t *testing.T, ctx context.Context, dir string, env []string, name string, args ...string) string {
	t.Helper()
	output, err := commandOutputErr(ctx, dir, env, name, args...)
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, output)
	}

	return output
}

func commandOutputErr(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()

	return output.String(), err
}

func commandOutputAllowError(dir string, name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	output, err := commandOutputErr(ctx, dir, nil, name, args...)
	if err != nil {
		output += "\n" + err.Error()
	}

	return output
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(content)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeSSHWrapper(t *testing.T, tmp string) string {
	t.Helper()
	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("create wrapper bin dir: %v", err)
	}
	wrapperPath := filepath.Join(binDir, "ssh")
	const wrapper = `#!/usr/bin/env bash
set -euo pipefail
exec "${REMOTE_MONITOR_E2E_REAL_SSH}" -F "${REMOTE_MONITOR_E2E_SSH_CONFIG}" "$@"
`
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o700); err != nil {
		t.Fatalf("write ssh wrapper: %v", err)
	}

	return binDir
}

func redactHome(content, home string) string {
	return strings.ReplaceAll(content, home, "$TMPDIR")
}
