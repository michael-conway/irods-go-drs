package internal

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"path"
	"strings"
	"testing"
	"time"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	"github.com/gorilla/mux"
	extension_irodsuri "github.com/michael-conway/go-irodsclient-extensions/irodsuri"
	extmetadata "github.com/michael-conway/go-irodsclient-extensions/metadata"
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
			IrodsAccessMethodSupported: false,
			FileAccessMethodSupported:  false,
			HttpsAccessMethodSupported: true,
			HttpsAccessImplementation:  "irods-go-rest",
			HttpsAccessMethodBaseURL:   "https://download.example.org/api/v1/path/contents?irods_path=",
			HttpsAccessUseTicket:       true,
			OidcUrl:                    "https://issuer.example.org",
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

	if response.Name != "/tempZone/home/test1/file.txt" {
		t.Fatalf("expected full iRODS path name, got %q", response.Name)
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

	if len(response.Aliases) != 3 {
		t.Fatalf("expected 3 aliases including irods uri alias, got %+v", response.Aliases)
	}
	if response.Aliases[0] != "alias-1" || response.Aliases[1] != "alias-2" {
		t.Fatalf("expected original aliases to be preserved first, got %+v", response.Aliases)
	}
	parsedAliasURI, err := extension_irodsuri.Parse(response.Aliases[2])
	if err != nil {
		t.Fatalf("expected valid iRODS uri alias, got %q: %v", response.Aliases[2], err)
	}
	if parsedAliasURI.UserInfo != nil {
		t.Fatalf("expected iRODS uri alias without user info, got %+v", parsedAliasURI.UserInfo)
	}
	if parsedAliasURI.Host != "icat-from-account.example.org" || parsedAliasURI.Port != 1247 || parsedAliasURI.Path != "/tempZone/home/test1/file.txt" {
		t.Fatalf("unexpected iRODS alias URI %+v", parsedAliasURI)
	}

	if len(response.AccessMethods) != 1 {
		t.Fatalf("expected 1 access method, got %+v", response.AccessMethods)
	}

	if response.AccessMethods[0].Type_ != "https" || response.AccessMethods[0].AccessId != "irods-go-rest-https" || response.AccessMethods[0].AccessUrl != nil || response.AccessMethods[0].Available {
		t.Fatalf("expected https access method, got %+v", response.AccessMethods[0])
	}
	if response.AccessMethods[0].Cloud != "irods:tempZone" {
		t.Fatalf("expected irods cloud name, got %+v", response.AccessMethods[0])
	}
	if response.AccessMethods[0].Region != "demoResc" {
		t.Fatalf("expected resource-backed region, got %+v", response.AccessMethods[0])
	}
	if response.AccessMethods[0].Authorizations == nil {
		t.Fatalf("expected access method authorizations, got %+v", response.AccessMethods[0])
	}
	if len(response.AccessMethods[0].Authorizations.SupportedTypes) != 2 || response.AccessMethods[0].Authorizations.SupportedTypes[0] != "BasicAuth" || response.AccessMethods[0].Authorizations.SupportedTypes[1] != "BearerAuth" {
		t.Fatalf("expected supported auth types, got %+v", response.AccessMethods[0].Authorizations)
	}
	if len(response.AccessMethods[0].Authorizations.BearerAuthIssuers) != 1 || response.AccessMethods[0].Authorizations.BearerAuthIssuers[0] != "https://issuer.example.org" {
		t.Fatalf("expected bearer auth issuer from oidc url, got %+v", response.AccessMethods[0].Authorizations)
	}
}

func TestGetObjectReturnsCompoundObjectWithOnlyHTTPSAccessMethod(t *testing.T) {
	oldFactory := createRouteFileSystem
	fs := newRouteTestFileSystem()
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return fs, nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-compound", nil)
	req.Host = "drs.example.org"
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			IrodsAccessMethodSupported: true,
			HttpsAccessMethodSupported: true,
			HttpsAccessImplementation:  "irods-go-rest",
			HttpsAccessMethodBaseURL:   "https://download.example.org/api/v1/path/contents?irods_path=",
			OidcUrl:                    "https://issuer.example.org",
			OidcRealm:                  "drs",
		},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-compound"})

	rec := httptest.NewRecorder()
	GetObject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response DrsObject
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if response.Id != "object-compound" {
		t.Fatalf("expected id object-compound, got %q", response.Id)
	}
	if len(response.AccessMethods) != 1 {
		t.Fatalf("expected one access method for compound object, got %+v", response.AccessMethods)
	}
	method := response.AccessMethods[0]
	if method.Type_ != "https" {
		t.Fatalf("expected compound https access method, got %+v", method)
	}
	if method.AccessId != "" {
		t.Fatalf("expected no access_id for compound https access method, got %+v", method)
	}
	if method.AccessUrl == nil {
		t.Fatalf("expected direct access_url for compound https access method, got %+v", method)
	}
	expectedURL := "http://drs.example.org/ga4gh/drs/v1/ext/compound/object-compound"
	if method.AccessUrl.Url != expectedURL {
		t.Fatalf("expected compound access_url %q, got %+v", expectedURL, method)
	}
	if method.Authorizations == nil || len(method.Authorizations.SupportedTypes) != 2 {
		t.Fatalf("expected basic/bearer authorizations, got %+v", method.Authorizations)
	}
	if method.Authorizations.SupportedTypes[0] != "BasicAuth" || method.Authorizations.SupportedTypes[1] != "BearerAuth" {
		t.Fatalf("expected supported auth types basic/bearer, got %+v", method.Authorizations)
	}

	manifest, err := drs_support.BuildCompoundRuntimeManifest(fs, "/tempZone/home/test1/compound")
	if err != nil {
		t.Fatalf("build expected compound manifest: %v", err)
	}
	manifestJSON, err := drs_support.MarshalCompoundRuntimeManifest(manifest)
	if err != nil {
		t.Fatalf("marshal expected compound manifest: %v", err)
	}
	sum := md5.Sum(manifestJSON)
	expectedChecksum := hex.EncodeToString(sum[:])
	if len(response.Checksums) != 1 || response.Checksums[0].Type_ != "md5" || response.Checksums[0].Checksum != expectedChecksum {
		t.Fatalf("expected generated manifest md5 checksum %q, got %+v", expectedChecksum, response.Checksums)
	}
}

