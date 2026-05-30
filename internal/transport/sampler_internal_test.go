package transport

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const samplerScriptMode fs.FileMode = 0o600

func TestRemoteSamplerShellSyntax(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is required to validate the remote sampler script")
	}
	if strings.TrimSpace(remoteSampler) == "" {
		t.Fatal("embedded remote sampler script is empty")
	}

	path := filepath.Join(t.TempDir(), "sampler.sh")
	if err := os.WriteFile(path, []byte(remoteSampler), samplerScriptMode); err != nil {
		t.Fatalf("write sampler script: %v", err)
	}

	cmd := exec.Command("bash", "-n", "sampler.sh")
	cmd.Dir = filepath.Dir(path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sampler script failed bash syntax check: %v\n%s", err, output)
	}
}
