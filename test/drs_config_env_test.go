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
}
