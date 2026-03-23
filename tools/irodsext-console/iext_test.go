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
	"github.com/cyverse/go-irodsclient/fs"
	"github.com/cyverse/go-irodsclient/irods/types"
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

func TestImiscsvrinfo_NoEnv(t *testing.T) {
	// Setup a temporary environment file that doesn't exist
	tempDir, err := os.MkdirTemp("", "iext-test-info")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	envFile := filepath.Join(tempDir, "non_existent_env.json")

	testEnvManager, _ := config.NewICommandsEnvironmentManager()
	testEnvManager.EnvironmentFilePath = envFile

	oldEnvManager := envManager
	envManager = testEnvManager
	defer func() { envManager = oldEnvManager }()

	cmd := getCommand()
	output := &bytes.Buffer{}
	errOutput := &bytes.Buffer{}
	cmd.Writer = output
	cmd.ErrWriter = errOutput

	// Propagate writers to subcommands
	for _, sc := range cmd.Commands {
		sc.Writer = output
		sc.ErrWriter = errOutput
	}

	// Run imiscsvrinfo command without an environment file
	// It should fail because there is no account info
	ctx := context.Background()
	args := []string{"iext", "imiscsvrinfo"}

	err = cmd.Run(ctx, args)
	// It will likely return an error because it can't connect to irods with empty account
	// But let's check if it tried to load and reported error

	if err == nil {
		t.Log("cmd.Run returned no error, but it was expected to fail connecting")
	}

	if !strings.Contains(errOutput.String(), "error getting irods account") && !strings.Contains(errOutput.String(), "error connecting to irods") {
		t.Errorf("expected error output about missing account or connection failure, got: %s", errOutput.String())
	}
}

type mockFileSystem struct {
	homeDir string
	entries map[string]*fs.Entry
}

func (m *mockFileSystem) GetHomeDirPath() string {
	return m.homeDir
}
func (m *mockFileSystem) Stat(irodsPath string) (*fs.Entry, error) {
	if entry, ok := m.entries[irodsPath]; ok {
		return entry, nil
	}
	return nil, types.NewFileNotFoundError(irodsPath)
}
func (m *mockFileSystem) Release() {}

func TestIpwd_DefaultToHome(t *testing.T) {
	// Setup a temporary environment file
	tempDir, err := os.MkdirTemp("", "iext-test-ipwd")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	envFile := filepath.Join(tempDir, "irods_environment.json")

	// Create an environment file with valid account but EMPTY CurrentWorkingDir
	testEnvManager, _ := config.NewICommandsEnvironmentManager()
	testEnvManager.EnvironmentFilePath = envFile
	testEnvManager.Environment = &config.Config{
		Host:                 "localhost",
		Port:                 1247,
		Username:             "testuser",
		ZoneName:             "testZone",
		AuthenticationScheme: "native",
		CurrentWorkingDir:    "", // This is what we want to test
	}
	err = testEnvManager.SaveEnvironment()
	if err != nil {
		t.Fatalf("failed to save environment: %v", err)
	}

	oldEnvManager := envManager
	envManager = testEnvManager
	defer func() { envManager = oldEnvManager }()

	// Mock filesystem creation
	expectedHome := "/testZone/home/testuser"
	oldCreateFS := createFileSystem
	createFileSystem = func(account *types.IRODSAccount, applicationName string) (FileSystem, error) {
		return &mockFileSystem{homeDir: expectedHome}, nil
	}
	defer func() { createFileSystem = oldCreateFS }()

	cmd := getCommand()
	output := &bytes.Buffer{}
	cmd.Writer = output
	cmd.ErrWriter = output

	// Propagate writers to subcommands
	for _, sc := range cmd.Commands {
		sc.Writer = output
		sc.ErrWriter = output
	}

	// Run ipwd command
	ctx := context.Background()
	args := []string{"iext", "ipwd"}

	err = cmd.Run(ctx, args)
	if err != nil {
		t.Fatalf("cmd.Run failed: %v", err)
	}

	outStr := output.String()
	t.Logf("Output: %q", outStr)

	if !strings.Contains(outStr, expectedHome) {
		t.Errorf("expected output to contain %q, got %q", expectedHome, outStr)
	}

	// Verify it was saved to the environment
	if envManager.Environment.CurrentWorkingDir != expectedHome {
		t.Errorf("expected CurrentWorkingDir to be updated to %q, got %q", expectedHome, envManager.Environment.CurrentWorkingDir)
	}
}

