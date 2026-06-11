//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"testing"

	extension_irodsuri "github.com/michael-conway/go-irodsclient-extensions/irodsuri"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
	"github.com/michael-conway/irods-go-drs/internal"
)

func TestGetObjectRequiresAuthenticationE2E(t *testing.T) {
	baseURL := requireE2EBaseURL(t)
	fixture := requireE2EObjectFixture(t)
	client := newE2EHTTPClient()

	req := newE2ERequest(t, http.MethodGet, getObjectURL(baseURL, fixture.objectID), nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get object without auth: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", resp.StatusCode)
	}
}

func TestGetObjectBearerAuthE2E(t *testing.T) {
	baseURL := requireE2EBaseURL(t)
	token := requireE2EBearerToken(t)
	fixture := requireE2EObjectFixture(t)
	client := newE2EHTTPClient()

	req := newE2ERequest(t, http.MethodGet, getObjectURL(baseURL, fixture.objectID), nil)
	setBearerAuth(req, token)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get object with bearer auth: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json; charset=UTF-8" {
		t.Fatalf("expected json content type, got %q", contentType)
	}

	var object internal.DrsObject
	if err := json.NewDecoder(resp.Body).Decode(&object); err != nil {
		t.Fatalf("decode object response: %v", err)
	}

	if object.Id != fixture.objectID {
		t.Fatalf("expected DRS id %q, got %q", fixture.objectID, object.Id)
	}
	if object.Name != fixture.objectName {
		t.Fatalf("expected object name %q, got %q", fixture.objectName, object.Name)
	}
	if object.Description != fixture.description {
		t.Fatalf("expected description %q, got %q", fixture.description, object.Description)
	}
	if object.MimeType != drs_support.DeriveMimeTypeFromDataObjectPath(fixture.objectPath) {
		t.Fatalf("expected mime type %q, got %q", drs_support.DeriveMimeTypeFromDataObjectPath(fixture.objectPath), object.MimeType)
	}
	if object.Version == "" {
		t.Fatal("expected non-empty object version")
	}
	if object.Size <= 0 {
		t.Fatalf("expected positive object size, got %d", object.Size)
	}
	if len(object.Checksums) == 0 || strings.TrimSpace(object.Checksums[0].Checksum) == "" {
		t.Fatalf("expected object checksum in response, got %+v", object.Checksums)
	}
	if len(object.Aliases) != len(fixture.aliases)+1 {
		t.Fatalf("expected %d aliases including iRODS uri alias, got %d", len(fixture.aliases)+1, len(object.Aliases))
	}
	for _, expectedAlias := range fixture.aliases {
		if !containsString(object.Aliases, expectedAlias) {
			t.Fatalf("expected alias %q in %+v", expectedAlias, object.Aliases)
		}
	}
	irodsAlias := ""
	for _, alias := range object.Aliases {
		parsedAlias, err := extension_irodsuri.Parse(alias)
		if err != nil {
			continue
		}
		irodsAlias = alias
		if parsedAlias.UserInfo != nil {
			t.Fatalf("expected iRODS alias without user info, got %+v", parsedAlias.UserInfo)
		}
		if parsedAlias.Path != fixture.objectPath {
			t.Fatalf("expected iRODS alias path %q, got %+v", fixture.objectPath, parsedAlias)
		}
		break
	}
	if strings.TrimSpace(irodsAlias) == "" {
		t.Fatalf("expected iRODS uri alias in %+v", object.Aliases)
	}
	if !strings.Contains(object.SelfUri, "/"+fixture.objectID) {
		t.Fatalf("expected self uri to contain object id %q, got %q", fixture.objectID, object.SelfUri)
	}

	irodsMethod := findAccessMethodByType(object.AccessMethods, "irods")
	if irodsMethod == nil {
		t.Fatalf("expected irods access method in %+v", object.AccessMethods)
	}
	if irodsMethod.AccessId != "irods" {
		t.Fatalf("expected irods access id %q, got %+v", "irods", irodsMethod)
	}
	if irodsMethod.AccessUrl != nil {
		t.Fatalf("expected deferred irods access method without inline url, got %+v", irodsMethod)
	}
}

func TestGetObjectBasicAuthE2E(t *testing.T) {
	baseURL := requireE2EBaseURL(t)
	username := requireE2EPrimaryTestUser(t)
	password := requireE2EPrimaryTestPassword(t)
	fixture := requireE2EBasicObjectFixture(t)
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

	if object.Id != fixture.objectID {
		t.Fatalf("expected DRS id %q, got %q", fixture.objectID, object.Id)
	}
	if object.Name != fixture.objectName {
		t.Fatalf("expected object name %q, got %q", fixture.objectName, object.Name)
	}
}

