package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
	"github.com/michael-conway/irods-go-drs/internal"
)

func TestServiceInfoSamplerCapturesSnapshotOnStart(t *testing.T) {
	serviceInfoPath := writeServiceInfoFixture(t, `{"name":"Configured DRS","drs":{"maxBulkRequestLength":25}}`)

	sampler, err := internal.NewServiceInfoSampler(&drs_support.DrsConfig{
		ServiceInfoSampleIntervalMinutes: 1,
		ServiceInfoFilePath:              serviceInfoPath,
		IrodsHost:                        "localhost",
		IrodsPort:                        1247,
		IrodsZone:                        "tempZone",
	}, internal.WithServiceInfoSummaryProvider(fixedServiceInfoSummaryProvider(3, 4096)))
	if err != nil {
		t.Fatalf("create sampler: %v", err)
	}

	if err := sampler.Start(context.Background()); err != nil {
		t.Fatalf("start sampler: %v", err)
	}

	snapshot := sampler.Snapshot()
	if snapshot == nil {
		t.Fatal("expected snapshot to be populated immediately on start")
	}

	if snapshot.DrsConfig == nil {
		t.Fatal("expected snapshot to retain drs config")
	}

	if len(snapshot.ServiceInfoJSON) == 0 {
		t.Fatal("expected service info json payload in snapshot")
	}
}

func TestGetServiceInfoReturnsLatestSnapshot(t *testing.T) {
	serviceInfoPath := writeServiceInfoFixture(t, `{"name":"Configured DRS","organization":{"name":"CyVerse"},"drs":{"maxBulkRequestLength":25}}`)

	sampler, err := internal.NewServiceInfoSampler(&drs_support.DrsConfig{
		ServiceInfoSampleIntervalMinutes: 1,
		ServiceInfoFilePath:              serviceInfoPath,
		IrodsHost:                        "localhost",
		IrodsPort:                        1247,
		IrodsZone:                        "tempZone",
	}, internal.WithServiceInfoSummaryProvider(fixedServiceInfoSummaryProvider(7, 12345)))
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

	router := internal.NewRouter()
	req := httptest.NewRequest(http.MethodGet, "/ga4gh/drs/v1/service-info", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode service info: %v", err)
	}

	if response["name"] != "Configured DRS" {
		t.Fatalf("expected sampled service info response, got name %v", response["name"])
	}

	drsSection, ok := response["drs"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected drs section in response, got %#v", response["drs"])
	}

	if drsSection["objectCount"] != float64(7) {
		t.Fatalf("expected sampled objectCount 7, got %v", drsSection["objectCount"])
	}

	if drsSection["totalObjectSize"] != float64(12345) {
		t.Fatalf("expected sampled totalObjectSize 12345, got %v", drsSection["totalObjectSize"])
	}
}

func TestServiceInfoSamplerStartFailsWhenInitialSummaryQueryFails(t *testing.T) {
	serviceInfoPath := writeServiceInfoFixture(t, `{"name":"Configured DRS"}`)

	sampler, err := internal.NewServiceInfoSampler(&drs_support.DrsConfig{
		ServiceInfoSampleIntervalMinutes: 1,
		ServiceInfoFilePath:              serviceInfoPath,
		IrodsHost:                        "localhost",
		IrodsPort:                        1247,
		IrodsZone:                        "tempZone",
	}, internal.WithServiceInfoSummaryProvider(func(context.Context, *drs_support.DrsConfig) (drs_support.DrsDataObjectSummary, error) {
		return drs_support.DrsDataObjectSummary{}, fmt.Errorf("catalog unavailable")
	}))
	if err != nil {
		t.Fatalf("create sampler: %v", err)
	}

	err = sampler.Start(context.Background())
	if err == nil {
		t.Fatal("expected initial service info sample failure")
	}
	if sampler.Snapshot() != nil {
		t.Fatal("expected no snapshot after failed initial sample")
	}
}

func fixedServiceInfoSummaryProvider(count int64, totalSize int64) internal.ServiceInfoSummaryProvider {
	return func(context.Context, *drs_support.DrsConfig) (drs_support.DrsDataObjectSummary, error) {
		return drs_support.DrsDataObjectSummary{
			DataObjectCount: count,
			TotalSize:       totalSize,
		}, nil
	}
}

func writeServiceInfoFixture(t *testing.T, body string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "service-info.json")

	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("write service info fixture: %v", err)
	}

	return path
}
