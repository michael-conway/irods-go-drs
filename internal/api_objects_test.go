package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
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

	if len(response.Aliases) != 2 || response.Aliases[0] != "alias-1" || response.Aliases[1] != "alias-2" {
		t.Fatalf("expected aliases to be mapped, got %+v", response.Aliases)
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

func TestGetBulkObjectsReturnsBadRequestUntilImplemented(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/ga4gh/drs/v1/objects", strings.NewReader(`{"passports":["example"]}`))
	rec := httptest.NewRecorder()

	GetBulkObjects(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if !strings.Contains(response["message"], "issue #22") {
		t.Fatalf("expected issue reference in response, got %+v", response)
	}
}

func TestPostObjectReturnsBadRequestUntilImplemented(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/ga4gh/drs/v1/objects/object-123", strings.NewReader(`{"passports":["example"]}`))
	req = mux.SetURLVars(req, map[string]string{"object_id": "object-123"})
	rec := httptest.NewRecorder()

	PostObject(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if !strings.Contains(response["message"], "issue #22") {
		t.Fatalf("expected issue reference in response, got %+v", response)
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
			HttpsAccessImplementation:  "irods-https-api",
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
			HttpsAccessImplementation:  "irods-https-api",
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
			HttpsAccessImplementation:  "irods-https-api",
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

type routeCreatedTicket struct {
	name string
	path string
	typ  irodstypes.TicketType
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
