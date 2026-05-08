package drs_support

import (
	"strings"
	"testing"
)

func TestSampleDRSIgnoreEmbedded(t *testing.T) {
	content := SampleDRSIgnore()
	if strings.TrimSpace(content) == "" {
		t.Fatal("expected embedded .drsignore sample to be non-empty")
	}

	if !strings.Contains(content, "*.tmp") {
		t.Fatalf("expected sample to include wildcard example, got:\n%s", content)
	}

	if !strings.Contains(content, "**/foo") {
		t.Fatalf("expected sample to include double-star example, got:\n%s", content)
	}
}
