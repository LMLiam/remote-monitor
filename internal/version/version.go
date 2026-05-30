package version

import "fmt"

// GoReleaser sets these variables with ldflags. The linker requires package
// variables for this, so keep them private and expose immutable snapshots.
//
//nolint:gochecknoglobals // Build metadata must be set through linker-addressable variables.
var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildDate    = "unknown"
)

// Info contains build metadata for the running binary.
type Info struct {
	Version string
	Commit  string
	Date    string
}

// Current returns build metadata embedded by ldflags, or development defaults.
func Current() Info {
	return Info{
		Version: buildVersion,
		Commit:  buildCommit,
		Date:    buildDate,
	}
}

// String formats version metadata for terminal output.
func (info Info) String() string {
	return fmt.Sprintf("remote-monitor %s (commit %s, built %s)", info.Version, info.Commit, info.Date)
}
