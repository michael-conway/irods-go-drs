//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"testing"
	"time"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
	"github.com/michael-conway/irods-go-drs/internal"
)

type e2eS3Fixture struct {
	rootPath             string
	bucketCollectionPath string
	objectPath           string
	objectID             string
	bucketName           string
	expectedAccessID     string
}

func TestResolveS3BucketContainingDrsObjectFromBucketAVUE2E(t *testing.T) {
	fixture, filesystem := setupE2ES3Fixture(t)
	defer filesystem.Release()

	cfg := requireE2EIRODSConfig(t)
	if !cfg.S3AccessMethodSupported {
		t.Skip("S3AccessMethodSupported is false in shared e2e config")
	}
	expectedPrefix := strings.TrimSpace(cfg.S3AccessMethodBaseURL)
	if expectedPrefix == "" {
		t.Fatalf("S3AccessMethodBaseURL must be set when S3AccessMethodSupported is true")
	}
	object, err := drs_support.GetDrsObjectByID(filesystem, fixture.objectID)
	if err != nil {
		t.Fatalf("get drs object by id %q: %v", fixture.objectID, err)
	}

	methods := drs_support.BuildAccessMethodsWithFilesystem(cfg, object, filesystem)
	var s3Method *drs_support.DrsAccessMethod
	for idx := range methods {
		if strings.EqualFold(strings.TrimSpace(methods[idx].Type), "s3") {
			s3Method = &methods[idx]
			break
		}
	}

	if s3Method == nil {
		t.Fatalf("expected s3 access method for drs object %q in bucket collection %q, got %+v", fixture.objectID, fixture.bucketCollectionPath, methods)
	}

	expectedURL := expectedPrefix
	if strings.HasSuffix(expectedURL, "://") || strings.HasSuffix(expectedURL, "/") {
		expectedURL += fixture.bucketName + "/subcoll/object.txt"
	} else {
		expectedURL += "/" + fixture.bucketName + "/subcoll/object.txt"
	}
	if s3Method.URL != expectedURL {
		t.Fatalf("expected s3 access url %q, got %+v", expectedURL, s3Method)
	}
	if s3Method.AccessID != fixture.expectedAccessID {
		t.Fatalf("expected s3 access id %q, got %+v", fixture.expectedAccessID, s3Method)
	}
}

func TestGetObjectReturnsS3AccessMethodFromBucketAVUE2E(t *testing.T) {
	cfg := requireE2EIRODSConfig(t)
	if !cfg.S3AccessMethodSupported {
		t.Skip("S3AccessMethodSupported is false in shared e2e config")
	}

	fixture, filesystem := setupE2ES3Fixture(t)
	defer filesystem.Release()

	baseURL := requireE2EBaseURL(t)
	username := requireE2EPrimaryTestUser(t)
	password := requireE2EPrimaryTestPassword(t)
	client := newE2EHTTPClient()

	req := newE2ERequest(t, http.MethodGet, getObjectURL(baseURL, fixture.objectID), nil)
	setBasicAuth(req, username, password)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get object with basic auth: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var object internal.DrsObject
	if err := json.NewDecoder(resp.Body).Decode(&object); err != nil {
		t.Fatalf("decode object response: %v", err)
	}

	s3Method := findAccessMethodByType(object.AccessMethods, "s3")
	if s3Method == nil {
		t.Fatalf("expected s3 access method in %+v", object.AccessMethods)
	}
	if s3Method.AccessUrl == nil || strings.TrimSpace(s3Method.AccessUrl.Url) == "" {
		t.Fatalf("expected inline s3 access_url, got %+v", s3Method)
	}
	if s3Method.AccessId != fixture.expectedAccessID {
		t.Fatalf("expected s3 access id %q, got %+v", fixture.expectedAccessID, s3Method)
	}

	expectedPrefix := strings.TrimSpace(cfg.S3AccessMethodBaseURL)
	if expectedPrefix == "" {
		t.Fatalf("S3AccessMethodBaseURL must be set when S3AccessMethodSupported is true")
	}
	expectedURL := expectedPrefix
	if strings.HasSuffix(expectedURL, "://") || strings.HasSuffix(expectedURL, "/") {
		expectedURL += fixture.bucketName + "/subcoll/object.txt"
	} else {
		expectedURL += "/" + fixture.bucketName + "/subcoll/object.txt"
	}
	if s3Method.AccessUrl.Url != expectedURL {
		t.Fatalf("expected s3 access_url %q, got %+v", expectedURL, s3Method)
	}
}

func TestGetAccessURLReturnsNotImplementedForS3AccessIDE2E(t *testing.T) {
	cfg := requireE2EIRODSConfig(t)
	if !cfg.S3AccessMethodSupported {
		t.Skip("S3AccessMethodSupported is false in shared e2e config")
	}

	fixture, filesystem := setupE2ES3Fixture(t)
	defer filesystem.Release()

	baseURL := requireE2EBaseURL(t)
	username := requireE2EPrimaryTestUser(t)
	password := requireE2EPrimaryTestPassword(t)
	client := newE2EHTTPClient()

	req := newE2ERequest(t, http.MethodGet, getObjectAccessURL(baseURL, fixture.objectID, fixture.expectedAccessID), nil)
	setBasicAuth(req, username, password)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get s3 access url by access_id with basic auth: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotImplemented {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 501 for s3 access_id resolution, got %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

func setupE2ES3Fixture(t *testing.T) (*e2eS3Fixture, *irodsfs.FileSystem) {
	t.Helper()

	cfg := requireE2EIRODSConfig(t)
	testUser := strings.TrimSpace(cfg.IrodsPrimaryTestUser)
	filesystem := newE2EIRODSFilesystem(t, testUser)

	rootPath := fmt.Sprintf("/%s/home/%s/drs-s3-e2e-%d", cfg.IrodsZone, testUser, time.Now().UnixNano())
	bucketCollectionPath := path.Join(rootPath, "drscoll")
	subCollectionPath := path.Join(bucketCollectionPath, "subcoll")
	objectPath := path.Join(subCollectionPath, "object.txt")
	bucketName := "drscol11"

	if err := filesystem.MakeDir(subCollectionPath, true); err != nil {
		filesystem.Release()
		t.Fatalf("create nested collection %q: %v", subCollectionPath, err)
	}

	if err := filesystem.AddMetadata(bucketCollectionPath, "iRODS:S3:Bucket", bucketName, ""); err != nil {
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
		if err := filesystem.RemoveDir(rootPath, true, true); err != nil && filesystem.Exists(rootPath) {
			t.Errorf("cleanup s3 e2e fixture root %q: %v", rootPath, err)
		}
	})

	return &e2eS3Fixture{
		rootPath:             rootPath,
		bucketCollectionPath: bucketCollectionPath,
		objectPath:           objectPath,
		objectID:             objectID,
		bucketName:           bucketName,
		expectedAccessID:     testUser,
	}, filesystem
}
