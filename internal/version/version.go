// Package version holds the single source of truth for the lazydash version.
package version

// Version is the current lazydash version. It is a var so a release build can
// override it with -ldflags "-X .../internal/version.Version=...".
var Version = "v0.2.2"
