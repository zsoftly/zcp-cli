// Package version holds the CLI version, injected at build time via ldflags.
package version

// Version is set at build time via:
//
//	go build -ldflags "-X github.com/zsoftly/zcp-cli/internal/version.Version=1.2.3"
var Version = "dev"
