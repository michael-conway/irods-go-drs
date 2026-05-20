//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"testing"
	"time"

	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
	"github.com/michael-conway/irods-go-drs/internal"
)

func TestServiceInfoSamplerCountsDRSDataObjectsE2E(t *testing.T) {
	cfg := requireE2EIRODSConfig(t)
	samplerConfig := *cfg
	samplerConfig.ServiceInfoFilePath = ""
	samplerConfig.ServiceInfoSampleIntervalMinutes = 60

	before := sampleServiceInfoTotalsE2E(t, &samplerConfig)

	filesystem := newE2EIRODSFilesystem(t, requireE2EEffectiveUser(t))
	testUser := filesystem.GetAccount().ClientUser
	rootPath := fmt.Sprintf("/%s/home/%s/drs-service-info-e2e-%d", cfg.IrodsZone, testUser, time.Now().UnixNano())
	if err := filesystem.MakeDir(rootPath, true); err != nil {
		filesystem.Release()
		t.Fatalf("create service info e2e fixture root %q: %v", rootPath, err)
	}

	t.Cleanup(func() {
		defer filesystem.Release()
		if err := filesystem.RemoveDir(rootPath, true, true); err != nil && filesystem.Exists(rootPath) {
			t.Errorf("cleanup service info e2e fixture root %q: %v", rootPath, err)
		}
	})

	contents := [][]byte{
		[]byte("service info e2e object one\n"),
		[]byte("service info e2e object two with more bytes\n"),
	}
	var expectedSizeDelta int64
	for idx, content := range contents {
		expectedSizeDelta += int64(len(content))
		objectPath := path.Join(rootPath, fmt.Sprintf("object-%d.txt", idx+1))
		if _, err := filesystem.UploadFileFromBuffer(bytes.NewBuffer(content), objectPath, "", false, true, nil); err != nil {
			t.Fatalf("upload service info e2e object %q: %v", objectPath, err)
		}
		if _, err := drs_support.CreateDrsObjectFromDataObject(filesystem, objectPath, "", "", nil); err != nil {
			t.Fatalf("create service info e2e DRS object %q: %v", objectPath, err)
		}
	}

	after := sampleServiceInfoTotalsE2E(t, &samplerConfig)

	expectedCount := before.ObjectCount + int64(len(contents))
	if after.ObjectCount != expectedCount {
		t.Fatalf("expected sampled objectCount %d, got %d (before %+v, after %+v)", expectedCount, after.ObjectCount, before, after)
	}

	expectedSize := before.TotalObjectSize + expectedSizeDelta
	if after.TotalObjectSize != expectedSize {
		t.Fatalf("expected sampled totalObjectSize %d, got %d (before %+v, after %+v)", expectedSize, after.TotalObjectSize, before, after)
	}
}

type serviceInfoTotalsE2E struct {
	ObjectCount     int64
	TotalObjectSize int64
}

func sampleServiceInfoTotalsE2E(t *testing.T, cfg *drs_support.DrsConfig) serviceInfoTotalsE2E {
	t.Helper()

	sampler, err := internal.NewServiceInfoSampler(cfg)
	if err != nil {
		t.Fatalf("create service info sampler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sampler.Start(ctx); err != nil {
		t.Fatalf("start service info sampler: %v", err)
	}
	cancel()

	snapshot := sampler.Snapshot()
	if snapshot == nil {
		t.Fatal("expected service info sampler snapshot")
	}

	var payload struct {
		DRS struct {
			ObjectCount     int64 `json:"objectCount"`
			TotalObjectSize int64 `json:"totalObjectSize"`
		} `json:"drs"`
	}
	if err := json.Unmarshal(snapshot.ServiceInfoJSON, &payload); err != nil {
		t.Fatalf("decode service info sampler snapshot: %v", err)
	}

	return serviceInfoTotalsE2E{
		ObjectCount:     payload.DRS.ObjectCount,
		TotalObjectSize: payload.DRS.TotalObjectSize,
	}
}
