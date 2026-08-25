package source

import "fmt"

// unknownSourceError is returned when a project references a source name that is
// not registered — normally a programming error rather than a user-facing one.
func unknownSourceError(name string) error {
	if name == "" {
		return fmt.Errorf("project has no source")
	}
	return fmt.Errorf("unknown source %q", name)
}
