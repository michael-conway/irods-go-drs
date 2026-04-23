package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/michael-conway/irods-go-drs/drs-support"
)

func TestReadDrsConfigEnvOverride(t *testing.T) {
	t.Setenv("DRS_IRODS_HOST", "env-host")
	t.Setenv("DRS_OIDC_CLIENT_SECRET", "env-secret")
	t.Setenv("DRS_DRS_LOG_LEVEL", "debug")
	t.Setenv("DRS_LISTEN_PORT", "9090")
	t.Setenv("DRS_SERVICE_INFO_SAMPLE_INTERVAL_MINUTES", "11")
	t.Setenv("DRS_SERVICE_INFO_FILE_PATH", "/tmp/service-info.json")

	var confs = [1]string{"./resources/"}
	config, err := drs_support.ReadDrsConfig("drs-config1", "yaml", confs[:])
	if err != nil {
		t.Fatalf("error reading drs config: %s", err)
	}

	if config.IrodsHost != "env-host" {
		t.Fatalf("expected env override for IrodsHost, got %q", config.IrodsHost)
	}

	if config.OidcClientSecret != "env-secret" {
		t.Fatalf("expected env override for OidcClientSecret, got %q", config.OidcClientSecret)
	}

	if config.DrsLogLevel != "debug" {
		t.Fatalf("expected env override for DrsLogLevel, got %q", config.DrsLogLevel)
	}

	if config.DrsListenPort != 9090 {
		t.Fatalf("expected env override for DrsListenPort, got %d", config.DrsListenPort)
	}

	if config.ServiceInfoSampleIntervalMinutes != 11 {
		t.Fatalf("expected env override for ServiceInfoSampleIntervalMinutes, got %d", config.ServiceInfoSampleIntervalMinutes)
	}

	if config.ServiceInfoFilePath != "/tmp/service-info.json" {
		t.Fatalf("expected env override for ServiceInfoFilePath, got %q", config.ServiceInfoFilePath)
	}
}

func TestReadDrsConfigMissingFileReturnsError(t *testing.T) {
	_, err := drs_support.ReadDrsConfig("does-not-exist", "yaml", []string{"./resources/"})
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestReadDrsConfigSecretFileSupport(t *testing.T) {
	var confs = [1]string{"./resources/"}
	config, err := drs_support.ReadDrsConfig("drs-config-secret-files", "yaml", confs[:])
	if err != nil {
		t.Fatalf("error reading drs config: %s", err)
	}

	if config.IrodsDrsAdminPassword != "rods" {
		t.Fatalf("expected secret file value for IrodsDrsAdminPassword, got %q", config.IrodsDrsAdminPassword)
	}

	if config.OidcClientSecret != "test-oidc-secret" {
		t.Fatalf("expected secret file value for OidcClientSecret, got %q", config.OidcClientSecret)
	}
}

func TestReadDrsConfigConfigFileEnvOverride(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "custom-drs-config.yaml")
	configBody := "" +
		"DrsIdAvuValue: env-config\n" +
		"DrsListenPort: 9191\n" +
		"ServiceInfoSampleIntervalMinutes: 13\n" +
		"ServiceInfoFilePath: service-info.json\n" +
		"IrodsHost: env-file-host\n" +
		"IrodsPort: 1247\n" +
		"IrodsZone: tempZone\n" +
		"IrodsDrsAdminUser: rods\n" +
		"IrodsAuthScheme: native\n" +
		"IrodsNegotiationPolicy: native\n"

	if err := os.WriteFile(configPath, []byte(configBody), 0600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv(drs_support.ConfigFileEnvVar, configPath)

	config, err := drs_support.ReadDrsConfig("does-not-exist", "yaml", []string{"./resources/"})
	if err != nil {
		t.Fatalf("error reading drs config with %s override: %s", drs_support.ConfigFileEnvVar, err)
	}

	if config.DrsIdAvuValue != "env-config" {
		t.Fatalf("expected config from %s override, got %q", drs_support.ConfigFileEnvVar, config.DrsIdAvuValue)
	}

	if config.IrodsHost != "env-file-host" {
		t.Fatalf("expected host from %s override, got %q", drs_support.ConfigFileEnvVar, config.IrodsHost)
	}

	if config.DrsListenPort != 9191 {
		t.Fatalf("expected listen port from %s override, got %d", drs_support.ConfigFileEnvVar, config.DrsListenPort)
	}

	if config.ServiceInfoSampleIntervalMinutes != 13 {
		t.Fatalf("expected service info sample interval from %s override, got %d", drs_support.ConfigFileEnvVar, config.ServiceInfoSampleIntervalMinutes)
	}

	expectedServiceInfoPath := filepath.Join(dir, "service-info.json")
	if config.ServiceInfoFilePath != expectedServiceInfoPath {
		t.Fatalf("expected service info path from %s override to resolve to %q, got %q", drs_support.ConfigFileEnvVar, expectedServiceInfoPath, config.ServiceInfoFilePath)
	}
}

func TestReadDrsConfigTrimsWhitespaceFromInputs(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "custom-drs-config.yaml")
	configBody := "" +
		"DrsIdAvuValue: trimmed-config\n" +
		"DrsListenPort: 8181\n" +
		"ServiceInfoFilePath: service-info.json\n" +
		"IrodsHost: trimmed-host\n" +
		"IrodsPort: 1247\n" +
		"IrodsZone: tempZone\n" +
		"IrodsDrsAdminUser: rods\n" +
		"IrodsAuthScheme: native\n" +
		"IrodsNegotiationPolicy: native\n"

	if err := os.WriteFile(configPath, []byte(configBody), 0600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv(drs_support.ConfigFileEnvVar, "  "+configPath+"  ")

	config, err := drs_support.ReadDrsConfig(" drs-config1 ", " yaml ", []string{"  " + dir + "  ", "   "})
	if err != nil {
		t.Fatalf("error reading drs config with whitespace-padded inputs: %s", err)
	}

	if config.DrsIdAvuValue != "trimmed-config" {
		t.Fatalf("expected config file override after trimming, got %q", config.DrsIdAvuValue)
	}

	if config.IrodsHost != "trimmed-host" {
		t.Fatalf("expected trimmed host from config file override, got %q", config.IrodsHost)
	}

	expectedServiceInfoPath := filepath.Join(dir, "service-info.json")
	if config.ServiceInfoFilePath != expectedServiceInfoPath {
		t.Fatalf("expected trimmed service info path to resolve to %q, got %q", expectedServiceInfoPath, config.ServiceInfoFilePath)
	}
}
