package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	irodsclientconfig "github.com/cyverse/go-irodsclient/config"
	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
	"github.com/urfave/cli/v3"
)

func TestIinit(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, "irods_environment.json")

	testEnvManager, err := irodsclientconfig.NewICommandsEnvironmentManager()
	if err != nil {
		t.Fatal(err)
	}
	if err := testEnvManager.SetEnvironmentFilePath(envFile); err != nil {
		t.Fatalf("set environment file path: %v", err)
	}

	oldEnvManager := envManager
	envManager = testEnvManager
	defer func() { envManager = oldEnvManager }()

	cmd := getCommand()
	output := &bytes.Buffer{}
	cmd.Writer = output
	cmd.ErrWriter = output
	for _, sc := range cmd.Commands {
		sc.Writer = output
		sc.ErrWriter = output
	}

	args := []string{
		APP_NAME, "iinit",
		"-h", "irods.example.org",
		"-o", strconv.Itoa(1247),
		"-u", "rods",
		"-z", "tempZone",
		"-p", "rods-password",
		"-t", "native",
	}

	if err := cmd.Run(context.Background(), args); err != nil {
		t.Fatalf("cmd.Run failed: %v", err)
	}

	if !strings.Contains(output.String(), "saved iRODS environment") {
		t.Fatalf("expected success output, got %q", output.String())
	}

	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		t.Fatalf("expected environment file %s to be created", envFile)
	}

	if err := envManager.Load(); err != nil {
		t.Fatalf("failed to load saved environment: %v", err)
	}

	account, err := envManager.ToIRODSAccount()
	if err != nil {
		t.Fatalf("failed to convert environment to account: %v", err)
	}

	if account.Host != "irods.example.org" {
		t.Fatalf("expected host to be persisted, got %q", account.Host)
	}
}

func TestConnectFileSystemLoadsSavedEnvironment(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, "irods_environment.json")

	testEnvManager, err := irodsclientconfig.NewICommandsEnvironmentManager()
	if err != nil {
		t.Fatal(err)
	}
	if err := testEnvManager.SetEnvironmentFilePath(envFile); err != nil {
		t.Fatalf("set environment file path: %v", err)
	}

	testEnvManager.Environment.Host = "irods.example.org"
	testEnvManager.Environment.Port = 1247
	testEnvManager.Environment.ZoneName = "tempZone"
	testEnvManager.Environment.Username = "rods"
	testEnvManager.Environment.Password = "rods-password"
	testEnvManager.Environment.AuthenticationScheme = "native"
	testEnvManager.Environment.Home = "/tempZone/home/rods"
	if err := testEnvManager.SaveEnvironment(); err != nil {
		t.Fatalf("save environment: %v", err)
	}

	testEnvManager.Environment = irodsclientconfig.GetDefaultConfig()

	oldEnvManager := envManager
	envManager = testEnvManager
	defer func() { envManager = oldEnvManager }()

	var capturedAccount *irodstypes.IRODSAccount
	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		capturedAccount = account
		return newFakeFileSystem("/tempZone/home/rods/file.txt"), nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	filesystem, err := connectFileSystem()
	if err != nil {
		t.Fatalf("connectFileSystem failed: %v", err)
	}
	defer filesystem.Release()

	if capturedAccount == nil {
		t.Fatal("expected connectFileSystem to create an account")
	}

	if capturedAccount.Host != "irods.example.org" {
		t.Fatalf("expected host loaded from saved environment, got %q", capturedAccount.Host)
	}
}

