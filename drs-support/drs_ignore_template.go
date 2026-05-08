package drs_support

import _ "embed"

var (
	//go:embed templates/sample.drsignore
	embeddedSampleDRSIgnore string
)

// SampleDRSIgnore returns the built-in .drsignore template used by tooling.
func SampleDRSIgnore() string {
	return embeddedSampleDRSIgnore
}
