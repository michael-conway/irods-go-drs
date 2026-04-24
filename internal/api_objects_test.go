package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	"github.com/gorilla/mux"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

func TestGetObjectReturnsMappedDrsObject(t *testing.T) {
	oldFactory := createRouteFileSystem
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return newRouteTestFileSystem(), nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123", nil)
	req.Host = "drs.example.org"
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-123"})

	rec := httptest.NewRecorder()
	GetObject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response DrsObject
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if response.Id != "object-123" {
		t.Fatalf("expected id object-123, got %q", response.Id)
	}

	if response.Name != "file.txt" {
		t.Fatalf("expected name file.txt, got %q", response.Name)
	}

	if response.SelfUri != "drs://drs.example.org/object-123" {
		t.Fatalf("expected self_uri to be built from request host, got %q", response.SelfUri)
	}

	if response.Description != "test description" {
		t.Fatalf("expected description to be mapped, got %q", response.Description)
	}

	if len(response.Checksums) != 1 || response.Checksums[0].Checksum != "abc123" {
		t.Fatalf("expected checksum to be mapped, got %+v", response.Checksums)
	}

	if response.Checksums[0].Type_ != "sha-256" {
		t.Fatalf("expected checksum type sha-256, got %+v", response.Checksums)
	}

	if len(response.Aliases) != 2 || response.Aliases[0] != "alias-1" || response.Aliases[1] != "alias-2" {
		t.Fatalf("expected aliases to be mapped, got %+v", response.Aliases)
	}
}

func TestGetObjectReturnsNotFound(t *testing.T) {
	oldFactory := createRouteFileSystem
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return newRouteTestFileSystem(), nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/missing", nil)
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{"object_id": "missing"})

	rec := httptest.NewRecorder()
	GetObject(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDrsObjectFromInternalIncludesExpandedContents(t *testing.T) {
	object := &drs_support.InternalDrsObject{
		Id:           "bundle-1",
		AbsolutePath: "/tempZone/home/test1/bundle.json",
		Size:         42,
		CreatedTime:  time.Unix(1000, 0).UTC(),
		UpdatedTime:  time.Unix(2000, 0).UTC(),
		Contents: []drs_support.DrsManifestEntry{
			{ID: "child-1", Name: "bam"},
			{ID: "child-2", Name: "bai"},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/bundle-1?expand=true", nil)
	req.Host = "drs.example.org"

	response := drsObjectFromInternal(req, object, true)
	if len(response.Contents) != 2 {
		t.Fatalf("expected 2 contents entries, got %d", len(response.Contents))
	}

	if response.Contents[0].Id != "child-1" || response.Contents[0].Name != "bam" {
		t.Fatalf("expected first contents entry to be mapped, got %+v", response.Contents[0])
	}

	if len(response.Contents[0].DrsUri) != 1 || response.Contents[0].DrsUri[0] != "drs://drs.example.org/child-1" {
		t.Fatalf("expected child drs_uri to be built, got %+v", response.Contents[0].DrsUri)
	}
}

type routeTestFileSystem struct {
	account        *irodstypes.IRODSAccount
	entriesByPath  map[string]*irodsfs.Entry
	metadataByPath map[string][]*irodstypes.IRODSMeta
}

func newRouteTestFileSystem() *routeTestFileSystem {
	createTime := time.Unix(1000, 0).UTC()
	updateTime := time.Unix(2000, 0).UTC()

	return &routeTestFileSystem{
		account: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
		entriesByPath: map[string]*irodsfs.Entry{
			"/tempZone/home/test1/file.txt": {
				ID:         1,
				Type:       irodsfs.FileEntry,
				Name:       "file.txt",
				Path:       "/tempZone/home/test1/file.txt",
				Size:       128,
				CreateTime: createTime,
				ModifyTime: updateTime,
				IRODSReplicas: []irodstypes.IRODSReplica{
					{
						Owner:      "test1",
						CreateTime: createTime,
						ModifyTime: updateTime,
						Checksum: &irodstypes.IRODSChecksum{
							Algorithm:           irodstypes.ChecksumAlgorithmSHA256,
							Checksum:            []byte("abc123"),
							IRODSChecksumString: "sha2:abc123",
						},
					},
				},
			},
		},
		metadataByPath: map[string][]*irodstypes.IRODSMeta{
			"/tempZone/home/test1/file.txt": {
				{Name: drs_support.DrsIdAvuAttrib, Value: "object-123", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuMimeTypeAttrib, Value: "text/plain", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuVersionAttrib, Value: "abc123", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuDescriptionAttrib, Value: "test description", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuAliasAttrib, Value: "alias-1", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuAliasAttrib, Value: "alias-2", Units: drs_support.DrsAvuUnit},
			},
		},
	}
}

func (f *routeTestFileSystem) Release() {}

func (f *routeTestFileSystem) StatFile(irodsPath string) (*irodsfs.Entry, error) {
	if entry, ok := f.entriesByPath[irodsPath]; ok {
		return entry, nil
	}
	return nil, http.ErrMissingFile
}

func (f *routeTestFileSystem) List(irodsPath string) ([]*irodsfs.Entry, error) {
	return []*irodsfs.Entry{}, nil
}

func (f *routeTestFileSystem) SearchByMeta(name string, value string) ([]*irodsfs.Entry, error) {
	results := []*irodsfs.Entry{}
	for path, metas := range f.metadataByPath {
		for _, meta := range metas {
			if meta != nil && meta.Name == name && meta.Value == value && meta.Units == drs_support.DrsAvuUnit {
				results = append(results, f.entriesByPath[path])
				break
			}
		}
	}
	return results, nil
}

func (f *routeTestFileSystem) ListMetadata(irodsPath string) ([]*irodstypes.IRODSMeta, error) {
	if metas, ok := f.metadataByPath[irodsPath]; ok {
		return append([]*irodstypes.IRODSMeta(nil), metas...), nil
	}
	return []*irodstypes.IRODSMeta{}, nil
}

func (f *routeTestFileSystem) AddMetadata(irodsPath string, attName string, attValue string, attUnits string) error {
	return nil
}

func (f *routeTestFileSystem) DeleteMetadataByAVU(irodsPath string, attName string, attValue string, attUnits string) error {
	return nil
}

func (f *routeTestFileSystem) GetAccount() *irodstypes.IRODSAccount {
	return f.account
}
