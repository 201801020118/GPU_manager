package version

import (
	"fmt"
)

// Info contains version information
type Info struct {
	Version string
	Commit  string
}

// String returns info as a human-friend version string.
func (info Info) String() string {
	return info.Commit
}

// Get returns the overall codebase version. It's for detecting
// what code a binary was built from.
func Get() Info {
	return Info{
		Version: fmt.Sprintf("%s.%s", gitMajor, gitMinor),
		Commit:  gitCommit,
	}
}
