package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

type ServiceInfoSnapshot struct {
	SampledAt       time.Time
	DrsConfig       *drs_support.DrsConfig
	ServiceInfoJSON json.RawMessage
}

func NewServiceInfoSnapshot(drsConfig *drs_support.DrsConfig, summaries ...drs_support.DrsDataObjectSummary) (*ServiceInfoSnapshot, error) {
	if drsConfig == nil {
		return nil, fmt.Errorf("no drs config provided")
	}

	now := time.Now().UTC()
	var summary *drs_support.DrsDataObjectSummary
	if len(summaries) > 0 {
		summary = &summaries[0]
	}

	serviceInfo, err := loadServiceInfoJSON(drsConfig, now, summary)
	if err != nil {
		return nil, err
	}

	return &ServiceInfoSnapshot{
		SampledAt:       now,
		DrsConfig:       drsConfig,
		ServiceInfoJSON: serviceInfo,
	}, nil
}

func loadServiceInfoJSON(drsConfig *drs_support.DrsConfig, now time.Time, summary *drs_support.DrsDataObjectSummary) (json.RawMessage, error) {
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

	applyServiceInfoPlaceholders(serviceInfoData, now, summary)

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

func applyServiceInfoPlaceholders(serviceInfoData map[string]interface{}, now time.Time, summary *drs_support.DrsDataObjectSummary) {
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

	if summary == nil {
		drsSection["objectCount"] = 0
		drsSection["totalObjectSize"] = 0
		return
	}

	drsSection["objectCount"] = summary.DataObjectCount
	drsSection["totalObjectSize"] = summary.TotalSize
}

type ServiceInfoSampler struct {
	drsConfig       *drs_support.DrsConfig
	interval        time.Duration
	summaryProvider ServiceInfoSummaryProvider

	mu       sync.RWMutex
	snapshot *ServiceInfoSnapshot
}

type ServiceInfoSummaryProvider func(context.Context, *drs_support.DrsConfig) (drs_support.DrsDataObjectSummary, error)

type ServiceInfoSamplerOption func(*ServiceInfoSampler)

func WithServiceInfoSummaryProvider(provider ServiceInfoSummaryProvider) ServiceInfoSamplerOption {
	return func(s *ServiceInfoSampler) {
		s.summaryProvider = provider
	}
}

func NewServiceInfoSampler(drsConfig *drs_support.DrsConfig, options ...ServiceInfoSamplerOption) (*ServiceInfoSampler, error) {
	if drsConfig == nil {
		return nil, fmt.Errorf("no drs config provided")
	}

	intervalMinutes := drsConfig.ServiceInfoSampleIntervalMinutes
	if intervalMinutes <= 0 {
		intervalMinutes = 5
	}

	sampler := &ServiceInfoSampler{
		drsConfig:       drsConfig,
		interval:        time.Duration(intervalMinutes) * time.Minute,
		summaryProvider: queryServiceInfoDRSDataObjectSummary,
	}

	for _, option := range options {
		if option != nil {
			option(sampler)
		}
	}
	if sampler.summaryProvider == nil {
		sampler.summaryProvider = queryServiceInfoDRSDataObjectSummary
	}

	return sampler, nil
}

func (s *ServiceInfoSampler) Start(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("no context provided")
	}

	if err := s.sample(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(s.interval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.sample(ctx); err != nil {
					logger.Error(fmt.Sprintf("service info sampling failed: %v", err))
				}
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

func (s *ServiceInfoSampler) sample(ctx context.Context) error {
	summary, err := s.summaryProvider(ctx, s.drsConfig)
	if err != nil {
		return fmt.Errorf("query service info DRS object summary: %w", err)
	}

	snapshot, err := NewServiceInfoSnapshot(s.drsConfig, summary)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.snapshot = snapshot
	s.mu.Unlock()

	return nil
}

func queryServiceInfoDRSDataObjectSummary(ctx context.Context, drsConfig *drs_support.DrsConfig) (drs_support.DrsDataObjectSummary, error) {
	var summary drs_support.DrsDataObjectSummary
	if ctx == nil {
		return summary, fmt.Errorf("no context provided")
	}
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	if drsConfig == nil {
		return summary, fmt.Errorf("no drs config provided")
	}

	account := drsConfig.ToIrodsAccount()
	filesystem, err := irodsfs.NewFileSystemWithDefault(&account, "irods-go-drs-service-info-sampler")
	if err != nil {
		return summary, fmt.Errorf("create iRODS filesystem for service info sampler: %w", err)
	}
	defer filesystem.Release()

	return drs_support.QueryDrsDataObjectSummary(filesystem)
}

var defaultServiceInfoSampler *ServiceInfoSampler

func SetDefaultServiceInfoSampler(sampler *ServiceInfoSampler) {
	defaultServiceInfoSampler = sampler
}

func GetDefaultServiceInfoSampler() *ServiceInfoSampler {
	return defaultServiceInfoSampler
}
