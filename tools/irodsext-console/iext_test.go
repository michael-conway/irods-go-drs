package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cyverse/go-irodsclient/config"
	"github.com/michael-conway/irods-go-drs/drs-support"
)

func TestIinit(t *testing.T) {
	// Load configuration from test resources
	// The path is relative to the project root, but tests run in the package directory.
	// We need to find the project root or use an absolute path.
	// Assuming the test is run from the project root using go test ./... or similar.
	// If run from tools/irodsext-console/, we need to go up two levels.
	wd, _ := os.Getwd()
	t.Logf("Current working directory: %s", wd)

	// Try to find the config file
	configPath := "../../test/resources"
	cfg, err := drs_support.ReadDrsConfig("drs-config1", "yaml", []string{configPath})
	if err != nil {
		t.Fatalf("failed to read drs-config1.yaml: %v", err)
	}

	// Setup a temporary environment file
	tempDir, err := os.MkdirTemp("", "iext-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	envFile := filepath.Join(tempDir, "irods_environment.json")

	// Create a new environment manager for testing
	testEnvManager, err := config.NewICommandsEnvironmentManager()
	if err != nil {
		t.Fatal(err)
	}
	testEnvManager.EnvironmentFilePath = envFile

	// Override the global envManager
	oldEnvManager := envManager
	envManager = testEnvManager
	defer func() { envManager = oldEnvManager }()

	cmd := getCommand()

	output := &bytes.Buffer{}
	cmd.Writer = output
	cmd.ErrWriter = output

	// Propagate writers to subcommands
	for _, sc := range cmd.Commands {
		sc.Writer = output
		sc.ErrWriter = output
	}

	// Run iinit command using values from cfg
	ctx := context.Background()
	args := []string{
		"iext", "iinit",
		"-h", cfg.IrodsHost,
		"-o", strconv.Itoa(cfg.IrodsPort),
		"-u", cfg.IrodsDrsAdminUser,
		"-z", cfg.IrodsZone,
		"-p", cfg.IrodsDrsAdminPassword,
		"-t", cfg.IrodsAuthScheme,
	}

	err = cmd.Run(ctx, args)
	if err != nil {
		t.Fatalf("cmd.Run failed: %v", err)
	}

	// Verify output
	outStr := output.String()
	t.Logf("Output: %q", outStr)
	if !strings.Contains(outStr, "saved iRODS environment") {
		t.Errorf("expected output to contain 'saved iRODS environment', got: %q", outStr)
	}

	// Verify file was created
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		t.Errorf("expected environment file %s to be created", envFile)
	}

	// Verify content of the file
	err = envManager.Load()
	if err != nil {
		t.Fatalf("failed to load saved environment: %v", err)
	}

	account, err := envManager.ToIRODSAccount()
	if err != nil {
		t.Fatalf("failed to get irods account: %v", err)
	}

	if account.Host != cfg.IrodsHost {
		t.Errorf("expected host %s, got %s", cfg.IrodsHost, account.Host)
	}
	if account.ClientUser != cfg.IrodsDrsAdminUser {
		t.Errorf("expected user %s, got %s", cfg.IrodsDrsAdminUser, account.ClientUser)
	}
}
