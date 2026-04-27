//go:build integration
// +build integration

package test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodslowfs "github.com/cyverse/go-irodsclient/irods/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

func TestCreateDrsObjectFromDataObjectIntegration(t *testing.T) {
	filesystem := newIntegrationIRODSFilesystem(t)
	defer filesystem.Release()

	testDir := makeIntegrationTestDir(t, filesystem)

	objectPath := testDir + "/object.txt"
	content := []byte("drs integration test object\n")
	if _, err := filesystem.UploadFileFromBuffer(bytes.NewBuffer(content), objectPath, "", false, true, nil); err != nil {
		t.Fatalf("upload object %q: %v", objectPath, err)
	}
	requireIntegrationObjectChecksum(t, filesystem, objectPath)

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
	expectedChecksumValue := normalizedIntegrationChecksumValue(entry.IRODSReplicas[0].Checksum.IRODSChecksumString)
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

func requireIntegrationObjectChecksum(t *testing.T, filesystem *irodsfs.FileSystem, irodsPath string) {
	t.Helper()

	conn, err := filesystem.GetMetadataConnection(true)
	if err != nil {
		t.Fatalf("get metadata connection for %q: %v", irodsPath, err)
	}
	defer func() {
		_ = filesystem.ReturnMetadataConnection(conn)
	}()

	checksum, err := irodslowfs.GetDataObjectChecksum(conn, irodsPath, "")
	if err != nil {
		t.Fatalf("compute checksum for %q: %v", irodsPath, err)
	}
	if checksum == nil || strings.TrimSpace(checksum.IRODSChecksumString) == "" {
		t.Fatalf("expected computed checksum for %q to be populated", irodsPath)
	}
}

func normalizedIntegrationChecksumValue(irodsChecksum string) string {
	trimmed := strings.TrimSpace(irodsChecksum)
	if trimmed == "" {
		return ""
	}

	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) == 2 {
		return parts[1]
	}

	return trimmed
}

func TestQueryAndRemovalMethodsIntegration(t *testing.T) {
	filesystem := newIntegrationIRODSFilesystem(t)
	defer filesystem.Release()

	testDir := makeIntegrationTestDir(t, filesystem)
	nestedDir := testDir + "/nested"
	if err := filesystem.MakeDir(nestedDir, true); err != nil {
		t.Fatalf("make nested dir %q: %v", nestedDir, err)
	}

	rootObjectPath := testDir + "/root.txt"
	nestedObjectPath := nestedDir + "/nested.txt"
	plainObjectPath := testDir + "/plain.txt"

	for path, content := range map[string]string{
		rootObjectPath:   "root drs object\n",
		nestedObjectPath: "nested drs object\n",
		plainObjectPath:  "plain object\n",
	} {
		if _, err := filesystem.UploadFileFromBuffer(bytes.NewBufferString(content), path, "", false, true, nil); err != nil {
			t.Fatalf("upload object %q: %v", path, err)
		}
	}

	rootDrsID, err := drs_support.CreateDrsObjectFromDataObject(filesystem, rootObjectPath, "", "root description", []string{"root-alias"})
	if err != nil {
		t.Fatalf("create root drs object: %v", err)
	}

	nestedDrsID, err := drs_support.CreateDrsObjectFromDataObject(filesystem, nestedObjectPath, "", "nested description", []string{"nested-alias"})
	if err != nil {
		t.Fatalf("create nested drs object: %v", err)
	}

	objectByID, err := drs_support.GetDrsObjectByID(filesystem, rootDrsID)
	if err != nil {
		t.Fatalf("get drs object by id: %v", err)
	}

	if objectByID.AbsolutePath != rootObjectPath {
		t.Fatalf("expected object path %q, got %q", rootObjectPath, objectByID.AbsolutePath)
	}

	objectByPath, err := drs_support.GetDrsObjectByIRODSPath(filesystem, rootObjectPath)
	if err != nil {
		t.Fatalf("get drs object by path: %v", err)
	}

	if objectByPath.Id != rootDrsID {
		t.Fatalf("expected DRS id %q, got %q", rootDrsID, objectByPath.Id)
	}

	nonRecursive, err := drs_support.ListDrsObjectsUnderCollection(filesystem, testDir, false)
	if err != nil {
		t.Fatalf("list non-recursive collection objects: %v", err)
	}

	assertDrsObjectIDs(t, nonRecursive, []string{rootDrsID})

	recursive, err := drs_support.ListDrsObjectsUnderCollection(filesystem, testDir, true)
	if err != nil {
		t.Fatalf("list recursive collection objects: %v", err)
	}

	assertDrsObjectIDs(t, recursive, []string{rootDrsID, nestedDrsID})

	pageZero, err := drs_support.ListDrsObjects(filesystem, 0, 1)
	if err != nil {
		t.Fatalf("list first page of DRS objects: %v", err)
	}

	if len(pageZero.Objects) > 1 {
		t.Fatalf("expected first page to contain at most 1 object, got %d", len(pageZero.Objects))
	}

	pageOne, err := drs_support.ListDrsObjects(filesystem, 1, 1)
	if err != nil {
		t.Fatalf("list second page of DRS objects: %v", err)
	}

	if pageZero.Total != pageOne.Total {
		t.Fatalf("expected consistent totals across pages, got %d and %d", pageZero.Total, pageOne.Total)
	}

	if pageZero.Total > 1 && len(pageOne.Objects) > 1 {
		t.Fatalf("expected second page to contain at most 1 object, got %d", len(pageOne.Objects))
	}

	pageAll, err := drs_support.ListDrsObjects(filesystem, 0, 1000)
	if err != nil {
		t.Fatalf("list larger page of DRS objects: %v", err)
	}

	assertContainsDrsObjectIDs(t, pageAll.Objects, []string{rootDrsID, nestedDrsID})

	if _, err := drs_support.GetDrsObjectByIRODSPath(filesystem, plainObjectPath); err == nil {
		t.Fatal("expected non-DRS object lookup by path to fail")
	}

	if err := drs_support.RemoveSingleDrsObjectFromDataObject(filesystem, rootObjectPath); err != nil {
		t.Fatalf("remove single DRS object metadata: %v", err)
	}

	if _, err := drs_support.GetDrsObjectByIRODSPath(filesystem, rootObjectPath); err == nil {
		t.Fatal("expected removed DRS object lookup by path to fail")
	}
}

