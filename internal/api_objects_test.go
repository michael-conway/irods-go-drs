package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
		DrsConfig: &drs_support.DrsConfig{
			AccessMethods:       []string{"http", "irods", "local", "s3"},
			HTTPAccessBaseURL:   "https://download.example.org",
			IRODSAccessHost:     "irods.example.org",
			IRODSAccessPort:     1247,
			LocalAccessRootPath: "/mnt/irods",
		},
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

	if len(response.AccessMethods) != 4 {
		t.Fatalf("expected 4 access methods, got %+v", response.AccessMethods)
	}

	if response.AccessMethods[0].Type_ != "http" || response.AccessMethods[0].AccessId != "http:object-123" || response.AccessMethods[0].AccessUrl != nil {
		t.Fatalf("expected http access method, got %+v", response.AccessMethods[0])
	}

	if response.AccessMethods[1].Type_ != "irods" || response.AccessMethods[1].AccessId != "irods:object-123" || response.AccessMethods[1].AccessUrl != nil {
		t.Fatalf("expected irods access method, got %+v", response.AccessMethods[1])
	}

	if response.AccessMethods[2].Type_ != "local" || response.AccessMethods[2].AccessUrl == nil || response.AccessMethods[2].AccessUrl.Url != "local:///mnt/irods/tempZone/home/test1/file.txt" {
		t.Fatalf("expected local access method, got %+v", response.AccessMethods[2])
	}

	if response.AccessMethods[3].Type_ != "s3" || response.AccessMethods[3].AccessId != "s3:object-123" {
		t.Fatalf("expected s3 access method stub, got %+v", response.AccessMethods[3])
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

func TestOptionsObjectReturnsAuthorizations(t *testing.T) {
	oldConfigReader := readRouteDrsConfig
	readRouteDrsConfig = func() (*drs_support.DrsConfig, error) {
		return &drs_support.DrsConfig{
			OidcUrl:   "https://issuer.example.org",
			OidcRealm: "drs",
		}, nil
	}
	defer func() { readRouteDrsConfig = oldConfigReader }()

	oldFactory := createAdminRouteFileSystem
	createAdminRouteFileSystem = func(drsConfig *drs_support.DrsConfig, applicationName string) (RouteFileSystem, error) {
		return newRouteTestFileSystem(), nil
	}
	defer func() { createAdminRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodOptions, "/ga4gh/drs/v1/objects/object-123", nil)
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-123"})

	rec := httptest.NewRecorder()
	OptionsObject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response Authorizations
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if response.DrsObjectId != "object-123" {
		t.Fatalf("expected DRS object id object-123, got %q", response.DrsObjectId)
	}

	if len(response.SupportedTypes) != 2 || response.SupportedTypes[0] != "BasicAuth" || response.SupportedTypes[1] != "BearerAuth" {
		t.Fatalf("expected basic and bearer auth support, got %+v", response.SupportedTypes)
	}

	if len(response.BearerAuthIssuers) != 1 || response.BearerAuthIssuers[0] != "https://issuer.example.org/realms/drs" {
		t.Fatalf("expected configured bearer issuer, got %+v", response.BearerAuthIssuers)
	}
}

func TestOptionsObjectReturnsNotFound(t *testing.T) {
	oldConfigReader := readRouteDrsConfig
	readRouteDrsConfig = func() (*drs_support.DrsConfig, error) {
		return &drs_support.DrsConfig{}, nil
	}
	defer func() { readRouteDrsConfig = oldConfigReader }()

	oldFactory := createAdminRouteFileSystem
	createAdminRouteFileSystem = func(drsConfig *drs_support.DrsConfig, applicationName string) (RouteFileSystem, error) {
		return newRouteTestFileSystem(), nil
	}
	defer func() { createAdminRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodOptions, "/ga4gh/drs/v1/objects/missing", nil)
	req = mux.SetURLVars(req, map[string]string{"object_id": "missing"})

	rec := httptest.NewRecorder()
	OptionsObject(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestOptionsBulkObjectReturnsAuthorizations(t *testing.T) {
	oldConfigReader := readRouteDrsConfig
	readRouteDrsConfig = func() (*drs_support.DrsConfig, error) {
		return &drs_support.DrsConfig{
			OidcUrl:   "https://issuer.example.org",
			OidcRealm: "drs",
		}, nil
	}
	defer func() { readRouteDrsConfig = oldConfigReader }()

	oldFactory := createAdminRouteFileSystem
	createAdminRouteFileSystem = func(drsConfig *drs_support.DrsConfig, applicationName string) (RouteFileSystem, error) {
		return newRouteTestFileSystem(), nil
	}
	defer func() { createAdminRouteFileSystem = oldFactory }()

	body := strings.NewReader(`{"bulk_object_ids":["object-123","object-456"]}`)
	req := httptest.NewRequest(http.MethodOptions, "/ga4gh/drs/v1/objects", body)

	rec := httptest.NewRecorder()
	OptionsBulkObject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response InlineResponse2002
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if response.Summary == nil || response.Summary.Requested != 2 || response.Summary.Resolved != 2 || response.Summary.Unresolved != 0 {
		t.Fatalf("expected summary counts 2/2/0, got %+v", response.Summary)
	}

	if len(response.ResolvedDrsObject) != 2 {
		t.Fatalf("expected 2 resolved objects, got %+v", response.ResolvedDrsObject)
	}

	if response.ResolvedDrsObject[0].DrsObjectId != "object-123" || response.ResolvedDrsObject[1].DrsObjectId != "object-456" {
		t.Fatalf("expected resolved ids in request order, got %+v", response.ResolvedDrsObject)
	}

	if len(response.ResolvedDrsObject[0].BearerAuthIssuers) != 1 || response.ResolvedDrsObject[0].BearerAuthIssuers[0] != "https://issuer.example.org/realms/drs" {
		t.Fatalf("expected bearer issuer, got %+v", response.ResolvedDrsObject[0].BearerAuthIssuers)
	}
}

func TestOptionsBulkObjectReturnsUnresolvedForMissingIDs(t *testing.T) {
	oldConfigReader := readRouteDrsConfig
	readRouteDrsConfig = func() (*drs_support.DrsConfig, error) {
		return &drs_support.DrsConfig{}, nil
	}
	defer func() { readRouteDrsConfig = oldConfigReader }()

	oldFactory := createAdminRouteFileSystem
	createAdminRouteFileSystem = func(drsConfig *drs_support.DrsConfig, applicationName string) (RouteFileSystem, error) {
		return newRouteTestFileSystem(), nil
	}
	defer func() { createAdminRouteFileSystem = oldFactory }()

	body := strings.NewReader(`{"bulk_object_ids":["object-123","missing"]}`)
	req := httptest.NewRequest(http.MethodOptions, "/ga4gh/drs/v1/objects", body)

	rec := httptest.NewRecorder()
	OptionsBulkObject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response InlineResponse2002
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if response.Summary == nil || response.Summary.Requested != 2 || response.Summary.Resolved != 1 || response.Summary.Unresolved != 1 {
		t.Fatalf("expected summary counts 2/1/1, got %+v", response.Summary)
	}

	if len(response.ResolvedDrsObject) != 1 || response.ResolvedDrsObject[0].DrsObjectId != "object-123" {
		t.Fatalf("expected one resolved id, got %+v", response.ResolvedDrsObject)
	}

	if response.UnresolvedDrsObjects == nil || len(*response.UnresolvedDrsObjects) != 1 {
		t.Fatalf("expected unresolved block, got %+v", response.UnresolvedDrsObjects)
	}

	unresolved := (*response.UnresolvedDrsObjects)[0]
	if unresolved.ErrorCode != http.StatusNotFound || len(unresolved.ObjectIds) != 1 || unresolved.ObjectIds[0] != "missing" {
		t.Fatalf("expected unresolved 404 for missing id, got %+v", unresolved)
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

func TestBearerAuthIssuerFromConfig(t *testing.T) {
	issuer := bearerAuthIssuerFromConfig(&drs_support.DrsConfig{
		OidcUrl:   "https://issuer.example.org/",
		OidcRealm: "drs",
	})
	if issuer != "https://issuer.example.org/realms/drs" {
		t.Fatalf("expected issuer URL, got %q", issuer)
	}

	noRealmIssuer := bearerAuthIssuerFromConfig(&drs_support.DrsConfig{
		OidcUrl: "https://issuer.example.org/",
	})
	if noRealmIssuer != "https://issuer.example.org" {
		t.Fatalf("expected base OIDC URL without realm, got %q", noRealmIssuer)
	}
}

func TestNormalizedBulkObjectIDs(t *testing.T) {
	actual := normalizedBulkObjectIDs([]string{" object-123 ", "", "object-456"})
	if len(actual) != 2 || actual[0] != "object-123" || actual[1] != "object-456" {
		t.Fatalf("expected trimmed non-empty ids, got %+v", actual)
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
			"/tempZone/home/test1/file-2.txt": {
				ID:         2,
				Type:       irodsfs.FileEntry,
				Name:       "file-2.txt",
				Path:       "/tempZone/home/test1/file-2.txt",
				Size:       64,
				CreateTime: createTime,
				ModifyTime: updateTime,
				IRODSReplicas: []irodstypes.IRODSReplica{
					{
						Owner:      "test1",
						CreateTime: createTime,
						ModifyTime: updateTime,
						Checksum: &irodstypes.IRODSChecksum{
							Algorithm:           irodstypes.ChecksumAlgorithmSHA256,
							Checksum:            []byte("def456"),
							IRODSChecksumString: "sha2:def456",
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
			"/tempZone/home/test1/file-2.txt": {
				{Name: drs_support.DrsIdAvuAttrib, Value: "object-456", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuMimeTypeAttrib, Value: "text/plain", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuVersionAttrib, Value: "def456", Units: drs_support.DrsAvuUnit},
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