func TestGetAccessURLBasicAuthE2E(t *testing.T) {
	baseURL := requireE2EBaseURL(t)
	username := requireE2EPrimaryTestUser(t)
	password := requireE2EPrimaryTestPassword(t)
	fixture := requireE2EBasicObjectFixture(t)
	client := newE2EHTTPClient()

	req := newE2ERequest(t, http.MethodGet, getObjectAccessURL(baseURL, fixture.objectID, "irods-go-rest-https"), nil)
	setBasicAuth(req, username, password)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get access url with basic auth: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var accessURL internal.AccessUrl
	if err := json.NewDecoder(resp.Body).Decode(&accessURL); err != nil {
		t.Fatalf("decode access url response: %v", err)
	}

	if strings.TrimSpace(accessURL.Url) == "" {
		t.Fatal("expected non-empty access url")
	}
	if len(accessURL.Headers) != 1 || !strings.HasPrefix(accessURL.Headers[0], "Authorization: Bearer irods-ticket:") {
		t.Fatalf("expected irods ticket bearer header, got %+v", accessURL.Headers)
	}

	resolvedAccessURL, err := resolveE2EURL(baseURL, accessURL.Url)
	if err != nil {
		t.Fatalf("resolve access url %q against base %q: %v", accessURL.Url, baseURL, err)
	}
	downloadReq := newE2ERequest(t, http.MethodGet, resolvedAccessURL, nil)
	applyAccessURLHeaders(downloadReq, accessURL.Headers)

	downloadResp, err := client.Do(downloadReq)
	if err != nil {
		fallbackURL, canFallback, fallbackErr := localhostToIPv4URL(resolvedAccessURL)
		if fallbackErr != nil {
			t.Fatalf("build IPv4 fallback URL for %q: %v", resolvedAccessURL, fallbackErr)
		}
		if isConnectionRefusedError(err) && canFallback {
			downloadReq = newE2ERequest(t, http.MethodGet, fallbackURL, nil)
			applyAccessURLHeaders(downloadReq, accessURL.Headers)
			downloadResp, err = client.Do(downloadReq)
			if err != nil {
				t.Fatalf("download via access url failed after localhost IPv4 fallback (%s -> %s): %v", resolvedAccessURL, fallbackURL, err)
			}
		} else {
			t.Fatalf("download via access url: %v", err)
		}
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(downloadResp.Body)
		t.Fatalf("expected 200 from download url, got %d: %s", downloadResp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(downloadResp.Body)
	if err != nil {
		t.Fatalf("read download body: %v", err)
	}

	if string(body) != "drs basic e2e object\n" {
		t.Fatalf("expected downloaded content %q, got %q", "drs basic e2e object\n", string(body))
	}
}

func TestGetIRODSAccessURLBasicAuthE2E(t *testing.T) {
	baseURL := requireE2EBaseURL(t)
	username := requireE2EPrimaryTestUser(t)
	password := requireE2EPrimaryTestPassword(t)
	fixture := requireE2EBasicObjectFixture(t)
	client := newE2EHTTPClient()

	req := newE2ERequest(t, http.MethodGet, getObjectAccessURL(baseURL, fixture.objectID, "irods"), nil)
	setBasicAuth(req, username, password)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get irods access url with basic auth: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var accessURL internal.AccessUrl
	if err := json.NewDecoder(resp.Body).Decode(&accessURL); err != nil {
		t.Fatalf("decode access url response: %v", err)
	}

	if len(accessURL.Headers) != 0 {
		t.Fatalf("expected no headers for irods uri access, got %+v", accessURL.Headers)
	}

	parsed, err := extension_irodsuri.Parse(accessURL.Url)
	if err != nil {
		t.Fatalf("parse returned irods uri %q: %v", accessURL.Url, err)
	}

	cfg := requireE2EIRODSConfig(t)
	expectedHost := strings.TrimSpace(cfg.IRODSAccessHost)
	if expectedHost == "" {
		expectedHost = strings.TrimSpace(cfg.IrodsHost)
	}
	expectedPort := cfg.IRODSAccessPort
	if expectedPort == 0 {
		expectedPort = cfg.IrodsPort
	}
	if parsed.Host != expectedHost || parsed.Port != expectedPort {
		t.Fatalf("expected iRODS access host/port %s:%d, got %+v", expectedHost, expectedPort, parsed)
	}
	if parsed.Path != fixture.objectPath {
		t.Fatalf("expected irods uri path %q, got %+v", fixture.objectPath, parsed)
	}
	if parsed.UserInfo == nil || parsed.UserInfo.UserName != "anonymous" || parsed.UserInfo.Zone != strings.TrimSpace(cfg.IrodsZone) {
		t.Fatalf("expected anonymous irods user info for zone %q, got %+v", strings.TrimSpace(cfg.IrodsZone), parsed.UserInfo)
	}
	if !strings.HasPrefix(strings.TrimSpace(parsed.Ticket), "ticket_") {
		t.Fatalf("expected ticket query parameter, got %+v", parsed)
	}
}

func TestGetCompoundObjectBasicAuthE2E(t *testing.T) {
	baseURL := requireE2EBaseURL(t)
	username := requireE2EPrimaryTestUser(t)
	password := requireE2EPrimaryTestPassword(t)
	fixture := requireE2ECompoundObjectFixture(t)
	client := newE2EHTTPClient()

	req := newE2ERequest(t, http.MethodGet, getObjectURL(baseURL, fixture.compoundID), nil)
	setBasicAuth(req, username, password)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get compound object with basic auth: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var object internal.DrsObject
	if err := json.NewDecoder(resp.Body).Decode(&object); err != nil {
		t.Fatalf("decode compound object response: %v", err)
	}

	if object.Id != fixture.compoundID {
		t.Fatalf("expected compound DRS id %q, got %q", fixture.compoundID, object.Id)
	}
	if object.Name != fixture.compoundPath {
		t.Fatalf("expected compound object path %q, got %q", fixture.compoundPath, object.Name)
	}
	if len(object.AccessMethods) != 1 {
		t.Fatalf("expected exactly one compound access method, got %+v", object.AccessMethods)
	}

	method := object.AccessMethods[0]
	if method.Type_ != "https" {
		t.Fatalf("expected https compound access method, got %+v", method)
	}
	if strings.TrimSpace(method.AccessId) != "" {
		t.Fatalf("expected compound access method to omit access_id, got %+v", method)
	}
	if method.AccessUrl == nil || strings.TrimSpace(method.AccessUrl.Url) == "" {
		t.Fatalf("expected direct compound access_url, got %+v", method)
	}

	expectedExtURL := getCompoundExtURL(baseURL, fixture.compoundID)
	if method.AccessUrl.Url != expectedExtURL {
		t.Fatalf("expected compound access_url %q, got %+v", expectedExtURL, method)
	}

	manifestReq := newE2ERequest(t, http.MethodGet, method.AccessUrl.Url, nil)
	setBasicAuth(manifestReq, username, password)

	manifestResp, err := client.Do(manifestReq)
	if err != nil {
		t.Fatalf("get compound manifest via direct access_url: %v", err)
	}
	defer manifestResp.Body.Close()

	if manifestResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(manifestResp.Body)
		t.Fatalf("expected 200 from compound manifest url, got %d: %s", manifestResp.StatusCode, strings.TrimSpace(string(body)))
	}

	var manifest drs_support.CompoundRuntimeManifest
	if err := json.NewDecoder(manifestResp.Body).Decode(&manifest); err != nil {
		t.Fatalf("decode compound manifest response: %v", err)
	}

	if manifest.RootDrsID != fixture.compoundID {
		t.Fatalf("expected compound root drs id %q, got %+v", fixture.compoundID, manifest)
	}
	if manifest.RootPath != fixture.compoundPath {
		t.Fatalf("expected compound root path %q, got %+v", fixture.compoundPath, manifest)
	}
	if manifest.Manifest == nil {
		t.Fatalf("expected nested compound manifest payload")
	}
	if !manifestContainsPathE2E(manifest.Manifest, fixture.childObjectPath) {
		t.Fatalf("expected manifest to include child object path %q, got %+v", fixture.childObjectPath, manifest.Manifest)
	}
	// Runtime manifest membership is based on persisted DRS IDs. Ignored
	// paths are excluded during compound creation and therefore should not
	// appear in the runtime manifest.
	if manifestContainsPathE2E(manifest.Manifest, fixture.ignoredPath) {
		t.Fatalf("expected runtime manifest to exclude ignored path %q, got %+v", fixture.ignoredPath, manifest.Manifest)
	}
	if manifestContainsPathE2E(manifest.Manifest, fixture.ignoreFilePath) {
		t.Fatalf("expected manifest to exclude ignore file path %q, got %+v", fixture.ignoreFilePath, manifest.Manifest)
	}

	setupFS := newE2EIRODSFilesystem(t, fixture.expectedUser)
	defer setupFS.Release()

	if _, err := drs_support.GetDrsObjectByIRODSPath(setupFS, fixture.ignoredPath); err == nil {
		t.Fatalf("expected ignored object %q not to be a DRS object", fixture.ignoredPath)
	}

	stripResult, err := drs_support.StripDrsSemantics(setupFS, fixture.compoundPath)
	if err != nil {
		t.Fatalf("strip DRS semantics for compound path %q: %v", fixture.compoundPath, err)
	}
	if stripResult == nil {
		t.Fatalf("expected strip result for compound path %q", fixture.compoundPath)
	}
	if stripResult.PathsWithDrsMetadata == 0 || stripResult.AvusRemoved == 0 {
		t.Fatalf("expected strip to remove DRS metadata, got %+v", stripResult)
	}

	reqAfterStrip := newE2ERequest(t, http.MethodGet, getObjectURL(baseURL, fixture.compoundID), nil)
	setBasicAuth(reqAfterStrip, username, password)
	respAfterStrip, err := client.Do(reqAfterStrip)
	if err != nil {
		t.Fatalf("get stripped compound object with basic auth: %v", err)
	}
	defer respAfterStrip.Body.Close()
	if respAfterStrip.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(respAfterStrip.Body)
		t.Fatalf("expected 404 for stripped compound object, got %d: %s", respAfterStrip.StatusCode, strings.TrimSpace(string(body)))
	}

	if _, err := drs_support.GetDrsObjectByIRODSPath(setupFS, fixture.childObjectPath); err == nil {
		t.Fatalf("expected included object %q not to be a DRS object after strip", fixture.childObjectPath)
	}
}

