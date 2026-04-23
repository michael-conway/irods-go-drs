package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

type ServiceInfoSnapshot struct {
	SampledAt       time.Time
	DrsConfig       *drs_support.DrsConfig
	ServiceInfoJSON json.RawMessage
}

func NewServiceInfoSnapshot(drsConfig *drs_support.DrsConfig) (*ServiceInfoSnapshot, error) {
	if drsConfig == nil {
		return nil, fmt.Errorf("no drs config provided")
	}

	now := time.Now().UTC()
	serviceInfo, err := loadServiceInfoJSON(drsConfig, now)
	if err != nil {
		return nil, err
	}

	return &ServiceInfoSnapshot{
		SampledAt:       now,
		DrsConfig:       drsConfig,
		ServiceInfoJSON: serviceInfo,
	}, nil
}

func loadServiceInfoJSON(drsConfig *drs_support.DrsConfig, now time.Time) (json.RawMessage, error) {
	serviceInfoData := defaultServiceInfoData(drsConfig, now)

	if drsConfig.ServiceInfoFilePath != "" {
		fileBytes, err := os.ReadFile(drsConfig.ServiceInfoFilePath)
		if err != nil {
			return nil, fmt.Errorf("read service info file %q: %w", drsConfig.ServiceInfoFilePath, err)
		}

		if err := json.Unmarshal(fileBytes, &serviceInfoData); err != nil {
			return nil, fmt.Errorf("unmarshal service info file %q: %w", drsConfig.ServiceInfoFilePath, err)
		}
	}

	applyServiceInfoPlaceholders(serviceInfoData, now)

	serviceInfoJSON, err := json.Marshal(serviceInfoData)
	if err != nil {
		return nil, fmt.Errorf("marshal service info json: %w", err)
	}

	return serviceInfoJSON, nil
}

func defaultServiceInfoData(drsConfig *drs_support.DrsConfig, now time.Time) map[string]interface{} {
	return map[string]interface{}{
		"maxBulkRequestLength": 1000,
		"type": map[string]interface{}{
			"artifact": "drs",
		},
		"drs": map[string]interface{}{
			"maxBulkRequestLength": 1000,
			"objectCount":          0,
			"totalObjectSize":      0,
		},
		"id":           "org.irods.irods-go-drs",
		"name":         "iRODS DRS Service",
		"description":  fmt.Sprintf("DRS service for iRODS zone %s on %s:%d", drsConfig.IrodsZone, drsConfig.IrodsHost, drsConfig.IrodsPort),
		"organization": map[string]interface{}{"name": "iRODS", "url": "https://irods.org/"},
		"createdAt":    now.Format(time.RFC3339),
		"updatedAt":    now.Format(time.RFC3339),
		"environment":  "prod",
		"version":      "1.5.0",
	}
}

func applyServiceInfoPlaceholders(serviceInfoData map[string]interface{}, now time.Time) {
	if _, ok := serviceInfoData["maxBulkRequestLength"]; !ok {
		serviceInfoData["maxBulkRequestLength"] = 1000
	}

	if _, ok := serviceInfoData["updatedAt"]; !ok {
		serviceInfoData["updatedAt"] = now.Format(time.RFC3339)
	}

	drsSection, ok := serviceInfoData["drs"].(map[string]interface{})
	if !ok || drsSection == nil {
		drsSection = map[string]interface{}{}
		serviceInfoData["drs"] = drsSection
	}

	if _, ok := drsSection["maxBulkRequestLength"]; !ok {
		drsSection["maxBulkRequestLength"] = serviceInfoData["maxBulkRequestLength"]
	}

	// Placeholder values until the real count logic is added.
	drsSection["objectCount"] = 0
	drsSection["totalObjectSize"] = 0
}

type ServiceInfoSampler struct {
	drsConfig *drs_support.DrsConfig
	interval  time.Duration

	mu       sync.RWMutex
	snapshot *ServiceInfoSnapshot
}

func NewServiceInfoSampler(drsConfig *drs_support.DrsConfig) (*ServiceInfoSampler, error) {
	if drsConfig == nil {
		return nil, fmt.Errorf("no drs config provided")
	}

	intervalMinutes := drsConfig.ServiceInfoSampleIntervalMinutes
	if intervalMinutes <= 0 {
		intervalMinutes = 5
	}

	return &ServiceInfoSampler{
		drsConfig: drsConfig,
		interval:  time.Duration(intervalMinutes) * time.Minute,
	}, nil
}

func (s *ServiceInfoSampler) Start(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("no context provided")
	}

	s.sample()

	ticker := time.NewTicker(s.interval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.sample()
			}
		}
	}()

	return nil
}

func (s *ServiceInfoSampler) Snapshot() *ServiceInfoSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshot
}

func (s *ServiceInfoSampler) sample() {
	snapshot, err := NewServiceInfoSnapshot(s.drsConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("service info sampling failed: %v", err))
		return
	}

	s.mu.Lock()
	s.snapshot = snapshot
	s.mu.Unlock()
}

var defaultServiceInfoSampler *ServiceInfoSampler

func SetDefaultServiceInfoSampler(sampler *ServiceInfoSampler) {
	defaultServiceInfoSampler = sampler
}

func GetDefaultServiceInfoSampler() *ServiceInfoSampler {
	return defaultServiceInfoSampler
}
