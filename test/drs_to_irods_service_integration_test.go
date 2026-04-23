//go:build integration
// +build integration

// Test setup:
// This integration test expects the iRODS docker compose stack in
// deployments/docker-test-framework/5-0 to be running and reachable.
//
// Optional environment variables:
//
//	DRS_TEST_IRODS_HOST
//	DRS_TEST_IRODS_PORT
//	DRS_TEST_IRODS_ZONE
//	DRS_TEST_IRODS_USER
//	DRS_TEST_IRODS_PASSWORD
//
// Default behavior when those variables are unset:
//
//	host=localhost
//	port=1247
//	zone=tempZone
//	user=test1
//	password=test
package test

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

func TestCreateDrsObjectFromDataObjectIntegration(t *testing.T) {
	filesystem := newIntegrationIRODSFilesystem(t)
	defer filesystem.Release()

	testDir := fmt.Sprintf("/%s/home/%s/drs-integration-%d", integrationIRODSZone(), integrationIRODSUser(), time.Now().UnixNano())
	if err := filesystem.MakeDir(testDir, true); err != nil {
		t.Fatalf("make dir %q: %v", testDir, err)
	}
	defer func() {
		if err := filesystem.RemoveDir(testDir, true, true); err != nil && filesystem.Exists(testDir) {
			t.Errorf("cleanup dir %q: %v", testDir, err)
		}
	}()

	objectPath := testDir + "/object.txt"
	content := []byte("drs integration test object\n")
	if _, err := filesystem.UploadFileFromBuffer(bytes.NewBuffer(content), objectPath, "", false, true, nil); err != nil {
		t.Fatalf("upload object %q: %v", objectPath, err)
	}

	drsID, err := drs_support.CreateDrsObjectFromDataObject(
		filesystem,
		objectPath,
		"",
		"integration description",
		[]string{"alias-one", "alias-two"},
	)
	if err != nil {
		t.Fatalf("create drs object: %v", err)
	}

	if drsID == "" {
		t.Fatal("expected generated drs id")
	}

	entry, err := filesystem.StatFile(objectPath)
	if err != nil {
		t.Fatalf("stat object %q: %v", objectPath, err)
	}

	if entry.Path != objectPath {
		t.Fatalf("expected path %q, got %q", objectPath, entry.Path)
	}

	if len(entry.IRODSReplicas) == 0 || entry.IRODSReplicas[0].Checksum == nil {
		t.Fatal("expected uploaded object to have a checksum")
	}

	expectedChecksumType := strings.ToLower(string(entry.IRODSReplicas[0].Checksum.Algorithm))
	expectedChecksumValue := entry.IRODSReplicas[0].Checksum.IRODSChecksumString
	expectedMimeType := drs_support.DeriveMimeTypeFromDataObjectPath(objectPath)

	metas, err := filesystem.ListMetadata(objectPath)
	if err != nil {
		t.Fatalf("list metadata for %q: %v", objectPath, err)
	}

	assertMetadataValue(t, metas, drs_support.DrsIdAvuAttrib, drsID)
	assertMetadataValue(t, metas, drs_support.DrsAvuVersionAttrib, expectedChecksumValue)
	assertMetadataValue(t, metas, drs_support.DrsAvuMimeTypeAttrib, expectedMimeType)
	assertMetadataValue(t, metas, drs_support.DrsAvuDescriptionAttrib, "integration description")
	assertMetadataValues(t, metas, drs_support.DrsAvuAliasAttrib, []string{"alias-one", "alias-two"})

	if expectedChecksumType == "md5" || expectedChecksumType == "sha-256" {
	} else {
		t.Fatalf("expected compose-backed object checksum type md5 | sha-256, got %q", expectedChecksumType)

	}

	_, err = drs_support.CreateDrsObjectFromDataObject(filesystem, objectPath, "", "integration description", nil)
	if err == nil {
		t.Fatal("expected second create call to fail for an existing DRS object")
	}

	if !strings.Contains(err.Error(), "already a DRS object") {
		t.Fatalf("expected already a DRS object error, got %v", err)
	}
}

func newIntegrationIRODSFilesystem(t *testing.T) *irodsfs.FileSystem {
	t.Helper()

	account, err := irodstypes.CreateIRODSAccount(
		integrationIRODSHost(),
		integrationIRODSPort(t),
		integrationIRODSUser(),
		integrationIRODSZone(),
		irodstypes.AuthSchemeNative,
		integrationIRODSPassword(),
		"",
	)
	if err != nil {
		t.Fatalf("create iRODS account: %v", err)
	}

	filesystem, err := irodsfs.NewFileSystemWithDefault(account, "irods-go-drs-integration-test")
	if err != nil {
		t.Fatalf("connect to iRODS. This test requires the docker compose stack in deployments/docker-test-framework/5-0 to be running: %v", err)
	}

	return filesystem
}

func integrationIRODSHost() string {
	return getenvDefault("DRS_TEST_IRODS_HOST", "localhost")
}

func integrationIRODSPort(t *testing.T) int {
	t.Helper()

	raw := getenvDefault("DRS_TEST_IRODS_PORT", "1247")
	port, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("invalid DRS_TEST_IRODS_PORT %q: %v", raw, err)
	}

	return port
}

func integrationIRODSZone() string {
	return getenvDefault("DRS_TEST_IRODS_ZONE", "tempZone")
}

func integrationIRODSUser() string {
	return getenvDefault("DRS_TEST_IRODS_USER", "test1")
}

func integrationIRODSPassword() string {
	return getenvDefault("DRS_TEST_IRODS_PASSWORD", "test")
}

func getenvDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func assertMetadataValue(t *testing.T, metas []*irodstypes.IRODSMeta, name string, expected string) {
	t.Helper()

	for _, meta := range metas {
		if meta != nil && meta.Name == name && meta.Value == expected && meta.Units == drs_support.DrsAvuUnit {
			return
		}
	}

	t.Fatalf("expected metadata %q=%q with units %q", name, expected, drs_support.DrsAvuUnit)
}

func assertMetadataValues(t *testing.T, metas []*irodstypes.IRODSMeta, name string, expected []string) {
	t.Helper()

	for _, value := range expected {
		assertMetadataValue(t, metas, name, value)
	}
}
