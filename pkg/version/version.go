// Package version provides version information for Ralph.
//
// Version information is set at build time using ldflags:
//
//	go build -ldflags "-X github.com/JanDeDobbeleer/copilot-ralph/pkg/version.Version=1.0.0"
//
// See specs/cli.md for version command specification.
package version

// Build-time variables set via ldflags
var (
	// Version is the semantic version number
	Version = "dev"

	// Commit is the git commit hash
	Commit = "unknown"

	// BuildDate is the build timestamp
	BuildDate = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = "unknown"
)

// Info contains version information
type Info struct {
	Version   string
	Commit    string
	BuildDate string
	GoVersion string
}

// Get returns the version information
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
	}
}