func newIntegrationIRODSFilesystem(t *testing.T) *irodsfs.FileSystem {
	t.Helper()

	cfg := requireIntegrationIrodsConfig(t)
	account, err := irodstypes.CreateIRODSProxyAccount(
		cfg.IrodsHost,
		cfg.IrodsPort,
		cfg.IrodsPrimaryTestUser,
		cfg.IrodsZone,
		cfg.IrodsAdminUser,
		cfg.IrodsZone,
		irodstypes.GetAuthScheme(cfg.IrodsAuthScheme),
		cfg.IrodsAdminPassword,
		cfg.IrodsDefaultResource,
	)
	if err != nil {
		t.Fatalf("create iRODS proxy account: %v", err)
	}

	filesystem, err := irodsfs.NewFileSystemWithDefault(account, "irods-go-drs-integration-test")
	if err != nil {
		t.Fatalf("connect to iRODS. This test requires the docker compose stack in deployments/docker-test-framework/5-0 to be running: %v", err)
	}

	return filesystem
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

func makeIntegrationTestDir(t *testing.T, filesystem *irodsfs.FileSystem) string {
	t.Helper()

	cfg := requireIntegrationIrodsConfig(t)
	testDir := fmt.Sprintf("/%s/home/%s/drs-integration-%d", cfg.IrodsZone, cfg.IrodsPrimaryTestUser, time.Now().UnixNano())
	if err := filesystem.MakeDir(testDir, true); err != nil {
		t.Fatalf("make dir %q: %v", testDir, err)
	}

	t.Cleanup(func() {
		if err := filesystem.RemoveDir(testDir, true, true); err != nil && filesystem.Exists(testDir) {
			t.Errorf("cleanup dir %q: %v", testDir, err)
		}
	})

	return testDir
}

func assertDrsObjectIDs(t *testing.T, objects []*drs_support.InternalDrsObject, expectedIDs []string) {
	t.Helper()

	if len(objects) != len(expectedIDs) {
		t.Fatalf("expected %d DRS objects, got %d", len(expectedIDs), len(objects))
	}

	expected := map[string]bool{}
	for _, id := range expectedIDs {
		expected[id] = true
	}

	for _, object := range objects {
		if object == nil {
			t.Fatal("expected non-nil DRS objects")
		}

		if !expected[object.Id] {
			t.Fatalf("unexpected DRS id %q in result set", object.Id)
		}
	}
}

func assertContainsDrsObjectIDs(t *testing.T, objects []*drs_support.InternalDrsObject, expectedIDs []string) {
	t.Helper()

	seen := map[string]bool{}
	for _, object := range objects {
		if object != nil {
			seen[object.Id] = true
		}
	}

	for _, id := range expectedIDs {
		if !seen[id] {
			t.Fatalf("expected DRS id %q to be present in result set", id)
		}
	}
}