func TestGetObjectReturnsHTTPSAccessMethodPerReplicaResource(t *testing.T) {
	oldFactory := createRouteFileSystem
	fs := newRouteTestFileSystem()
	entry := fs.entriesByPath["/tempZone/home/test1/file.txt"]
	entry.IRODSReplicas = append(entry.IRODSReplicas, irodstypes.IRODSReplica{
		Owner:        "test1",
		ResourceName: "archiveResc",
		CreateTime:   entry.CreateTime,
		ModifyTime:   entry.ModifyTime,
	})
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return fs, nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123", nil)
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			HttpsAccessMethodSupported: true,
			HttpsAccessImplementation:  "irods-go-rest",
			HttpsAccessMethodBaseURL:   "/api/v1/path/contents?irods_path=",
			HttpsResourceAffinity: []drs_support.ResourceAffinityEntry{
				{Host: "https://primary.example.org", Resources: []string{"demoResc"}},
				{Host: "https://archive.example.org", Resources: []string{"archiveResc"}},
			},
		},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-123"})

	rec := httptest.NewRecorder()
	GetObject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response DrsObject
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(response.AccessMethods) != 2 {
		t.Fatalf("expected 2 https access methods, got %+v", response.AccessMethods)
	}
	if response.AccessMethods[0].AccessId != "irods-go-rest-https-demoResc" || response.AccessMethods[0].Region != "demoResc" {
		t.Fatalf("expected demoResc access method first, got %+v", response.AccessMethods[0])
	}
	if response.AccessMethods[1].AccessId != "irods-go-rest-https-archiveResc" || response.AccessMethods[1].Region != "archiveResc" {
		t.Fatalf("expected archiveResc access method second, got %+v", response.AccessMethods[1])
	}
}

func TestGetObjectSkipsS3AccessMethodWithoutBucketAVU(t *testing.T) {
	oldFactory := createRouteFileSystem
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return newRouteTestFileSystem(), nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123", nil)
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			S3AccessMethodSupported: true,
			S3AccessMethodBaseURL:   "s3://",
		},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-123"})

	rec := httptest.NewRecorder()
	GetObject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response DrsObject
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(response.AccessMethods) != 0 {
		t.Fatalf("expected no s3 access method without iRODS:S3:Bucket AVU, got %+v", response.AccessMethods)
	}
}

func TestGetAccessURLReturnsIRODSGoRestAffinityHostAccessURL(t *testing.T) {
	oldFactory := createRouteFileSystem
	fs := newRouteTestFileSystem()
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return fs, nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	affinityAccessID := "irods-go-rest-https-demoResc"

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123/access/"+affinityAccessID, nil)
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			HttpsAccessMethodSupported: true,
			HttpsAccessImplementation:  "irods-go-rest",
			HttpsAccessMethodBaseURL:   "/api/v1/path/contents?irods_path=",
			HttpsResourceAffinity: []drs_support.ResourceAffinityEntry{
				{
					Host:      "https://dedicated.example.org",
					Resources: []string{"demoResc"},
				},
			},
		},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{
		"object_id": "object-123",
		"access_id": affinityAccessID,
	})

	rec := httptest.NewRecorder()
	GetAccessURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response AccessUrl
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	expectedURL := "https://dedicated.example.org/api/v1/path/contents?irods_path=" + neturl.QueryEscape("/tempZone/home/test1/file.txt")
	if response.Url != expectedURL {
		t.Fatalf("expected access url %q, got %q", expectedURL, response.Url)
	}
}