func TestGetMissingObjectBearerAuthE2E(t *testing.T) {
	baseURL := requireE2EBaseURL(t)
	token := requireE2EBearerToken(t)
	fixture := requireE2EObjectFixture(t)
	client := newE2EHTTPClient()

	req := newE2ERequest(t, http.MethodGet, getObjectURL(baseURL, fixture.missingID), nil)
	setBearerAuth(req, token)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get missing object with bearer auth: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

func TestOptionsObjectE2E(t *testing.T) {
	baseURL := requireE2EBaseURL(t)
	fixture := requireE2EObjectFixture(t)
	client := newE2EHTTPClient()

	req := newE2ERequest(t, http.MethodOptions, getObjectURL(baseURL, fixture.objectID), nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("options object request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json; charset=UTF-8" {
		t.Fatalf("expected json content type, got %q", contentType)
	}

	var response internal.Authorizations
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode options response: %v", err)
	}

	if response.DrsObjectId != fixture.objectID {
		t.Fatalf("expected DRS id %q, got %q", fixture.objectID, response.DrsObjectId)
	}

	if len(response.SupportedTypes) != 2 || response.SupportedTypes[0] != "BasicAuth" || response.SupportedTypes[1] != "BearerAuth" {
		t.Fatalf("expected basic and bearer support, got %+v", response.SupportedTypes)
	}

	cfg := requireE2EIRODSConfig(t)
	expectedIssuer := strings.TrimRight(cfg.OidcUrl, "/")
	if strings.TrimSpace(cfg.OidcRealm) != "" {
		expectedIssuer += "/realms/" + strings.TrimSpace(cfg.OidcRealm)
	}

	if expectedIssuer != "" {
		if len(response.BearerAuthIssuers) != 1 || response.BearerAuthIssuers[0] != expectedIssuer {
			t.Fatalf("expected bearer issuer %q, got %+v", expectedIssuer, response.BearerAuthIssuers)
		}
	}
}

func TestOptionsBulkObjectE2E(t *testing.T) {
	baseURL := requireE2EBaseURL(t)
	fixture := requireE2EBulkObjectFixture(t)
	client := newE2EHTTPClient()

	requestBody, err := json.Marshal(internal.BulkObjectIdNoPassport{BulkObjectIds: fixture.objectIDs})
	if err != nil {
		t.Fatalf("marshal bulk request body: %v", err)
	}

	req := newE2ERequest(t, http.MethodOptions, strings.TrimRight(baseURL, "/")+"/ga4gh/drs/v1/objects", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("bulk options request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json; charset=UTF-8" {
		t.Fatalf("expected json content type, got %q", contentType)
	}

	var response internal.InlineResponse2002
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode bulk options response: %v", err)
	}

	if response.Summary == nil || response.Summary.Requested != int32(len(fixture.objectIDs)) || response.Summary.Resolved != int32(len(fixture.objectIDs)) || response.Summary.Unresolved != 0 {
		t.Fatalf("expected summary counts %d/%d/0, got %+v", len(fixture.objectIDs), len(fixture.objectIDs), response.Summary)
	}

	if len(response.ResolvedDrsObject) != len(fixture.objectIDs) {
		t.Fatalf("expected %d resolved ids, got %+v", len(fixture.objectIDs), response.ResolvedDrsObject)
	}

	for i, objectID := range fixture.objectIDs {
		if response.ResolvedDrsObject[i].DrsObjectId != objectID {
			t.Fatalf("expected resolved id %q at index %d, got %+v", objectID, i, response.ResolvedDrsObject)
		}
	}

	cfg := requireE2EIRODSConfig(t)
	expectedIssuer := strings.TrimRight(cfg.OidcUrl, "/")
	if strings.TrimSpace(cfg.OidcRealm) != "" {
		expectedIssuer += "/realms/" + strings.TrimSpace(cfg.OidcRealm)
	}

	if expectedIssuer != "" {
		for _, authorization := range response.ResolvedDrsObject {
			if len(authorization.BearerAuthIssuers) != 1 || authorization.BearerAuthIssuers[0] != expectedIssuer {
				t.Fatalf("expected bearer issuer %q, got %+v", expectedIssuer, authorization.BearerAuthIssuers)
			}
		}
	}
}

func getObjectURL(baseURL string, objectID string) string {
	return strings.TrimRight(baseURL, "/") + "/ga4gh/drs/v1/objects/" + url.PathEscape(objectID)
}

func getObjectAccessURL(baseURL string, objectID string, accessID string) string {
	return getObjectURL(baseURL, objectID) + "/access/" + url.PathEscape(accessID)
}

func getCompoundExtURL(baseURL string, objectID string) string {
	return strings.TrimRight(baseURL, "/") + "/ga4gh/drs/v1/ext/compound/" + url.PathEscape(objectID)
}

func resolveE2EURL(baseURL string, candidate string) (string, error) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return "", nil
	}

	parsedCandidate, err := url.Parse(candidate)
	if err != nil {
		return "", err
	}
	if parsedCandidate.IsAbs() {
		return parsedCandidate.String(), nil
	}

	parsedBase, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", err
	}

	return parsedBase.ResolveReference(parsedCandidate).String(), nil
}