func TestIcd(t *testing.T) {
	// Setup a temporary environment file
	tempDir, err := os.MkdirTemp("", "iext-test-icd")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	envFile := filepath.Join(tempDir, "irods_environment.json")

	// Create an environment file with a fixed HomeDirPath and CurrentWorkingDir
	homeDir := "/testZone/home/testuser"
	cwd := "/testZone/home/testuser"
	testEnvManager, _ := config.NewICommandsEnvironmentManager()
	testEnvManager.EnvironmentFilePath = envFile
	testEnvManager.Environment = &config.Config{
		Host:              "localhost",
		Port:              1247,
		Username:          "testuser",
		ZoneName:          "testZone",
		CurrentWorkingDir: cwd,
	}
	err = testEnvManager.SaveEnvironment()
	if err != nil {
		t.Fatalf("failed to save environment: %v", err)
	}

	oldEnvManager := envManager
	envManager = testEnvManager
	defer func() { envManager = oldEnvManager }()

	// Mock filesystem creation
	oldCreateFS := createFileSystem
	createFileSystem = func(account *types.IRODSAccount, applicationName string) (FileSystem, error) {
		return &mockFileSystem{
			homeDir: homeDir,
			entries: map[string]*fs.Entry{
				"/testZone/home/testuser": {
					Type: fs.DirectoryEntry,
					Path: "/testZone/home/testuser",
				},
				"/tempZone/sub": {
					Type: fs.DirectoryEntry,
					Path: "/tempZone/sub",
				},
				"/testZone/home/testuser/sub": {
					Type: fs.DirectoryEntry,
					Path: "/testZone/home/testuser/sub",
				},
				"/testZone/home": {
					Type: fs.DirectoryEntry,
					Path: "/testZone/home",
				},
				"/testZone/home/testuser/file": {
					Type: fs.FileEntry,
					Path: "/testZone/home/testuser/file",
				},
			},
		}, nil
	}
	defer func() { createFileSystem = oldCreateFS }()

	testCases := []struct {
		args        []string
		expectedCwd string
		expectError bool
	}{
		{[]string{"iext", "icd"}, homeDir, false},                                // empty -> home
		{[]string{"iext", "icd", "/tempZone/sub"}, "/tempZone/sub", false},       // absolute
		{[]string{"iext", "icd", "sub"}, "/testZone/home/testuser/sub", false},   // relative
		{[]string{"iext", "icd", "./sub"}, "/testZone/home/testuser/sub", false}, // dot-relative
		{[]string{"iext", "icd", ".."}, "/testZone/home", false},                 // double-dot relative
		{[]string{"iext", "icd", "nonexistent"}, cwd, true},                      // nonexistent
		{[]string{"iext", "icd", "file"}, cwd, true},                             // file not dir
	}

	for _, tc := range testCases {
		// Reset CWD for each test case
		envManager.Environment.CurrentWorkingDir = cwd
		err = envManager.SaveEnvironment()
		if err != nil {
			t.Fatalf("failed to save environment: %v", err)
		}

		cmd := getCommand()
		output := &bytes.Buffer{}
		cmd.Writer = output
		cmd.ErrWriter = output

		for _, sc := range cmd.Commands {
			sc.Writer = output
			sc.ErrWriter = output
		}

		err = cmd.Run(context.Background(), tc.args)
		if tc.expectError {
			if err == nil {
				t.Errorf("for args %v: expected error but got none", tc.args)
			}
			if envManager.Environment.CurrentWorkingDir != tc.expectedCwd {
				t.Errorf("for args %v (error case): expected CWD to remain %q, got %q", tc.args, tc.expectedCwd, envManager.Environment.CurrentWorkingDir)
			}
			continue
		}

		if err != nil {
			t.Errorf("cmd.Run failed for args %v: %v", tc.args, err)
			continue
		}

		if envManager.Environment.CurrentWorkingDir != tc.expectedCwd {
			t.Errorf("for args %v: expected CWD %q, got %q", tc.args, tc.expectedCwd, envManager.Environment.CurrentWorkingDir)
		}
	}
}