func TestGetAccessURLReturnsCompoundManifestExtURL(t *testing.T) {
	oldFactory := createRouteFileSystem
	fs := newRouteTestFileSystem()
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return fs, nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-compound/access/"+compoundManifestHTTPSAccessID, nil)
	req.Host = "drs.example.org"
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig:    &drs_support.DrsConfig{},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{
		"object_id": "object-compound",
		"access_id": compoundManifestHTTPSAccessID,
	})

	rec := httptest.NewRecorder()
	GetAccessURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response AccessUrl
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	expectedURL := "http://drs.example.org/ga4gh/drs/v1/ext/compound/object-compound"
	if response.Url != expectedURL {
		t.Fatalf("expected compound ext url %q, got %q", expectedURL, response.Url)
	}
	if len(response.Headers) != 0 {
		t.Fatalf("expected no headers for compound manifest ext url, got %+v", response.Headers)
	}
}

func TestGetCompoundManifestExtReturnsRuntimeManifest(t *testing.T) {
	oldFactory := createRouteFileSystem
	fs := newRouteTestFileSystem()
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return fs, nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/ext/compound/object-compound", nil)
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig:    &drs_support.DrsConfig{},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-compound"})

	rec := httptest.NewRecorder()
	GetCompoundManifestExt(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response drs_support.CompoundRuntimeManifest
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if response.RootPath != "/tempZone/home/test1/compound" || response.RootDrsID != "object-compound" {
		t.Fatalf("unexpected compound manifest root %+v", response)
	}
	if response.Manifest == nil {
		t.Fatalf("expected nested manifest payload")
	}
	if manifestContainsPathForInternalTests(response.Manifest, "/tempZone/home/test1/compound/.drsignore") {
		t.Fatalf("expected .drsignore to be excluded from runtime manifest")
	}
}

func TestGetObjectReturnsMappedDrsObjectWithIRODSAccessMethod(t *testing.T) {
	oldFactory := createRouteFileSystem
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return newRouteTestFileSystem(), nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123", nil)
	req.Host = "drs.example.org"
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			IrodsAccessMethodSupported: true,
			IRODSAccessHost:            "icat.example.org",
			IRODSAccessPort:            1247,
			OidcUrl:                    "https://issuer.example.org",
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

	if len(response.AccessMethods) != 1 {
		t.Fatalf("expected 1 access method, got %+v", response.AccessMethods)
	}

	if response.AccessMethods[0].Type_ != "irods" || response.AccessMethods[0].AccessId != "irods" || response.AccessMethods[0].AccessUrl != nil || response.AccessMethods[0].Available {
		t.Fatalf("expected irods access method, got %+v", response.AccessMethods[0])
	}
	if response.AccessMethods[0].Cloud != "irods:tempZone" {
		t.Fatalf("expected irods cloud name, got %+v", response.AccessMethods[0])
	}
	if response.AccessMethods[0].Region != "demoResc" {
		t.Fatalf("expected resource-backed region, got %+v", response.AccessMethods[0])
	}
	if response.AccessMethods[0].Authorizations != nil {
		t.Fatalf("expected no separate authorizations for embedded-ticket irods uri, got %+v", response.AccessMethods[0].Authorizations)
	}
}

func TestGetObjectReturnsBucketAVUMappedDrsObjectWithS3AccessMethod(t *testing.T) {
	oldFactory := createRouteFileSystem
	fs := newRouteTestFileSystem()
	fs.metadataByPath["/tempZone/home/test1"] = []*irodstypes.IRODSMeta{
		{Name: "iRODS:S3:Bucket", Value: "drscol11", Units: ""},
	}
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return fs, nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123", nil)
	req.Host = "drs.example.org"
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			S3AccessMethodSupported: true,
			S3AccessMethodBaseURL:   "s3://",
		},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-123"})

	rec := httptest.NewRecorder()
	GetObject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response DrsObject
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(response.AccessMethods) != 1 {
		t.Fatalf("expected one s3 access method, got %+v", response.AccessMethods)
	}

	method := response.AccessMethods[0]
	if method.Type_ != "s3" {
		t.Fatalf("expected s3 access method type, got %+v", method)
	}
	if method.AccessId != "test1" {
		t.Fatalf("expected s3 access_id %q, got %+v", "test1", method)
	}
	if method.AccessUrl == nil || method.AccessUrl.Url != "s3://drscol11/file.txt" {
		t.Fatalf("expected s3 access_url %q, got %+v", "s3://drscol11/file.txt", method)
	}
	if method.Cloud != "irods:tempZone" {
		t.Fatalf("expected irods cloud name, got %+v", method)
	}
	if method.Region != "demoResc" {
		t.Fatalf("expected resource-backed region, got %+v", method)
	}
	if !method.Available {
		t.Fatalf("expected s3 method to be available, got %+v", method)
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

func TestGetObjectReturnsUnauthorizedForBasicIRODSAuthFailure(t *testing.T) {
	oldFactory := createRouteFileSystem
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return nil, irodstypes.NewAuthError(account)
	}
	defer func() { createRouteFileSystem = oldFactory }()

	account := &irodstypes.IRODSAccount{ClientUser: "test1", ClientZone: "tempZone"}
	ctx := context.WithValue(context.Background(), authSchemeContextKey, "basic")
	ctx = context.WithValue(ctx, drsServiceContextKey, &DrsServiceContext{
		DrsConfig:    &drs_support.DrsConfig{},
		IrodsAccount: account,
	})
	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123", nil)
	req = req.WithContext(ctx)
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-123"})

	rec := httptest.NewRecorder()
	GetObject(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for Basic iRODS auth failure, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetObjectReturnsServerErrorForBearerIRODSAuthFailure(t *testing.T) {
	oldFactory := createRouteFileSystem
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return nil, irodstypes.NewAuthError(account)
	}
	defer func() { createRouteFileSystem = oldFactory }()

	account := &irodstypes.IRODSAccount{ClientUser: "test1", ClientZone: "tempZone"}
	ctx := context.WithValue(context.Background(), authSchemeContextKey, "bearer")
	ctx = context.WithValue(ctx, drsServiceContextKey, &DrsServiceContext{
		DrsConfig:    &drs_support.DrsConfig{},
		IrodsAccount: account,
	})
	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123", nil)
	req = req.WithContext(ctx)
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-123"})

	rec := httptest.NewRecorder()
	GetObject(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for bearer-backed iRODS auth failure, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetObjectReturnsUnauthorizedForBasicIRODSAuthFailureDuringLookup(t *testing.T) {
	account := &irodstypes.IRODSAccount{ClientUser: "test1", ClientZone: "tempZone"}
	oldFactory := createRouteFileSystem
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return &queryErrorRouteFileSystem{
			routeTestFileSystem: newRouteTestFileSystem(),
			err:                 irodstypes.NewAuthError(account),
		}, nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	ctx := context.WithValue(context.Background(), authSchemeContextKey, "basic")
	ctx = context.WithValue(ctx, drsServiceContextKey, &DrsServiceContext{
		DrsConfig:    &drs_support.DrsConfig{},
		IrodsAccount: account,
	})
	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123", nil)
	req = req.WithContext(ctx)
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-123"})

	rec := httptest.NewRecorder()
	GetObject(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for Basic iRODS auth failure during lookup, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetBulkObjectsReturnsNotImplemented(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/ga4gh/drs/v1/objects", strings.NewReader(`{"passports":["example"]}`))
	rec := httptest.NewRecorder()

	GetBulkObjects(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if !strings.Contains(response["message"], "not supported in this deployment") {
		t.Fatalf("expected unsupported operation message, got %+v", response)
	}
}

func TestPostObjectReturnsNotImplemented(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/ga4gh/drs/v1/objects/object-123", strings.NewReader(`{"passports":["example"]}`))
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-123"})
	rec := httptest.NewRecorder()

	PostObject(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if !strings.Contains(response["message"], "not supported in this deployment") {
		t.Fatalf("expected unsupported operation message, got %+v", response)
	}
}

func TestPostAccessURLReturnsNotImplemented(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/ga4gh/drs/v1/objects/object-123/access/some-id", strings.NewReader(`{"passports":["example"]}`))
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-123", "access_id": "some-id"})
	rec := httptest.NewRecorder()

	PostAccessURL(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !strings.Contains(response["message"], "not supported in this deployment") {
		t.Fatalf("expected unsupported operation message, got %+v", response)
	}
}

func TestGetBulkAccessURLReturnsNotImplemented(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/ga4gh/drs/v1/objects/access", strings.NewReader(`{"bulk_object_access_ids":[]}`))
	rec := httptest.NewRecorder()

	GetBulkAccessURL(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !strings.Contains(response["message"], "not supported in this deployment") {
		t.Fatalf("expected unsupported operation message, got %+v", response)
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

func TestOptionsObjectReturnsNotImplementedForUnsupportedHTTPSProvider(t *testing.T) {
	oldConfigReader := readRouteDrsConfig
	readRouteDrsConfig = func() (*drs_support.DrsConfig, error) {
		return &drs_support.DrsConfig{
			HttpsAccessMethodSupported: true,
			HttpsAccessImplementation:  "unsupported-provider",
		}, nil
	}
	defer func() { readRouteDrsConfig = oldConfigReader }()

	req := httptest.NewRequest(http.MethodOptions, "/ga4gh/drs/v1/objects/object-123", nil)
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-123"})

	rec := httptest.NewRecorder()
	OptionsObject(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rec.Code)
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

func TestOptionsBulkObjectReturnsNotImplementedForUnsupportedHTTPSProvider(t *testing.T) {
	oldConfigReader := readRouteDrsConfig
	readRouteDrsConfig = func() (*drs_support.DrsConfig, error) {
		return &drs_support.DrsConfig{
			HttpsAccessMethodSupported: true,
			HttpsAccessImplementation:  "unsupported-provider",
		}, nil
	}
	defer func() { readRouteDrsConfig = oldConfigReader }()

	body := strings.NewReader(`{"bulk_object_ids":["object-123","object-456"]}`)
	req := httptest.NewRequest(http.MethodOptions, "/ga4gh/drs/v1/objects", body)

	rec := httptest.NewRecorder()
	OptionsBulkObject(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rec.Code)
	}
}

func TestGetAccessURLReturnsIRODSGoRestTicketAccessURL(t *testing.T) {
	oldFactory := createRouteFileSystem
	fs := newRouteTestFileSystem()
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return fs, nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123/access/irods-go-rest-https", nil)
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			HttpsAccessMethodSupported:   true,
			HttpsAccessImplementation:    "irods-go-rest",
			HttpsAccessMethodBaseURL:     "https://rest.example.org/api/v1/path/contents?irods_path=",
			HttpsAccessUseTicket:         true,
			DefaultTicketLifetimeMinutes: 30,
			DefaultTicketUseLimit:        5,
		},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{
		"object_id": "object-123",
		"access_id": "irods-go-rest-https",
	})

	rec := httptest.NewRecorder()
	GetAccessURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response AccessUrl
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	expectedURL := "https://rest.example.org/api/v1/path/contents?irods_path=" + neturl.QueryEscape("/tempZone/home/test1/file.txt")
	if response.Url != expectedURL {
		t.Fatalf("expected access url %q, got %q", expectedURL, response.Url)
	}
	if len(response.Headers) != 1 || !strings.HasPrefix(response.Headers[0], "Authorization: Bearer irods-ticket:ticket_") {
		t.Fatalf("expected bearer ticket authorization header, got %+v", response.Headers)
	}
	if len(fs.createdTickets) != 1 {
		t.Fatalf("expected one created ticket, got %+v", fs.createdTickets)
	}
	if fs.createdTickets[0].path != "/tempZone/home/test1/file.txt" {
		t.Fatalf("expected ticket path to match object path, got %+v", fs.createdTickets[0])
	}
	if fs.ticketUseLimits[fs.createdTickets[0].name] != 5 {
		t.Fatalf("expected ticket use limit 5, got %+v", fs.ticketUseLimits)
	}
	if fs.ticketExpiryTimes[fs.createdTickets[0].name].IsZero() {
		t.Fatalf("expected ticket expiry time to be set, got %+v", fs.ticketExpiryTimes)
	}
}

func TestGetAccessURLDefaultIDUsesPrimaryResourceAffinityHost(t *testing.T) {
	oldFactory := createRouteFileSystem
	fs := newRouteTestFileSystem()
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return fs, nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123/access/irods-go-rest-https", nil)
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			HttpsAccessMethodSupported: true,
			HttpsAccessImplementation:  "irods-go-rest",
			HttpsAccessMethodBaseURL:   "/api/v1/path/contents?irods_path=",
			HttpsResourceAffinity: []drs_support.ResourceAffinityEntry{
				{
					Host:      "https://default.example.org",
					Resources: []string{},
				},
			},
		},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{
		"object_id": "object-123",
		"access_id": "irods-go-rest-https",
	})

	rec := httptest.NewRecorder()
	GetAccessURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response AccessUrl
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	expectedURL := "https://default.example.org/api/v1/path/contents?irods_path=" + neturl.QueryEscape("/tempZone/home/test1/file.txt")
	if response.Url != expectedURL {
		t.Fatalf("expected access url %q, got %q", expectedURL, response.Url)
	}
}

func TestGetAccessURLReturnsDirectURLWhenTicketDisabled(t *testing.T) {
	oldFactory := createRouteFileSystem
	fs := newRouteTestFileSystem()
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return fs, nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123/access/irods-go-rest-https", nil)
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			HttpsAccessMethodSupported: true,
			HttpsAccessImplementation:  "irods-go-rest",
			HttpsAccessMethodBaseURL:   "https://rest.example.org/api/v1/path/contents?irods_path=",
		},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{
		"object_id": "object-123",
		"access_id": "irods-go-rest-https",
	})

	rec := httptest.NewRecorder()
	GetAccessURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response AccessUrl
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(response.Headers) != 0 {
		t.Fatalf("expected no headers when ticket support is disabled, got %+v", response.Headers)
	}
	if len(fs.createdTickets) != 0 {
		t.Fatalf("expected no created tickets, got %+v", fs.createdTickets)
	}
}

func TestGetAccessURLReturnsIRODSHTTPSAPIAccessURL(t *testing.T) {
	oldFactory := createRouteFileSystem
	fs := newRouteTestFileSystem()
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return fs, nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123/access/irods-https-api-https", nil)
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			HttpsAccessMethodSupported: true,
			HttpsAccessImplementation:  "irods-https-api",
			HttpsAccessMethodBaseURL:   "https://https-api.example.org/download?path=",
		},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{
		"object_id": "object-123",
		"access_id": "irods-https-api-https",
	})

	rec := httptest.NewRecorder()
	GetAccessURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response AccessUrl
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	expectedURL := "https://https-api.example.org/download?path=" + neturl.QueryEscape("/tempZone/home/test1/file.txt")
	if response.Url != expectedURL {
		t.Fatalf("expected access url %q, got %q", expectedURL, response.Url)
	}
	if len(response.Headers) != 0 {
		t.Fatalf("expected no headers when ticket support is disabled, got %+v", response.Headers)
	}
}

func TestGetAccessURLReturnsIRODSTicketURI(t *testing.T) {
	oldFactory := createRouteFileSystem
	fs := newRouteTestFileSystem()
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return fs, nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123/access/irods", nil)
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			IrodsAccessMethodSupported:   true,
			IRODSAccessHost:              "icat.example.org",
			IRODSAccessPort:              1247,
			IrodsZone:                    "tempZone",
			DefaultTicketLifetimeMinutes: 30,
			DefaultTicketUseLimit:        5,
		},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{
		"object_id": "object-123",
		"access_id": "irods",
	})

	rec := httptest.NewRecorder()
	GetAccessURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response AccessUrl
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	parsed, err := extension_irodsuri.Parse(response.Url)
	if err != nil {
		t.Fatalf("expected valid irods URI, got %q: %v", response.Url, err)
	}

	if parsed.Host != "icat.example.org" || parsed.Port != 1247 {
		t.Fatalf("expected iRODS host/port, got %+v", parsed)
	}
	if parsed.Path != "/tempZone/home/test1/file.txt" {
		t.Fatalf("expected irods path, got %+v", parsed)
	}
	if parsed.UserInfo == nil || parsed.UserInfo.UserName != "anonymous" || parsed.UserInfo.Zone != "tempZone" {
		t.Fatalf("expected anonymous tempZone user info, got %+v", parsed.UserInfo)
	}
	if !strings.HasPrefix(parsed.Ticket, "ticket_") {
		t.Fatalf("expected ticket query parameter, got %+v", parsed)
	}
	if len(response.Headers) != 0 {
		t.Fatalf("expected no headers for irods URI access, got %+v", response.Headers)
	}
	if len(fs.createdTickets) != 1 {
		t.Fatalf("expected one created ticket, got %+v", fs.createdTickets)
	}
	if fs.createdTickets[0].path != "/tempZone/home/test1/file.txt" {
		t.Fatalf("expected ticket path to match object path, got %+v", fs.createdTickets[0])
	}
	if fs.createdTickets[0].name != parsed.Ticket {
		t.Fatalf("expected URI ticket to match created ticket, got ticket %q and created %+v", parsed.Ticket, fs.createdTickets[0])
	}
	if fs.ticketUseLimits[fs.createdTickets[0].name] != 5 {
		t.Fatalf("expected ticket use limit 5, got %+v", fs.ticketUseLimits)
	}
	if fs.ticketExpiryTimes[fs.createdTickets[0].name].IsZero() {
		t.Fatalf("expected ticket expiry time to be set, got %+v", fs.ticketExpiryTimes)
	}
}

func TestGetAccessURLReturnsIRODSTicketURIFallbackFromFilesystemAccount(t *testing.T) {
	oldFactory := createRouteFileSystem
	fs := newRouteTestFileSystem()
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return fs, nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123/access/irods", nil)
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			IrodsAccessMethodSupported:   true,
			IrodsZone:                    "tempZone",
			DefaultTicketLifetimeMinutes: 30,
			DefaultTicketUseLimit:        5,
		},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{
		"object_id": "object-123",
		"access_id": "irods",
	})

	rec := httptest.NewRecorder()
	GetAccessURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response AccessUrl
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	parsed, err := extension_irodsuri.Parse(response.Url)
	if err != nil {
		t.Fatalf("expected valid irods URI, got %q: %v", response.Url, err)
	}

	if parsed.Host != "icat-from-account.example.org" || parsed.Port != 1247 {
		t.Fatalf("expected fallback host/port from filesystem account, got %+v", parsed)
	}
}

