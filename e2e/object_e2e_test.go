//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

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
	if len(object.Aliases) != len(fixture.aliases) {
		t.Fatalf("expected %d aliases, got %d", len(fixture.aliases), len(object.Aliases))
	}
	for _, expectedAlias := range fixture.aliases {
		if !containsString(object.Aliases, expectedAlias) {
			t.Fatalf("expected alias %q in %+v", expectedAlias, object.Aliases)
		}
	}
	if !strings.Contains(object.SelfUri, "/"+fixture.objectID) {
		t.Fatalf("expected self uri to contain object id %q, got %q", fixture.objectID, object.SelfUri)
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

	downloadReq := newE2ERequest(t, http.MethodGet, accessURL.Url, nil)
	applyAccessURLHeaders(downloadReq, accessURL.Headers)

	downloadResp, err := client.Do(downloadReq)
	if err != nil {
		t.Fatalf("download via access url: %v", err)
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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}
