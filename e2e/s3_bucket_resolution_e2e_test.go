//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"testing"
	"time"

	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

func TestResolveS3BucketContainingDrsObjectE2E(t *testing.T) {
	cfg := requireE2EIRODSConfig(t)
	testUser := strings.TrimSpace(cfg.IrodsPrimaryTestUser)
	filesystem := newE2EIRODSFilesystem(t, testUser)

	rootPath := fmt.Sprintf("/%s/home/%s/drs-s3-e2e-%d", cfg.IrodsZone, testUser, time.Now().UnixNano())
	bucketCollectionPath := path.Join(rootPath, "drscoll")
	subCollectionPath := path.Join(bucketCollectionPath, "subcoll")
	objectPath := path.Join(subCollectionPath, "object.txt")

	if err := filesystem.MakeDir(subCollectionPath, true); err != nil {
		filesystem.Release()
		t.Fatalf("create nested collection %q: %v", subCollectionPath, err)
	}

	if err := filesystem.AddMetadata(bucketCollectionPath, "iRODS:S3:Bucket", "drscol11", ""); err != nil {
		filesystem.Release()
		t.Fatalf("add s3 bucket AVU on %q: %v", bucketCollectionPath, err)
	}

	if _, err := filesystem.UploadFileFromBuffer(bytes.NewBuffer([]byte("s3 e2e object\n")), objectPath, "", false, true, nil); err != nil {
		filesystem.Release()
		t.Fatalf("upload object %q: %v", objectPath, err)
	}

	objectID, err := drs_support.CreateDrsObjectFromDataObject(filesystem, objectPath, "", "s3 e2e description", []string{"s3-e2e-alias"})
	if err != nil {
		filesystem.Release()
		t.Fatalf("create drs object %q: %v", objectPath, err)
	}

	t.Cleanup(func() {
		defer filesystem.Release()
		if err := filesystem.RemoveDir(rootPath, true, true); err != nil && filesystem.Exists(rootPath) {
			t.Errorf("cleanup s3 e2e fixture root %q: %v", rootPath, err)
		}
	})

	object, err := drs_support.GetDrsObjectByID(filesystem, objectID)
	if err != nil {
		t.Fatalf("get drs object by id %q: %v", objectID, err)
	}

	accessConfig := *cfg
	accessConfig.S3AccessMethodSupported = true
	if strings.TrimSpace(accessConfig.S3AccessMethodBaseURL) == "" {
		accessConfig.S3AccessMethodBaseURL = "s3://"
	}

	methods := drs_support.BuildAccessMethodsWithFilesystem(&accessConfig, object, filesystem)
	var s3Method *drs_support.DrsAccessMethod
	for idx := range methods {
		if strings.EqualFold(strings.TrimSpace(methods[idx].Type), "s3") {
			s3Method = &methods[idx]
			break
		}
	}

	if s3Method == nil {
		t.Fatalf("expected s3 access method for drs object %q in bucket collection %q, got %+v", objectID, bucketCollectionPath, methods)
	}

	expectedURL := "s3://drscol11/subcoll/object.txt"
	if s3Method.URL != expectedURL {
		t.Fatalf("expected s3 access url %q, got %+v", expectedURL, s3Method)
	}
}