func TestGetAccessURLReturnsNotFoundForUnknownAccessID(t *testing.T) {
	oldFactory := createRouteFileSystem
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return newRouteTestFileSystem(), nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123/access/unknown", nil)
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			HttpsAccessMethodSupported: true,
			HttpsAccessImplementation:  "irods-go-rest",
			HttpsAccessMethodBaseURL:   "https://rest.example.org/api/v1/path/contents?irods_path=",
			HttpsAccessUseTicket:       true,
		},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{
		"object_id": "object-123",
		"access_id": "unknown",
	})

	rec := httptest.NewRecorder()
	GetAccessURL(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGetAccessURLReturnsNotImplementedForS3AccessID(t *testing.T) {
	oldFactory := createRouteFileSystem
	fs := newRouteTestFileSystem()
	fs.metadataByPath["/tempZone/home/test1"] = []*irodstypes.IRODSMeta{
		{Name: "iRODS:S3:Bucket", Value: "drscol11", Units: ""},
	}
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return fs, nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123/access/test1", nil)
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			S3AccessMethodSupported: true,
			S3AccessMethodBaseURL:   "s3://",
		},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{
		"object_id": "object-123",
		"access_id": "test1",
	})

	rec := httptest.NewRecorder()
	GetAccessURL(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 for unsupported s3 access_id resolution, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !strings.Contains(response["message"], "not supported by /access in this deployment") {
		t.Fatalf("expected explicit unsupported-access-method message, got %+v", response)
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

	response := drsObjectFromInternal(req, object, true, nil)
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

func TestGetObjectReturnsNotImplementedForUnsupportedHTTPSImplementation(t *testing.T) {
	oldFactory := createRouteFileSystem
	createRouteFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (RouteFileSystem, error) {
		return newRouteTestFileSystem(), nil
	}
	defer func() { createRouteFileSystem = oldFactory }()

	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/objects/object-123", nil)
	req = req.WithContext(context.WithValue(context.Background(), drsServiceContextKey, &DrsServiceContext{
		DrsConfig: &drs_support.DrsConfig{
			HttpsAccessMethodSupported: true,
			HttpsAccessImplementation:  "unsupported-provider",
			HttpsAccessMethodBaseURL:   "https://download.example.org/api/v1/path/contents?irods_path=",
		},
		IrodsAccount: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
	}))
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-123"})

	rec := httptest.NewRecorder()
	GetObject(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rec.Code)
	}
}

type routeTestFileSystem struct {
	account           *irodstypes.IRODSAccount
	entriesByPath     map[string]*irodsfs.Entry
	metadataByPath    map[string][]*irodstypes.IRODSMeta
	createdTickets    []routeCreatedTicket
	ticketUseLimits   map[string]int64
	ticketExpiryTimes map[string]time.Time
}

type queryErrorRouteFileSystem struct {
	*routeTestFileSystem
	err error
}

func (f *queryErrorRouteFileSystem) QueryMetadataEntries(query extmetadata.EntryQuery) (extmetadata.EntryQueryResult, error) {
	return extmetadata.EntryQueryResult{}, f.err
}

type routeCreatedTicket struct {
	name string
	path string
	typ  irodstypes.TicketType
}

func newRouteTestFileSystem() *routeTestFileSystem {
	createTime := time.Unix(1000, 0).UTC()
	updateTime := time.Unix(2000, 0).UTC()

	return &routeTestFileSystem{
		account: &irodstypes.IRODSAccount{
			Host:       "icat-from-account.example.org",
			Port:       1247,
			ClientZone: "tempZone",
		},
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
						Owner:        "test1",
						ResourceName: "demoResc",
						CreateTime:   createTime,
						ModifyTime:   updateTime,
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
						Owner:        "test1",
						ResourceName: "demoResc",
						CreateTime:   createTime,
						ModifyTime:   updateTime,
						Checksum: &irodstypes.IRODSChecksum{
							Algorithm:           irodstypes.ChecksumAlgorithmSHA256,
							Checksum:            []byte("def456"),
							IRODSChecksumString: "sha2:def456",
						},
					},
				},
			},
			"/tempZone/home/test1/compound": {
				ID:         100,
				Type:       irodsfs.DirectoryEntry,
				Name:       "compound",
				Path:       "/tempZone/home/test1/compound",
				CreateTime: createTime,
				ModifyTime: updateTime,
			},
			"/tempZone/home/test1/compound/child.txt": {
				ID:         101,
				Type:       irodsfs.FileEntry,
				Name:       "child.txt",
				Path:       "/tempZone/home/test1/compound/child.txt",
				Size:       32,
				CreateTime: createTime,
				ModifyTime: updateTime,
				IRODSReplicas: []irodstypes.IRODSReplica{
					{
						Owner:        "test1",
						ResourceName: "demoResc",
						CreateTime:   createTime,
						ModifyTime:   updateTime,
					},
				},
			},
			"/tempZone/home/test1/compound/.drsignore": {
				ID:         102,
				Type:       irodsfs.FileEntry,
				Name:       ".drsignore",
				Path:       "/tempZone/home/test1/compound/.drsignore",
				Size:       8,
				CreateTime: createTime,
				ModifyTime: updateTime,
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
			"/tempZone/home/test1/compound": {
				{Name: drs_support.DrsIdAvuAttrib, Value: "object-compound", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuCompoundManifestAttrib, Value: "true", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuAliasAttrib, Value: ".", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuDescriptionAttrib, Value: "compound root", Units: drs_support.DrsAvuUnit},
			},
			"/tempZone/home/test1/compound/child.txt": {
				{Name: drs_support.DrsIdAvuAttrib, Value: "object-compound-child", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuAliasAttrib, Value: "child.txt", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuDescriptionAttrib, Value: "child data object", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuMimeTypeAttrib, Value: "text/plain", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuVersionAttrib, Value: "child-version", Units: drs_support.DrsAvuUnit},
			},
			"/tempZone/home/test1/compound/.drsignore": {},
		},
		ticketUseLimits:   map[string]int64{},
		ticketExpiryTimes: map[string]time.Time{},
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
	irodsPath = strings.TrimSuffix(irodsPath, "/")
	if irodsPath == "" {
		irodsPath = "/"
	}

	results := []*irodsfs.Entry{}
	for candidatePath, entry := range f.entriesByPath {
		if entry == nil || candidatePath == irodsPath {
			continue
		}
		parent := path.Dir(candidatePath)
		if parent == "." {
			parent = "/"
		}
		if parent == irodsPath {
			results = append(results, entry)
		}
	}
	return results, nil
}

