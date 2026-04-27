//go:build e2e
// +build e2e

package e2e

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
	"github.com/spf13/viper"
)

var (
	e2eConfigOnce  sync.Once
	e2eConfigValue *drs_support.DrsConfig
	e2eFileConfig  *e2eTestConfig
	e2eConfigErr   error
)

const e2eConfigFileEnvVar = "DRS_E2E_CONFIG_FILE"

type e2eTestConfig struct {
	drs_support.DrsConfig `mapstructure:",squash"`

	E2E struct {
		BaseURL       string
		SkipTLSVerify bool
		BearerToken   string
	}
}

func TestMain(m *testing.M) {
	e2eConfigOnce.Do(func() {
		loadE2EConfigs()
	})

	if e2eConfigErr != nil {
		_, _ = fmt.Fprintln(os.Stderr, e2eConfigErr)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func requireE2EBaseURL(t *testing.T) string {
	t.Helper()

	baseURL := strings.TrimSpace(os.Getenv("DRS_E2E_BASE_URL"))
	if baseURL != "" {
		return baseURL
	}

	if cfg := optionalE2EFileConfig(t); cfg != nil && strings.TrimSpace(cfg.E2E.BaseURL) != "" {
		return strings.TrimSpace(cfg.E2E.BaseURL)
	}

	t.Fatalf("e2e tests require E2E.BaseURL or DRS_E2E_BASE_URL with %s set", e2eConfigFileEnvVar)
	return ""
}

func requireE2EBearerToken(t *testing.T) string {
	t.Helper()

	token := strings.TrimSpace(os.Getenv("DRS_TEST_BEARER_TOKEN"))
	if token == "" {
		if cfg := optionalE2EFileConfig(t); cfg != nil {
			token = strings.TrimSpace(cfg.E2E.BearerToken)
		}
	}
	if token == "" {
		t.Skip("DRS_TEST_BEARER_TOKEN is not set")
	}

	return token
}

func newE2EHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	skipTLSVerify := strings.EqualFold(strings.TrimSpace(os.Getenv("DRS_E2E_SKIP_TLS_VERIFY")), "true")
	if !skipTLSVerify {
		if cfg := optionalE2EFileConfig(nil); cfg != nil {
			skipTLSVerify = cfg.E2E.SkipTLSVerify
		}
	}
	if skipTLSVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

func optionalE2EDrsConfig(t *testing.T) *drs_support.DrsConfig {
	e2eConfigOnce.Do(func() {
		loadE2EConfigs()
	})

	if e2eConfigErr != nil && t != nil {
		t.Fatalf("%v", e2eConfigErr)
	}

	return e2eConfigValue
}

func optionalE2EFileConfig(t *testing.T) *e2eTestConfig {
	e2eConfigOnce.Do(func() {
		loadE2EConfigs()
	})

	if e2eConfigErr != nil && t != nil {
		t.Fatalf("%v", e2eConfigErr)
	}

	return e2eFileConfig
}

func loadE2EConfigs() {
	configFile := strings.TrimSpace(os.Getenv(e2eConfigFileEnvVar))
	if configFile == "" {
		e2eConfigErr = fmt.Errorf("e2e tests require %s to point at the shared E2E config file", e2eConfigFileEnvVar)
		return
	}

	resolvedPath, err := resolveE2EConfigPath(configFile)
	if err != nil {
		e2eConfigErr = err
		return
	}

	fileCfg, err := readE2ETestConfig(resolvedPath)
	if err != nil {
		e2eConfigErr = fmt.Errorf("read e2e config from %s=%q: %w", e2eConfigFileEnvVar, resolvedPath, err)
		return
	}
	e2eFileConfig = fileCfg

	if err := os.Setenv(drs_support.ConfigFileEnvVar, resolvedPath); err != nil {
		e2eConfigErr = fmt.Errorf("set %s=%q: %w", drs_support.ConfigFileEnvVar, resolvedPath, err)
		return
	}

	cfg, err := drs_support.ReadDrsConfig("", "", nil)
	if err != nil {
		e2eConfigErr = fmt.Errorf("read e2e drs config from %s=%q: %w", e2eConfigFileEnvVar, resolvedPath, err)
		return
	}

	applySharedDrsConfigFallbacks(cfg, fileCfg)
	e2eConfigValue = cfg
}

func readE2ETestConfig(configFile string) (*e2eTestConfig, error) {
	v := viper.New()
	v.SetConfigFile(configFile)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &e2eTestConfig{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func resolveE2EConfigPath(configFile string) (string, error) {
	configFile = strings.TrimSpace(configFile)
	if configFile == "" {
		return "", fmt.Errorf("empty config file path")
	}

	if filepath.IsAbs(configFile) {
		return configFile, nil
	}

	repoRoot, err := e2eRepoRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(repoRoot, configFile), nil
}

func e2eRepoRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve relative %s path: runtime caller unavailable", e2eConfigFileEnvVar)
	}

	e2eDir := filepath.Dir(filename)
	return filepath.Dir(e2eDir), nil
}

func applySharedDrsConfigFallbacks(cfg *drs_support.DrsConfig, fileCfg *e2eTestConfig) {
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
	if strings.TrimSpace(cfg.IrodsSecondaryTestUser) == "" {
		cfg.IrodsSecondaryTestUser = strings.TrimSpace(fileCfg.IrodsSecondaryTestUser)
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
