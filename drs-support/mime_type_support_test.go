package drs_support

import "testing"

func TestDeriveMimeTypeFromDataObjectPath(t *testing.T) {
	mimeType := DeriveMimeTypeFromDataObjectPath("/tempZone/home/rods/file.txt")
	if mimeType != "text/plain" {
		t.Fatalf("expected text/plain, got %q", mimeType)
	}
}

func TestDeriveMimeTypeFromDataObjectPathUnknownExtension(t *testing.T) {
	mimeType := DeriveMimeTypeFromDataObjectPath("/tempZone/home/rods/file.unknown-extension")
	if mimeType != "" {
		t.Fatalf("expected empty mime type for unknown extension, got %q", mimeType)
	}
}

func TestDeriveMimeTypeFromDataObjectPathWithoutExtension(t *testing.T) {
	mimeType := DeriveMimeTypeFromDataObjectPath("/tempZone/home/rods/file")
	if mimeType != "" {
		t.Fatalf("expected empty mime type for extensionless path, got %q", mimeType)
	}
}
