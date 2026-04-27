//go:build integration
// +build integration

package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
	"github.com/michael-conway/irods-go-drs/internal"
)

func TestOptionsObjectIntegration(t *testing.T) {
	filesystem := newIntegrationIRODSFilesystem(t)
	defer filesystem.Release()

	testDir := makeIntegrationTestDir(t, filesystem)
	objectPath := testDir + "/options-object.txt"
	content := []byte("drs options integration object\n")
	if _, err := filesystem.UploadFileFromBuffer(bytes.NewBuffer(content), objectPath, "", false, true, nil); err != nil {
		t.Fatalf("upload object %q: %v", objectPath, err)
	}

	drsID, err := drs_support.CreateDrsObjectFromDataObject(
		filesystem,
		objectPath,
		"",
		"options integration description",
		[]string{"options-alias"},
	)
	if err != nil {
		t.Fatalf("create drs object: %v", err)
	}

	server := httptest.NewServer(internal.NewRouter())
	defer server.Close()

	req, err := http.NewRequest(http.MethodOptions, server.URL+"/ga4gh/drs/v1/objects/"+drsID, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send options request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var response internal.Authorizations
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.DrsObjectId != drsID {
		t.Fatalf("expected DRS id %q, got %q", drsID, response.DrsObjectId)
	}

	if len(response.SupportedTypes) != 2 || response.SupportedTypes[0] != "BasicAuth" || response.SupportedTypes[1] != "BearerAuth" {
		t.Fatalf("expected basic and bearer support, got %+v", response.SupportedTypes)
	}

	cfg := requireIntegrationDrsConfig(t)
	expectedIssuer := strings.TrimRight(strings.TrimSpace(cfg.OidcUrl), "/")
	if realm := strings.TrimSpace(cfg.OidcRealm); realm != "" {
		expectedIssuer += "/realms/" + realm
	}

	if expectedIssuer != "" {
		if len(response.BearerAuthIssuers) != 1 || response.BearerAuthIssuers[0] != expectedIssuer {
			t.Fatalf("expected bearer issuer %q, got %+v", expectedIssuer, response.BearerAuthIssuers)
		}
	}
}

func TestOptionsBulkObjectIntegration(t *testing.T) {
	filesystem := newIntegrationIRODSFilesystem(t)
	defer filesystem.Release()

	testDir := makeIntegrationTestDir(t, filesystem)
	objectPaths := []string{
		testDir + "/options-bulk-1.txt",
		testDir + "/options-bulk-2.txt",
		testDir + "/options-bulk-3.txt",
	}

	objectIDs := make([]string, 0, len(objectPaths))
	for i, objectPath := range objectPaths {
		content := []byte("drs bulk options integration object\n")
		if _, err := filesystem.UploadFileFromBuffer(bytes.NewBuffer(content), objectPath, "", false, true, nil); err != nil {
			t.Fatalf("upload object %q: %v", objectPath, err)
		}

		drsID, err := drs_support.CreateDrsObjectFromDataObject(
			filesystem,
			objectPath,
			"",
			"options bulk integration description",
			[]string{objectPath},
		)
		if err != nil {
			t.Fatalf("create drs object %d: %v", i, err)
		}
		objectIDs = append(objectIDs, drsID)
	}

	server := httptest.NewServer(internal.NewRouter())
	defer server.Close()

	requestBody, err := json.Marshal(internal.BulkObjectIdNoPassport{BulkObjectIds: objectIDs})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodOptions, server.URL+"/ga4gh/drs/v1/objects", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send bulk options request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var response internal.InlineResponse2002
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Summary == nil || response.Summary.Requested != int32(len(objectIDs)) || response.Summary.Resolved != int32(len(objectIDs)) || response.Summary.Unresolved != 0 {
		t.Fatalf("expected summary counts %d/%d/0, got %+v", len(objectIDs), len(objectIDs), response.Summary)
	}

	if len(response.ResolvedDrsObject) != len(objectIDs) {
		t.Fatalf("expected %d resolved objects, got %+v", len(objectIDs), response.ResolvedDrsObject)
	}

	for i, objectID := range objectIDs {
		if response.ResolvedDrsObject[i].DrsObjectId != objectID {
			t.Fatalf("expected resolved id %q at index %d, got %+v", objectID, i, response.ResolvedDrsObject)
		}
	}

	cfg := requireIntegrationDrsConfig(t)
	expectedIssuer := strings.TrimRight(strings.TrimSpace(cfg.OidcUrl), "/")
	if realm := strings.TrimSpace(cfg.OidcRealm); realm != "" {
		expectedIssuer += "/realms/" + realm
	}

	if expectedIssuer != "" {
		for _, authorization := range response.ResolvedDrsObject {
			if len(authorization.BearerAuthIssuers) != 1 || authorization.BearerAuthIssuers[0] != expectedIssuer {
				t.Fatalf("expected bearer issuer %q, got %+v", expectedIssuer, authorization.BearerAuthIssuers)
			}
		}
	}
}
