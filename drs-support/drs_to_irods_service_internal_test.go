package drs_support

import (
	"testing"
	"time"

	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
)

func TestChecksumFromReplicaPreservesTypeAndValue(t *testing.T) {
	checksum, err := irodstypes.CreateIRODSChecksum("d41d8cd98f00b204e9800998ecf8427e")
	if err != nil {
		t.Fatalf("create checksum: %v", err)
	}

	replica := &irodstypes.IRODSReplica{
		Checksum:   checksum,
		CreateTime: time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC),
		ModifyTime: time.Date(2026, 4, 23, 10, 5, 0, 0, time.UTC),
	}

	internalChecksum := checksumFromReplica(replica)
	if internalChecksum == nil {
		t.Fatal("expected internal checksum to be populated")
	}

	if internalChecksum.Type != "md5" {
		t.Fatalf("expected checksum type md5, got %q", internalChecksum.Type)
	}

	if internalChecksum.Value != "d41d8cd98f00b204e9800998ecf8427e" {
		t.Fatalf("expected checksum value to be preserved, got %q", internalChecksum.Value)
	}
}

func TestChecksumFromReplicaNormalizesSHA256ValueToHex(t *testing.T) {
	// Base64 iRODS SHA256 checksum
	checksum, err := irodstypes.CreateIRODSChecksum("sha2:JzZYwVeBDkKwp8dtxc6ZDZbe287HDy9NkS0+Let9UyQ=")
	if err != nil {
		t.Fatalf("create checksum: %v", err)
	}

	replica := &irodstypes.IRODSReplica{Checksum: checksum}
	internalChecksum := checksumFromReplica(replica)
	if internalChecksum == nil {
		t.Fatal("expected internal checksum to be populated")
	}

	if internalChecksum.Type != "sha-256" {
		t.Fatalf("expected checksum type sha-256, got %q", internalChecksum.Type)
	}

	// JzZYwVeBDkKwp8dtxc6ZDZbe287HDy9NkS0+Let9UyQ= (base64)
	// -> 273658c157810e42b0a7c76dc5ce990d96dedbcec70f2f4d912d3e2deb7d5324 (hex)
	expectedHex := "273658c157810e42b0a7c76dc5ce990d96dedbcec70f2f4d912d3e2deb7d5324"
	if internalChecksum.Value != expectedHex {
		t.Fatalf("expected hex-normalized checksum, got %q", internalChecksum.Value)
	}
}

func TestNormalizeChecksumValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"sha2:JzZYwVeBDkKwp8dtxc6ZDZbe287HDy9NkS0+Let9UyQ=", "273658c157810e42b0a7c76dc5ce990d96dedbcec70f2f4d912d3e2deb7d5324"},
		{"JzZYwVeBDkKwp8dtxc6ZDZbe287HDy9NkS0+Let9UyQ=", "273658c157810e42b0a7c76dc5ce990d96dedbcec70f2f4d912d3e2deb7d5324"},
		{"d41d8cd98f00b204e9800998ecf8427e", "d41d8cd98f00b204e9800998ecf8427e"},
		{"sha2:273658c157810e42b0a7c76dc5ce990d96ded9dec70f2f4d912d3e2deb7d5324", "273658c157810e42b0a7c76dc5ce990d96ded9dec70f2f4d912d3e2deb7d5324"},
		{"md5:d41d8cd98f00b204e9800998ecf8427e", "d41d8cd98f00b204e9800998ecf8427e"},
		{"", ""},
		{"random-string", "random-string"},
	}

	for _, tc := range tests {
		got := normalizeChecksumValue(tc.input)
		if got != tc.expected {
			t.Errorf("normalizeChecksumValue(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestChecksumFromReplicaPreservesMD5ValueAndType(t *testing.T) {
	checksum, err := irodstypes.CreateIRODSChecksum("d41d8cd98f00b204e9800998ecf8427e")
	if err != nil {
		t.Fatalf("create checksum: %v", err)
	}

	replica := &irodstypes.IRODSReplica{Checksum: checksum}
	internalChecksum := checksumFromReplica(replica)
	if internalChecksum == nil {
		t.Fatal("expected internal checksum to be populated")
	}

	if internalChecksum.Type != "md5" {
		t.Fatalf("expected checksum type md5, got %q", internalChecksum.Type)
	}

	if internalChecksum.Value != "d41d8cd98f00b204e9800998ecf8427e" {
		t.Fatalf("expected md5 checksum value to be preserved, got %q", internalChecksum.Value)
	}
}

func TestNormalizedMimeTypeUsesMimeTypeSupportWhenUnset(t *testing.T) {
	mimeType := normalizedMimeType("/tempZone/home/rods/file.txt", "")
	if mimeType != "text/plain" {
		t.Fatalf("expected text/plain from MimeTypeSupport, got %q", mimeType)
	}
}

func TestNormalizedMimeTypePreservesExplicitMimeType(t *testing.T) {
	mimeType := normalizedMimeType("/tempZone/home/rods/file.txt", "application/custom")
	if mimeType != "application/custom" {
		t.Fatalf("expected explicit mime type to be preserved, got %q", mimeType)
	}
}
