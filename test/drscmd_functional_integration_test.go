//go:build integration
// +build integration

package test

import (
	"bytes"
	"testing"

	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

func TestDrsCommandFunctionalFlowIntegration(t *testing.T) {
	filesystem := newIntegrationIRODSFilesystem(t)
	defer filesystem.Release()

	testDir := makeIntegrationTestDir(t, filesystem)

	objectPath := testDir + "/USERGUIDE.md"
	content := []byte("# drscmd integration fixture\n")
	if _, err := filesystem.UploadFileFromBuffer(bytes.NewBuffer(content), objectPath, "", false, true, nil); err != nil {
		t.Fatalf("upload object %q: %v", objectPath, err)
	}

	drsID, err := drs_support.CreateDrsObjectFromDataObject(
		filesystem,
		objectPath,
		"",
		"functional flow description",
		[]string{"functional-alias"},
	)
	if err != nil {
		t.Fatalf("create drs object: %v", err)
	}

	objectByID, err := drs_support.GetDrsObjectByID(filesystem, drsID)
	if err != nil {
		t.Fatalf("get drs object by id: %v", err)
	}

	if objectByID.AbsolutePath != objectPath {
		t.Fatalf("expected path %q, got %q", objectPath, objectByID.AbsolutePath)
	}

	if objectByID.Description != "functional flow description" {
		t.Fatalf("expected description to round-trip, got %q", objectByID.Description)
	}

	objectByPath, err := drs_support.GetDrsObjectByIRODSPath(filesystem, objectPath)
	if err != nil {
		t.Fatalf("get drs object by path: %v", err)
	}

	if objectByPath.Id != drsID {
		t.Fatalf("expected DRS id %q, got %q", drsID, objectByPath.Id)
	}

	if err := drs_support.RemoveSingleDrsObjectFromDataObject(filesystem, objectPath); err != nil {
		t.Fatalf("remove drs object metadata: %v", err)
	}

	if _, err := drs_support.GetDrsObjectByIRODSPath(filesystem, objectPath); err == nil {
		t.Fatal("expected removed DRS object lookup by path to fail")
	}
}