func (f *routeTestFileSystem) QueryMetadataEntries(query extmetadata.EntryQuery) (extmetadata.EntryQueryResult, error) {
	normalized, err := extmetadata.NormalizeEntryQuery(query)
	if err != nil {
		return extmetadata.EntryQueryResult{}, err
	}

	result := extmetadata.EntryQueryResult{
		Entries:     []*extmetadata.Entry{},
		MatchedAVUs: map[string][]extmetadata.AVUStat{},
		Page: extmetadata.EntryQueryPage{
			Limit: normalized.Limit,
		},
	}

	for irodsPath, metas := range f.metadataByPath {
		entry := f.entriesByPath[irodsPath]
		if entry == nil || !routeQueryIncludesEntryKind(normalized, entry) || !routeQueryEntryInScope(normalized, entry) {
			continue
		}

		matched := routeMatchedAVUsForQuery(normalized, metas)
		if len(matched) == 0 {
			continue
		}

		result.Entries = append(result.Entries, entry)
		result.MatchedAVUs[entry.Path] = matched
		if entry.IsDir() {
			result.Page.Returned.Collections++
			continue
		}
		result.Page.Returned.DataObjects++
	}

	return result, nil
}

func routeQueryIncludesEntryKind(query extmetadata.EntryQuery, entry *irodsfs.Entry) bool {
	if entry.IsDir() {
		return extmetadata.EntryQueryHasKind(query, extmetadata.EntryKindCollection)
	}
	return extmetadata.EntryQueryHasKind(query, extmetadata.EntryKindDataObject)
}

