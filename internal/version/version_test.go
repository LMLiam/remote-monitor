package version_test

import (
	"testing"

	"github.com/lmliam/remote-monitor/internal/version"
)

func TestInfoStringIncludesVersionCommitAndDate(t *testing.T) {
	t.Parallel()

	info := version.Info{
		Version: "v0.1.0",
		Commit:  "abc1234",
		Date:    "2026-05-30T19:00:00Z",
	}

	const want = "remote-monitor v0.1.0 (commit abc1234, built 2026-05-30T19:00:00Z)"
	if got := info.String(); got != want {
		t.Fatalf("String() = %q", got)
	}
}
