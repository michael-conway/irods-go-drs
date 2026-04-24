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

func TestChecksumFromReplicaNormalizesSHA256ValueAndType(t *testing.T) {
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

	if internalChecksum.Value != "JzZYwVeBDkKwp8dtxc6ZDZbe287HDy9NkS0+Let9UyQ=" {
		t.Fatalf("expected checksum value without irods prefix, got %q", internalChecksum.Value)
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
