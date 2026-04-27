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
	testBearerTokenEnvVar       = "DRS_TEST_BEARER_TOKEN"
	integrationConfigFileEnvVar = "DRS_E2E_CONFIG_FILE"
)

type integrationTestConfig struct {
	drs_support.DrsConfig `mapstructure:",squash"`

	E2E struct {
		BearerToken string
	}
}

var (
	integrationConfigOnce  sync.Once
	integrationConfigValue *drs_support.DrsConfig
	integrationFileConfig  *integrationTestConfig
	integrationConfigErr   error
)

func optionalBearerToken() string {
	token := strings.TrimSpace(os.Getenv(testBearerTokenEnvVar))
	if token != "" {
		return token
	}

	if cfg := optionalIntegrationFileConfig(nil); cfg != nil {
		return strings.TrimSpace(cfg.E2E.BearerToken)
	}

	return ""
}

func requireBearerToken(t *testing.T) string {
	t.Helper()

	token := optionalBearerToken()
	if token == "" {
		t.Skip("no bearer token configured in E2E.BearerToken or DRS_TEST_BEARER_TOKEN")
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

func optionalIntegrationFileConfig(t *testing.T) *integrationTestConfig {
	integrationConfigOnce.Do(func() {
		loadIntegrationConfigs()
	})

	if integrationConfigErr != nil && t != nil {
		t.Fatalf("%v", integrationConfigErr)
	}

	return integrationFileConfig
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

	fileCfg, err := readIntegrationTestConfig(resolvedPath)
	if err != nil {
		integrationConfigErr = fmt.Errorf("read integration config from %s=%q: %w", integrationConfigFileEnvVar, resolvedPath, err)
		return
	}
	integrationFileConfig = fileCfg

	if err := os.Setenv(drs_support.ConfigFileEnvVar, resolvedPath); err != nil {
		integrationConfigErr = fmt.Errorf("set %s=%q: %w", drs_support.ConfigFileEnvVar, resolvedPath, err)
		return
	}

	cfg, err := drs_support.ReadDrsConfig("", "", nil)
	if err != nil {
		integrationConfigErr = fmt.Errorf("read integration drs config from %s=%q: %w", integrationConfigFileEnvVar, resolvedPath, err)
		return
	}

	applySharedDrsConfigFallbacks(cfg, fileCfg)
	integrationConfigValue = cfg
}

func readIntegrationTestConfig(configFile string) (*integrationTestConfig, error) {
	v := viper.New()
	v.SetConfigFile(configFile)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &integrationTestConfig{}
	if err := v.Unmarshal(&cfg.DrsConfig); err != nil {
		return nil, err
	}
	if err := v.UnmarshalKey("E2E", &cfg.E2E); err != nil {
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

func applySharedDrsConfigFallbacks(cfg *drs_support.DrsConfig, fileCfg *integrationTestConfig) {
	if cfg == nil || fileCfg == nil {
		return
	}

	if strings.TrimSpace(cfg.IrodsHost) == "" {
		cfg.IrodsHost = strings.TrimSpace(fileCfg.IrodsHost)
	}
	if cfg.IrodsPort == 0 {
		cfg.IrodsPort = fileCfg.IrodsPort
	}
	if strings.TrimSpace(cfg.IrodsZone) == "" {
		cfg.IrodsZone = strings.TrimSpace(fileCfg.IrodsZone)
	}
	if strings.TrimSpace(cfg.IrodsAdminUser) == "" {
		cfg.IrodsAdminUser = strings.TrimSpace(fileCfg.IrodsAdminUser)
	}
	if strings.TrimSpace(cfg.IrodsAdminPassword) == "" {
		cfg.IrodsAdminPassword = strings.TrimSpace(fileCfg.IrodsAdminPassword)
	}
	if strings.TrimSpace(cfg.IrodsPrimaryTestUser) == "" {
		cfg.IrodsPrimaryTestUser = strings.TrimSpace(fileCfg.IrodsPrimaryTestUser)
	}
	if strings.TrimSpace(cfg.IrodsPrimaryTestPassword) == "" {
		cfg.IrodsPrimaryTestPassword = strings.TrimSpace(fileCfg.IrodsPrimaryTestPassword)
	}
	if strings.TrimSpace(cfg.IrodsSecondaryTestUser) == "" {
		cfg.IrodsSecondaryTestUser = strings.TrimSpace(fileCfg.IrodsSecondaryTestUser)
	}
	if strings.TrimSpace(cfg.IrodsSecondaryTestPassword) == "" {
		cfg.IrodsSecondaryTestPassword = strings.TrimSpace(fileCfg.IrodsSecondaryTestPassword)
	}
	if strings.TrimSpace(cfg.IrodsAuthScheme) == "" {
		cfg.IrodsAuthScheme = strings.TrimSpace(fileCfg.IrodsAuthScheme)
	}
	if strings.TrimSpace(cfg.IrodsNegotiationPolicy) == "" {
		cfg.IrodsNegotiationPolicy = strings.TrimSpace(fileCfg.IrodsNegotiationPolicy)
	}
	if strings.TrimSpace(cfg.IrodsDefaultResource) == "" {
		cfg.IrodsDefaultResource = strings.TrimSpace(fileCfg.IrodsDefaultResource)
	}
	if strings.TrimSpace(cfg.OidcUrl) == "" {
		cfg.OidcUrl = strings.TrimSpace(fileCfg.OidcUrl)
	}
	if strings.TrimSpace(cfg.OidcClientId) == "" {
		cfg.OidcClientId = strings.TrimSpace(fileCfg.OidcClientId)
	}
	if strings.TrimSpace(cfg.OidcClientSecret) == "" {
		cfg.OidcClientSecret = strings.TrimSpace(fileCfg.OidcClientSecret)
	}
	if strings.TrimSpace(cfg.OidcRealm) == "" {
		cfg.OidcRealm = strings.TrimSpace(fileCfg.OidcRealm)
	}
	if strings.TrimSpace(cfg.OidcScope) == "" {
		cfg.OidcScope = strings.TrimSpace(fileCfg.OidcScope)
	}
}