func routeMatchedAVUsForQuery(query extmetadata.EntryQuery, metas []*irodstypes.IRODSMeta) []extmetadata.AVUStat {
	matched := []extmetadata.AVUStat{}
	for _, meta := range metas {
		if meta == nil || !routeMetaMatchesConditions(meta, query.Conditions) {
			continue
		}
		matched = append(matched, extmetadata.AVUStat{
			Name:  meta.Name,
			Value: meta.Value,
			Units: meta.Units,
		})
	}
	return matched
}

func routeQueryEntryInScope(query extmetadata.EntryQuery, entry *irodsfs.Entry) bool {
	if query.Scope == nil || query.Scope.Mode == extmetadata.EntryQueryScopeAbsolute {
		return true
	}

	root := strings.TrimRight(query.Scope.Root, "/")
	if root == "" {
		root = "/"
	}
	entryPath := path.Clean(entry.Path)

	switch query.Scope.Mode {
	case extmetadata.EntryQueryScopeSelf:
		return entry.IsDir() && entryPath == root
	case extmetadata.EntryQueryScopeChildren:
		return path.Dir(entryPath) == root
	case extmetadata.EntryQueryScopeDescendants:
		if entry.IsDir() {
			return strings.HasPrefix(entryPath, root+"/")
		}
		return strings.HasPrefix(path.Dir(entryPath), root+"/")
	default:
		return true
	}
}

