package transport

import (
	"crypto/sha256"
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"os"
	"strconv"
	"strings"
	"time"
)

// SSHArgs builds the ssh command arguments used to run the remote sampler.
func SSHArgs(cfg core.Config, intervalSeconds int) []string {
	controlPath := ResolveSSHControlPath(cfg)

	return []string{
		"-T",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=" + strconv.Itoa(durationSeconds(cfg.SSHConnectTimeout)),
		"-o", "ServerAliveInterval=" + strconv.Itoa(durationSeconds(cfg.SSHAliveInterval)),
		"-o", "ServerAliveCountMax=" + strconv.Itoa(max(1, cfg.SSHAliveCountMax)),
		"-o", "TCPKeepAlive=yes",
		"-o", "ControlMaster=auto",
		"-o", "ControlPersist=" + formatSSHDuration(cfg.SSHControlPersist),
		"-o", "ControlPath=" + controlPath,
		cfg.Host,
		"bash", "-s", "--", strconv.Itoa(intervalSeconds),
		normalizedProcessSort(cfg),
		sshShellQuote(cfg.ProcessFilter),
		strconv.Itoa(normalizedProcessCount(cfg)),
	}
}

// ResolveSSHControlPath returns the configured or generated SSH control socket path.
func ResolveSSHControlPath(cfg core.Config) string {
	if cfg.SSHControlPath != "" {
		return cfg.SSHControlPath
	}

	sum := sha256.Sum256([]byte(cfg.Host))

	return fmt.Sprintf("/tmp/rm-%d-%x.sock", os.Getpid(), sum[:6])
}

func normalizedProcessSort(cfg core.Config) string {
	switch cfg.ProcessSort {
	case core.ProcessSortMemory:
		return core.ProcessSortMemory
	default:
		return core.ProcessSortCPU
	}
}

func normalizedProcessCount(cfg core.Config) int {
	if cfg.ProcessCount < 1 {
		return core.DefaultProcessCount
	}

	return cfg.ProcessCount
}

func sshShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func durationSeconds(d time.Duration) int {
	if d <= 0 {
		return 1
	}
	seconds := int(d / time.Second)
	if seconds < 1 {
		return 1
	}

	return seconds
}

func formatSSHDuration(d time.Duration) string {
	if d <= 0 {
		return "0"
	}

	return strconv.Itoa(durationSeconds(d)) + "s"
}