func TestResolveIRODSPathUsesSessionCwd(t *testing.T) {
	testEnvManager, err := irodsclientconfig.NewICommandsEnvironmentManager()
	if err != nil {
		t.Fatal(err)
	}

	testEnvManager.Environment = &irodsclientconfig.Config{
		Home: "/tempZone/home/rods",
	}
	testEnvManager.Session = &irodsclientconfig.Config{
		CurrentWorkingDir: "/tempZone/home/rods/projects/demo",
	}

	oldEnvManager := envManager
	envManager = testEnvManager
	defer func() { envManager = oldEnvManager }()

	fakeFS := newFakeFileSystem("/tempZone/home/rods/file.txt")
	resolvedPath, err := resolveIRODSPath("USERGUIDE.md", fakeFS)
	if err != nil {
		t.Fatalf("resolveIRODSPath failed: %v", err)
	}

	expectedPath := "/tempZone/home/rods/projects/demo/USERGUIDE.md"
	if resolvedPath != expectedPath {
		t.Fatalf("expected resolved path %q, got %q", expectedPath, resolvedPath)
	}
}

func TestDrsCommands(t *testing.T) {
	const objectPath = "/tempZone/home/rods/file.txt"

	testEnvManager, err := irodsclientconfig.NewICommandsEnvironmentManager()
	if err != nil {
		t.Fatal(err)
	}
	testEnvManager.Environment = &irodsclientconfig.Config{
		Host:                 "irods.example.org",
		Port:                 1247,
		ZoneName:             "tempZone",
		Username:             "rods",
		Password:             "rods-password",
		AuthenticationScheme: "native",
		CurrentWorkingDir:    "/tempZone/home/rods",
		Home:                 "/tempZone/home/rods",
	}

	oldEnvManager := envManager
	envManager = testEnvManager
	defer func() { envManager = oldEnvManager }()

	fakeFS := newFakeFileSystem(objectPath)
	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()

	makeOutput := runCommand(t, cmd, []string{
		APP_NAME, "drsmake", objectPath,
		"--mime-type", "application/octet-stream",
		"--description", "test description",
		"--alias", "alias-1",
		"--alias", "alias-2",
	})
	if !strings.Contains(makeOutput, "\"drsId\"") {
		t.Fatalf("expected drsmake to emit a drs id, got %q", makeOutput)
	}

	infoByPath := runCommand(t, cmd, []string{APP_NAME, "drsinfo", "--path", objectPath})
	if !strings.Contains(infoByPath, "\"path\": \""+objectPath+"\"") {
		t.Fatalf("expected drsinfo path output to contain the object path, got %q", infoByPath)
	}
	if !strings.Contains(infoByPath, "\"mimeType\": \"application/octet-stream\"") {
		t.Fatalf("expected drsinfo output to contain the provided mime type, got %q", infoByPath)
	}
	if !strings.Contains(infoByPath, "\"alias-1\"") || !strings.Contains(infoByPath, "\"alias-2\"") {
		t.Fatalf("expected drsinfo output to contain aliases, got %q", infoByPath)
	}

	object, err := drs_support.GetDrsObjectByIRODSPath(fakeFS, objectPath)
	if err != nil {
		t.Fatalf("expected fake filesystem to expose DRS object by path: %v", err)
	}

	infoByID := runCommand(t, cmd, []string{APP_NAME, "drsinfo", "--id", object.Id})
	if !strings.Contains(infoByID, "\"drsId\": \""+object.Id+"\"") {
		t.Fatalf("expected drsinfo id output to contain the drs id, got %q", infoByID)
	}

	removeOutput := runCommand(t, cmd, []string{APP_NAME, "drsrm", objectPath})
	if !strings.Contains(removeOutput, "\"path\": \""+objectPath+"\"") {
		t.Fatalf("expected drsrm output to contain the removed path, got %q", removeOutput)
	}

	if _, err := drs_support.GetDrsObjectByIRODSPath(fakeFS, objectPath); err == nil {
		t.Fatal("expected DRS metadata to be removed from the fake filesystem")
	}
}

