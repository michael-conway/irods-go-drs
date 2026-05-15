//go:build integration
// +build integration

package test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
	"github.com/spf13/viper"
)

const (
	integrationConfigFileEnvVar = "DRS_E2E_CONFIG_FILE"
)

type integrationTestConfig struct {
	drs_support.DrsConfig `mapstructure:",squash"`
}

var (
	integrationConfigOnce  sync.Once
	integrationConfigValue *drs_support.DrsConfig
	integrationConfigPath  string
	integrationConfigErr   error
)

func optionalBearerToken() string {
	// Bearer token is intentionally config-file only. If no token is available,
	// bearer-only integration tests are skipped.
	return ""
}

func requireBearerToken(t *testing.T) string {
	t.Helper()

	token := optionalBearerToken()
	if token == "" {
		t.Skip("no bearer token configured in shared integration config")
	}

	return token
}

func newIntegrationRequest(t *testing.T, method string, url string, body io.Reader) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	if token := optionalBearerToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return req
}

func requireIntegrationDrsConfig(t *testing.T) *drs_support.DrsConfig {
	t.Helper()

	cfg := optionalIntegrationDrsConfig(t)
	if cfg == nil {
		t.Fatalf("integration tests require %s to point at the shared E2E config file", integrationConfigFileEnvVar)
	}

	return cfg
}

func requireIntegrationRouteDrsConfig(t *testing.T) *drs_support.DrsConfig {
	t.Helper()

	cfg := requireIntegrationDrsConfig(t)
	if strings.TrimSpace(integrationConfigPath) == "" {
		t.Fatalf("integration tests require resolved %s path before starting route handlers", integrationConfigFileEnvVar)
	}

	t.Setenv(drs_support.ConfigFileEnvVar, integrationConfigPath)
	return cfg
}

func requireIntegrationIrodsConfig(t *testing.T) drs_support.DrsConfig {
	t.Helper()

	cfg := requireIntegrationDrsConfig(t)

	requireNonEmptyIntegrationValue(t, "IrodsHost", cfg.IrodsHost)
	if cfg.IrodsPort <= 0 {
		t.Fatalf("integration tests require IrodsPort in %s", integrationConfigFileEnvVar)
	}
	requireNonEmptyIntegrationValue(t, "IrodsZone", cfg.IrodsZone)
	requireNonEmptyIntegrationValue(t, "IrodsAdminUser", cfg.IrodsAdminUser)
	requireNonEmptyIntegrationValue(t, "IrodsAdminPassword", cfg.IrodsAdminPassword)
	requireNonEmptyIntegrationValue(t, "IrodsPrimaryTestUser", cfg.IrodsPrimaryTestUser)
	requireNonEmptyIntegrationValue(t, "IrodsAuthScheme", cfg.IrodsAuthScheme)

	return *cfg
}

func optionalIntegrationDrsConfig(t *testing.T) *drs_support.DrsConfig {
	integrationConfigOnce.Do(func() {
		loadIntegrationConfigs()
	})

	if integrationConfigErr != nil && t != nil {
		t.Fatalf("%v", integrationConfigErr)
	}

	return integrationConfigValue
}

func loadIntegrationConfigs() {
	configFile := strings.TrimSpace(os.Getenv(integrationConfigFileEnvVar))
	if configFile == "" {
		integrationConfigErr = fmt.Errorf("integration tests require %s to point at the shared E2E config file", integrationConfigFileEnvVar)
		return
	}

	resolvedPath, err := resolveIntegrationConfigPath(configFile)
	if err != nil {
		integrationConfigErr = err
		return
	}

	cfg, err := readIntegrationDrsConfig(resolvedPath)
	if err != nil {
		integrationConfigErr = fmt.Errorf("read integration config from %s=%q: %w", integrationConfigFileEnvVar, resolvedPath, err)
		return
	}
	integrationConfigValue = cfg
	integrationConfigPath = resolvedPath
}

func readIntegrationDrsConfig(configFile string) (*drs_support.DrsConfig, error) {
	v := viper.New()
	v.SetConfigFile(configFile)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &drs_support.DrsConfig{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func resolveIntegrationConfigPath(configFile string) (string, error) {
	configFile = strings.TrimSpace(configFile)
	if configFile == "" {
		return "", fmt.Errorf("empty config file path")
	}

	if filepath.IsAbs(configFile) {
		return configFile, nil
	}

	repoRoot, err := integrationRepoRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(repoRoot, configFile), nil
}

func integrationRepoRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve relative %s path: runtime caller unavailable", integrationConfigFileEnvVar)
	}

	testDir := filepath.Dir(filename)
	return filepath.Dir(testDir), nil
}

func requireNonEmptyIntegrationValue(t *testing.T, field string, value string) {
	t.Helper()

	if strings.TrimSpace(value) == "" {
		t.Fatalf("integration tests require %s in %s", field, integrationConfigFileEnvVar)
	}
}