func routeMetaMatchesConditions(meta *irodstypes.IRODSMeta, conditions []extmetadata.EntryCondition) bool {
	for _, condition := range conditions {
		var candidate string
		switch condition.Field {
		case extmetadata.FieldAVUAttrib:
			candidate = meta.Name
		case extmetadata.FieldAVUValue:
			candidate = meta.Value
		case extmetadata.FieldAVUUnit:
			candidate = meta.Units
		default:
			continue
		}

		if condition.Op == extmetadata.OpLike {
			pattern := strings.TrimSuffix(extmetadata.NormalizeLikePattern(condition.Value), "%")
			if !strings.HasPrefix(candidate, pattern) {
				return false
			}
			continue
		}
		if candidate != condition.Value {
			return false
		}
	}
	return true
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

func (f *routeTestFileSystem) CreateTicket(ticketName string, ticketType irodstypes.TicketType, path string) error {
	f.createdTickets = append(f.createdTickets, routeCreatedTicket{name: ticketName, path: path, typ: ticketType})
	return nil
}

func (f *routeTestFileSystem) ModifyTicketUseLimit(ticketName string, uses int64) error {
	f.ticketUseLimits[ticketName] = uses
	return nil
}

func (f *routeTestFileSystem) ModifyTicketExpirationTime(ticketName string, expirationTime time.Time) error {
	f.ticketExpiryTimes[ticketName] = expirationTime
	return nil
}

func (f *routeTestFileSystem) DeleteTicket(ticketName string) error {
	delete(f.ticketUseLimits, ticketName)
	delete(f.ticketExpiryTimes, ticketName)
	return nil
}

func manifestContainsPathForInternalTests(node *drs_support.CompoundManifestNode, targetPath string) bool {
	if node == nil {
		return false
	}
	if node.Path == targetPath {
		return true
	}
	for _, child := range node.Children {
		childCopy := child
		if manifestContainsPathForInternalTests(&childCopy, targetPath) {
			return true
		}
	}
	return false
}