func TestDrsMakeHelp(t *testing.T) {
	cmd := getCommand()

	helpOutput := runCommand(t, cmd, []string{APP_NAME, "drsmake", "--help"})
	if !strings.Contains(helpOutput, "drscmd drsmake") {
		t.Fatalf("expected drsmake help header, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "<irods-data-object-path>") {
		t.Fatalf("expected drsmake help to include argument usage, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "--mime-type") {
		t.Fatalf("expected drsmake help to include mime-type flag, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "--description") {
		t.Fatalf("expected drsmake help to include description flag, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "--alias") {
		t.Fatalf("expected drsmake help to include alias flag, got %q", helpOutput)
	}
}

func TestIinitHelp(t *testing.T) {
	cmd := getCommand()

	helpOutput := runCommand(t, cmd, []string{APP_NAME, "iinit", "--help"})
	if !strings.Contains(helpOutput, "drscmd iinit") {
		t.Fatalf("expected iinit help header, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "drscmd iinit [flags]") {
		t.Fatalf("expected iinit help usage, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "-h                 host") {
		t.Fatalf("expected iinit help to preserve host flag, got %q", helpOutput)
	}
}

func TestDrsInfoHelp(t *testing.T) {
	cmd := getCommand()

	helpOutput := runCommand(t, cmd, []string{APP_NAME, "drsinfo", "--help"})
	if !strings.Contains(helpOutput, "drscmd drsinfo") {
		t.Fatalf("expected drsinfo help header, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "<path-or-drs-id>") {
		t.Fatalf("expected drsinfo help to include argument usage, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "--path") || !strings.Contains(helpOutput, "--id") {
		t.Fatalf("expected drsinfo help to include selector flags, got %q", helpOutput)
	}
}

func TestDrsRemoveHelp(t *testing.T) {
	cmd := getCommand()

	helpOutput := runCommand(t, cmd, []string{APP_NAME, "drsrm", "--help"})
	if !strings.Contains(helpOutput, "drscmd drsrm") {
		t.Fatalf("expected drsrm help header, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "<irods-data-object-path>") {
		t.Fatalf("expected drsrm help to include argument usage, got %q", helpOutput)
	}
}

func TestDrsMakeUsageErrorShowsHelp(t *testing.T) {
	cmd := getCommand()

	output, err := runCommandAllowError(t, cmd, []string{APP_NAME, "drsmake"})
	if err == nil {
		t.Fatal("expected drsmake usage error")
	}

	if !strings.Contains(output, "drscmd drsmake") {
		t.Fatalf("expected drsmake help content on usage error, got %q", output)
	}

	if !strings.Contains(err.Error(), "an iRODS data object path is required") {
		t.Fatalf("expected drsmake usage error message, got %v", err)
	}
}

func TestDrsInfoUsageErrorShowsHelp(t *testing.T) {
	cmd := getCommand()

	output, err := runCommandAllowError(t, cmd, []string{APP_NAME, "drsinfo"})
	if err == nil {
		t.Fatal("expected drsinfo usage error")
	}

	if !strings.Contains(output, "drscmd drsinfo") {
		t.Fatalf("expected drsinfo help content on usage error, got %q", output)
	}

	if !strings.Contains(err.Error(), "a DRS id or iRODS path is required") {
		t.Fatalf("expected drsinfo usage error message, got %v", err)
	}
}

func TestDrsInfoConflictingFlagsShowHelp(t *testing.T) {
	cmd := getCommand()

	output, err := runCommandAllowError(t, cmd, []string{APP_NAME, "drsinfo", "--path", "--id", "value"})
	if err == nil {
		t.Fatal("expected drsinfo conflicting flags error")
	}

	if !strings.Contains(output, "drscmd drsinfo") {
		t.Fatalf("expected drsinfo help content on conflicting flags, got %q", output)
	}

	if !strings.Contains(err.Error(), "--path and --id cannot be used together") {
		t.Fatalf("expected drsinfo conflicting flags message, got %v", err)
	}
}

func TestDrsRemoveUsageErrorShowsHelp(t *testing.T) {
	cmd := getCommand()

	output, err := runCommandAllowError(t, cmd, []string{APP_NAME, "drsrm"})
	if err == nil {
		t.Fatal("expected drsrm usage error")
	}

	if !strings.Contains(output, "drscmd drsrm") {
		t.Fatalf("expected drsrm help content on usage error, got %q", output)
	}

	if !strings.Contains(err.Error(), "an iRODS data object path is required") {
		t.Fatalf("expected drsrm usage error message, got %v", err)
	}
}

func runCommand(t *testing.T, cmd *cli.Command, args []string) string {
	t.Helper()

	output := &bytes.Buffer{}
	cmd.Writer = output
	cmd.ErrWriter = output
	for _, sc := range cmd.Commands {
		sc.Writer = output
		sc.ErrWriter = output
	}

	if err := cmd.Run(context.Background(), args); err != nil {
		t.Fatalf("cmd.Run(%v) failed: %v", args, err)
	}

	return output.String()
}

func runCommandAllowError(t *testing.T, cmd *cli.Command, args []string) (string, error) {
	t.Helper()

	output := &bytes.Buffer{}
	cmd.Writer = output
	cmd.ErrWriter = output
	for _, sc := range cmd.Commands {
		sc.Writer = output
		sc.ErrWriter = output
	}

	err := cmd.Run(context.Background(), args)
	return output.String(), err
}

type fakeFileSystem struct {
	account        *irodstypes.IRODSAccount
	entriesByPath  map[string]*irodsfs.Entry
	metadataByPath map[string][]*irodstypes.IRODSMeta
}

func newFakeFileSystem(objectPath string) *fakeFileSystem {
	return &fakeFileSystem{
		account: &irodstypes.IRODSAccount{
			Host:       "irods.example.org",
			Port:       1247,
			ClientUser: "rods",
			ClientZone: "tempZone",
			Password:   "rods-password",
		},
		entriesByPath: map[string]*irodsfs.Entry{
			objectPath: {
				ID:   1,
				Type: irodsfs.FileEntry,
				Name: "file.txt",
				Path: objectPath,
				Size: 128,
			},
		},
		metadataByPath: map[string][]*irodstypes.IRODSMeta{
			objectPath: {},
		},
	}
}

func (f *fakeFileSystem) GetHomeDirPath() string {
	return "/tempZone/home/rods"
}

func (f *fakeFileSystem) Release() {}

func (f *fakeFileSystem) StatFile(irodsPath string) (*irodsfs.Entry, error) {
	entry, ok := f.entriesByPath[irodsPath]
	if !ok {
		return nil, os.ErrNotExist
	}

	return entry, nil
}

func (f *fakeFileSystem) SearchByMeta(name string, value string) ([]*irodsfs.Entry, error) {
	matches := []*irodsfs.Entry{}

	for path, metas := range f.metadataByPath {
		for _, meta := range metas {
			if meta != nil && meta.Name == name && meta.Value == value && meta.Units == drs_support.DrsAvuUnit {
				matches = append(matches, f.entriesByPath[path])
				break
			}
		}
	}

	return matches, nil
}

func (f *fakeFileSystem) List(irodsPath string) ([]*irodsfs.Entry, error) {
	_ = irodsPath
	return []*irodsfs.Entry{}, nil
}

func (f *fakeFileSystem) ListMetadata(irodsPath string) ([]*irodstypes.IRODSMeta, error) {
	metas, ok := f.metadataByPath[irodsPath]
	if !ok {
		return nil, os.ErrNotExist
	}

	return append([]*irodstypes.IRODSMeta(nil), metas...), nil
}

func (f *fakeFileSystem) AddMetadata(irodsPath string, attName string, attValue string, attUnits string) error {
	f.metadataByPath[irodsPath] = append(f.metadataByPath[irodsPath], &irodstypes.IRODSMeta{
		Name:  attName,
		Value: attValue,
		Units: attUnits,
	})
	return nil
}

func (f *fakeFileSystem) DeleteMetadataByAVU(irodsPath string, attName string, attValue string, attUnits string) error {
	metas := f.metadataByPath[irodsPath]
	filtered := metas[:0]

	for _, meta := range metas {
		if meta != nil && meta.Name == attName && meta.Value == attValue && meta.Units == attUnits {
			continue
		}

		filtered = append(filtered, meta)
	}

	f.metadataByPath[irodsPath] = filtered
	return nil
}

func (f *fakeFileSystem) GetAccount() *irodstypes.IRODSAccount {
	return f.account
}