func applyAccessURLHeaders(req *http.Request, headers []string) {
	for _, header := range headers {
		name, value, found := strings.Cut(header, ":")
		if !found {
			continue
		}

		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if name == "" || value == "" {
			continue
		}

		req.Header.Add(name, value)
	}
}

func findAccessMethodByType(methods []internal.AccessMethod, methodType string) *internal.AccessMethod {
	for idx := range methods {
		if strings.EqualFold(strings.TrimSpace(methods[idx].Type_), strings.TrimSpace(methodType)) {
			return &methods[idx]
		}
	}

	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}

func manifestContainsPathE2E(node *drs_support.CompoundManifestNode, targetPath string) bool {
	if node == nil {
		return false
	}

	if strings.TrimSpace(node.Path) == strings.TrimSpace(targetPath) {
		return true
	}

	for idx := range node.Children {
		child := &node.Children[idx]
		if manifestContainsPathE2E(child, targetPath) {
			return true
		}
	}

	return false
}

func isConnectionRefusedError(err error) bool {
	if err == nil {
		return false
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.ECONNREFUSED) {
			return true
		}
	}

	return strings.Contains(strings.ToLower(err.Error()), "connection refused")
}

func localhostToIPv4URL(rawURL string) (string, bool, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", false, nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", false, err
	}

	host := strings.TrimSpace(parsed.Hostname())
	if !strings.EqualFold(host, "localhost") {
		return "", false, nil
	}

	port := strings.TrimSpace(parsed.Port())
	if port != "" {
		parsed.Host = "127.0.0.1:" + port
	} else {
		parsed.Host = "127.0.0.1"
	}

	return parsed.String(), true, nil
}
