//go:build integration
// +build integration

package test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
	"github.com/michael-conway/irods-go-drs/internal"
)

func TestServiceInfoEndpointIntegration(t *testing.T) {
	serviceInfoPath := writeServiceInfoFixture(t, `{
		"name":"Configured DRS",
		"organization":{"name":"CyVerse"},
		"drs":{"maxBulkRequestLength":25}
	}`)

	sampler, err := internal.NewServiceInfoSampler(&drs_support.DrsConfig{
		ServiceInfoSampleIntervalMinutes: 1,
		ServiceInfoFilePath:              serviceInfoPath,
		IrodsHost:                        "localhost",
		IrodsPort:                        1247,
		IrodsZone:                        "tempZone",
	})
	if err != nil {
		t.Fatalf("create sampler: %v", err)
	}

	if err := sampler.Start(context.Background()); err != nil {
		t.Fatalf("start sampler: %v", err)
	}

	internal.SetDefaultServiceInfoSampler(sampler)
	t.Cleanup(func() {
		internal.SetDefaultServiceInfoSampler(nil)
	})

	server := httptest.NewServer(internal.NewRouter())
	defer server.Close()

	req := newIntegrationRequest(t, http.MethodGet, server.URL+"/ga4gh/drs/v1/service-info", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get service info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json; charset=UTF-8" {
		t.Fatalf("expected json content type, got %q", contentType)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode service info: %v", err)
	}

	if response["name"] != "Configured DRS" {
		t.Fatalf("expected sampled service info response, got name %v", response["name"])
	}

	drsSection, ok := response["drs"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected drs section in response, got %#v", response["drs"])
	}

	if drsSection["objectCount"] != float64(0) {
		t.Fatalf("expected placeholder objectCount 0, got %v", drsSection["objectCount"])
	}

	if drsSection["totalObjectSize"] != float64(0) {
		t.Fatalf("expected placeholder totalObjectSize 0, got %v", drsSection["totalObjectSize"])
	}
}
