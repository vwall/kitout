package buildinfo

import "strings"

const (
	defaultVersion = "dev"
	unknown        = "unknown"
)

var (
	// Version, Commit, and BuildDate are set by release and local build tooling
	// with Go linker flags.
	Version   = defaultVersion
	Commit    = unknown
	BuildDate = unknown
)

// Info describes the metadata embedded in a Kitout binary.
type Info struct {
	Version   string
	Commit    string
	BuildDate string
}

// Current returns normalized build metadata for the running binary.
func Current() Info {
	return Info{
		Version:   valueOrDefault(Version, defaultVersion),
		Commit:    valueOrDefault(Commit, unknown),
		BuildDate: valueOrDefault(BuildDate, unknown),
	}
}

func valueOrDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
