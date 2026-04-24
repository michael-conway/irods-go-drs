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

	updateOutput := runCommand(t, cmd, []string{APP_NAME, "drsupdate", "--id", object.Id, "description", "updated description"})
	if !strings.Contains(updateOutput, "\"item\": \"description\"") || !strings.Contains(updateOutput, "\"value\": \"updated description\"") {
		t.Fatalf("expected drsupdate output to report updated description, got %q", updateOutput)
	}

	updatedInfo := runCommand(t, cmd, []string{APP_NAME, "drsinfo", "--path", objectPath})
	if !strings.Contains(updatedInfo, "\"description\": \"updated description\"") {
		t.Fatalf("expected drsinfo output to reflect updated description, got %q", updatedInfo)
	}

	removeOutput := runCommand(t, cmd, []string{APP_NAME, "drsrm", "--id", object.Id})
	if !strings.Contains(removeOutput, "\"path\": \""+objectPath+"\"") {
		t.Fatalf("expected drsrm output to contain the removed path, got %q", removeOutput)
	}

	if _, err := drs_support.GetDrsObjectByIRODSPath(fakeFS, objectPath); err == nil {
		t.Fatal("expected DRS metadata to be removed from the fake filesystem")
	}
}

func TestDrsListDefaultsToSessionCwdAndPagesResults(t *testing.T) {
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
		Home:                 "/tempZone/home/rods",
	}
	testEnvManager.Session = &irodsclientconfig.Config{
		CurrentWorkingDir: "/tempZone/home/rods/projects/demo",
	}

	oldEnvManager := envManager
	envManager = testEnvManager
	defer func() { envManager = oldEnvManager }()

	fakeFS := newFakeFileSystem("/tempZone/home/rods/file.txt")
	fakeFS.addCollection("/tempZone/home/rods/projects/demo")
	fakeFS.addDataObject("/tempZone/home/rods/projects/demo/alpha.txt", 101)
	fakeFS.addDataObject("/tempZone/home/rods/projects/demo/beta.txt", 102)

	if _, err := drs_support.CreateDrsObjectFromDataObject(fakeFS, "/tempZone/home/rods/projects/demo/alpha.txt", "", "alpha object", nil); err != nil {
		t.Fatalf("create alpha DRS object: %v", err)
	}
	if _, err := drs_support.CreateDrsObjectFromDataObject(fakeFS, "/tempZone/home/rods/projects/demo/beta.txt", "", "beta object", nil); err != nil {
		t.Fatalf("create beta DRS object: %v", err)
	}

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()
	output := runCommand(t, cmd, []string{APP_NAME, "drsls", "--limit", "1"})

	if !strings.Contains(output, "\"path\": \"/tempZone/home/rods/projects/demo\"") {
		t.Fatalf("expected drsls to list from session cwd, got %q", output)
	}

	if !strings.Contains(output, "\"total\": 2") || !strings.Contains(output, "\"hasMore\": true") {
		t.Fatalf("expected drsls paging metadata, got %q", output)
	}

	if !strings.Contains(output, "\"path\": \"/tempZone/home/rods/projects/demo/alpha.txt\"") {
		t.Fatalf("expected first paged result to contain alpha.txt, got %q", output)
	}

	if strings.Contains(output, "\"path\": \"/tempZone/home/rods/projects/demo/beta.txt\"") {
		t.Fatalf("expected second result to be omitted by limit, got %q", output)
	}

	if !strings.Contains(output, "\"description\": \"alpha object\"") {
		t.Fatalf("expected drsls output to include description, got %q", output)
	}

	if !strings.Contains(output, "\"isBundle\": false") {
		t.Fatalf("expected drsls output to include isBundle false for atomic objects, got %q", output)
	}
}

func TestDrsListUsesProvidedCollectionPath(t *testing.T) {
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
		Home:                 "/tempZone/home/rods",
	}
	testEnvManager.Session = &irodsclientconfig.Config{
		CurrentWorkingDir: "/tempZone/home/rods",
	}

	oldEnvManager := envManager
	envManager = testEnvManager
	defer func() { envManager = oldEnvManager }()

	fakeFS := newFakeFileSystem("/tempZone/home/rods/file.txt")
	fakeFS.addCollection("/tempZone/home/rods/projects")
	fakeFS.addCollection("/tempZone/home/rods/projects/demo")
	fakeFS.addDataObject("/tempZone/home/rods/projects/demo/gamma.txt", 103)

	if _, err := drs_support.CreateDrsObjectFromDataObject(fakeFS, "/tempZone/home/rods/projects/demo/gamma.txt", "", "gamma object", nil); err != nil {
		t.Fatalf("create gamma DRS object: %v", err)
	}

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()
	output := runCommand(t, cmd, []string{APP_NAME, "drsls", "projects/demo"})

	if !strings.Contains(output, "\"path\": \"/tempZone/home/rods/projects/demo\"") {
		t.Fatalf("expected drsls to resolve relative collection path, got %q", output)
	}

	if !strings.Contains(output, "\"drsId\"") || !strings.Contains(output, "\"description\": \"gamma object\"") {
		t.Fatalf("expected drsls output to contain DRS fields, got %q", output)
	}

	if !strings.Contains(output, "\"isBundle\": false") {
		t.Fatalf("expected drsls output to include isBundle false for atomic objects, got %q", output)
	}
}

func TestDrsListRecursiveIncludesChildCollections(t *testing.T) {
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
		Home:                 "/tempZone/home/rods",
	}
	testEnvManager.Session = &irodsclientconfig.Config{
		CurrentWorkingDir: "/tempZone/home/rods",
	}

	oldEnvManager := envManager
	envManager = testEnvManager
	defer func() { envManager = oldEnvManager }()

	fakeFS := newFakeFileSystem("/tempZone/home/rods/file.txt")
	fakeFS.addCollection("/tempZone/home/rods/projects")
	fakeFS.addCollection("/tempZone/home/rods/projects/demo")
	fakeFS.addDataObject("/tempZone/home/rods/projects/root.txt", 104)
	fakeFS.addDataObject("/tempZone/home/rods/projects/demo/child.txt", 105)

	if _, err := drs_support.CreateDrsObjectFromDataObject(fakeFS, "/tempZone/home/rods/projects/root.txt", "", "root object", nil); err != nil {
		t.Fatalf("create root DRS object: %v", err)
	}
	if _, err := drs_support.CreateDrsObjectFromDataObject(fakeFS, "/tempZone/home/rods/projects/demo/child.txt", "", "child object", nil); err != nil {
		t.Fatalf("create child DRS object: %v", err)
	}

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()

	nonRecursiveOutput := runCommand(t, cmd, []string{APP_NAME, "drsls", "/tempZone/home/rods/projects"})
	if !strings.Contains(nonRecursiveOutput, "\"path\": \"/tempZone/home/rods/projects/root.txt\"") {
		t.Fatalf("expected non-recursive drsls to include direct child DRS object, got %q", nonRecursiveOutput)
	}
	if strings.Contains(nonRecursiveOutput, "\"path\": \"/tempZone/home/rods/projects/demo/child.txt\"") {
		t.Fatalf("expected non-recursive drsls to exclude nested DRS object, got %q", nonRecursiveOutput)
	}

	recursiveOutput := runCommand(t, cmd, []string{APP_NAME, "drsls", "--recursive", "/tempZone/home/rods/projects"})
	if !strings.Contains(recursiveOutput, "\"path\": \"/tempZone/home/rods/projects/root.txt\"") {
		t.Fatalf("expected recursive drsls to include direct child DRS object, got %q", recursiveOutput)
	}
	if !strings.Contains(recursiveOutput, "\"path\": \"/tempZone/home/rods/projects/demo/child.txt\"") {
		t.Fatalf("expected recursive drsls to include nested DRS object, got %q", recursiveOutput)
	}
}

func TestDrsUpdateByPathUpdatesSupportedFields(t *testing.T) {
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
	if _, err := drs_support.CreateDrsObjectFromDataObject(fakeFS, objectPath, "text/plain", "initial description", nil); err != nil {
		t.Fatalf("create DRS object: %v", err)
	}

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()

	versionOutput := runCommand(t, cmd, []string{APP_NAME, "drsupdate", "--path", objectPath, "version", "v2"})
	if !strings.Contains(versionOutput, "\"item\": \"version\"") || !strings.Contains(versionOutput, "\"value\": \"v2\"") {
		t.Fatalf("expected drsupdate output to report version update, got %q", versionOutput)
	}

	mimeOutput := runCommand(t, cmd, []string{APP_NAME, "drsupdate", "--path", objectPath, "mimeType", "application/json"})
	if !strings.Contains(mimeOutput, "\"item\": \"mimeType\"") || !strings.Contains(mimeOutput, "\"value\": \"application/json\"") {
		t.Fatalf("expected drsupdate output to report mimeType update, got %q", mimeOutput)
	}

	info := runCommand(t, cmd, []string{APP_NAME, "drsinfo", "--path", objectPath})
	if !strings.Contains(info, "\"version\": \"v2\"") {
		t.Fatalf("expected drsinfo output to reflect updated version, got %q", info)
	}
	if !strings.Contains(info, "\"mimeType\": \"application/json\"") {
		t.Fatalf("expected drsinfo output to reflect updated mimeType, got %q", info)
	}
}

func TestDrsUpdateAliasReplacesAliasSet(t *testing.T) {
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
	if _, err := drs_support.CreateDrsObjectFromDataObject(fakeFS, objectPath, "text/plain", "initial description", []string{"alias-1", "alias-2"}); err != nil {
		t.Fatalf("create DRS object: %v", err)
	}

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()

	updateOutput := runCommand(t, cmd, []string{APP_NAME, "drsupdate", "--path", objectPath, "alias", "-a", "alias-2", "-a", "alias-3"})
	if !strings.Contains(updateOutput, "\"item\": \"alias\"") {
		t.Fatalf("expected drsupdate alias output to report alias item, got %q", updateOutput)
	}
	if !strings.Contains(updateOutput, "\"alias-2\"") || !strings.Contains(updateOutput, "\"alias-3\"") {
		t.Fatalf("expected drsupdate alias output to report replacement set, got %q", updateOutput)
	}

	info := runCommand(t, cmd, []string{APP_NAME, "drsinfo", "--path", objectPath})
	if strings.Contains(info, "\"alias-1\"") {
		t.Fatalf("expected omitted alias to be removed, got %q", info)
	}
	if !strings.Contains(info, "\"alias-2\"") || !strings.Contains(info, "\"alias-3\"") {
		t.Fatalf("expected retained aliases to remain after update, got %q", info)
	}
}

func TestDrsListReportsBundleState(t *testing.T) {
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
		Home:                 "/tempZone/home/rods",
	}
	testEnvManager.Session = &irodsclientconfig.Config{
		CurrentWorkingDir: "/tempZone/home/rods/projects/demo",
	}

	oldEnvManager := envManager
	envManager = testEnvManager
	defer func() { envManager = oldEnvManager }()

	fakeFS := newFakeFileSystem("/tempZone/home/rods/file.txt")
	fakeFS.addCollection("/tempZone/home/rods/projects/demo")
	fakeFS.addDataObject("/tempZone/home/rods/projects/demo/atomic.txt", 106)
	fakeFS.addDataObject("/tempZone/home/rods/projects/demo/bundle.json", 107)

	if _, err := drs_support.CreateDrsObjectFromDataObject(fakeFS, "/tempZone/home/rods/projects/demo/atomic.txt", "", "atomic object", nil); err != nil {
		t.Fatalf("create atomic DRS object: %v", err)
	}
	if _, err := drs_support.CreateDrsObjectFromDataObject(fakeFS, "/tempZone/home/rods/projects/demo/bundle.json", "application/json", "bundle object", nil); err != nil {
		t.Fatalf("create bundle DRS object: %v", err)
	}

	if err := fakeFS.AddMetadata("/tempZone/home/rods/projects/demo/bundle.json", drs_support.DrsAvuCompoundManifestAttrib, "true", drs_support.DrsAvuUnit); err != nil {
		t.Fatalf("add manifest metadata: %v", err)
	}

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()
	output := runCommand(t, cmd, []string{APP_NAME, "drsls"})

	if !strings.Contains(output, "\"path\": \"/tempZone/home/rods/projects/demo/atomic.txt\"") || !strings.Contains(output, "\"isBundle\": false") {
		t.Fatalf("expected atomic object to report isBundle false, got %q", output)
	}

	if !strings.Contains(output, "\"path\": \"/tempZone/home/rods/projects/demo/bundle.json\"") || !strings.Contains(output, "\"isBundle\": true") {
		t.Fatalf("expected bundle object to report isBundle true, got %q", output)
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

func TestDrsListHelp(t *testing.T) {
	cmd := getCommand()

	helpOutput := runCommand(t, cmd, []string{APP_NAME, "drsls", "--help"})
	if !strings.Contains(helpOutput, "drscmd drsls") {
		t.Fatalf("expected drsls help header, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "[irods-collection-path]") {
		t.Fatalf("expected drsls help to include optional collection path, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "--offset") || !strings.Contains(helpOutput, "--limit") || !strings.Contains(helpOutput, "--recursive") {
		t.Fatalf("expected drsls help to include paging and recursive flags, got %q", helpOutput)
	}
}

func TestDrsUpdateHelp(t *testing.T) {
	cmd := getCommand()

	helpOutput := runCommand(t, cmd, []string{APP_NAME, "drsupdate", "--help"})
	if !strings.Contains(helpOutput, "drscmd drsupdate") {
		t.Fatalf("expected drsupdate help header, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "<path-or-drs-id> <item> <value>") {
		t.Fatalf("expected drsupdate help to include argument usage, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "mimeType") || !strings.Contains(helpOutput, "version") || !strings.Contains(helpOutput, "description") || !strings.Contains(helpOutput, "alias") {
		t.Fatalf("expected drsupdate help to include supported items, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "--alias") {
		t.Fatalf("expected drsupdate help to include alias flag, got %q", helpOutput)
	}
}

func TestDrsRemoveHelp(t *testing.T) {
	cmd := getCommand()

	helpOutput := runCommand(t, cmd, []string{APP_NAME, "drsrm", "--help"})
	if !strings.Contains(helpOutput, "drscmd drsrm") {
		t.Fatalf("expected drsrm help header, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "<path-or-drs-id>") {
		t.Fatalf("expected drsrm help to include argument usage, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "--path") || !strings.Contains(helpOutput, "--id") {
		t.Fatalf("expected drsrm help to include selector flags, got %q", helpOutput)
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

func TestDrsUpdateUsageErrorShowsHelp(t *testing.T) {
	cmd := getCommand()

	output, err := runCommandAllowError(t, cmd, []string{APP_NAME, "drsupdate", "target", "description"})
	if err == nil {
		t.Fatal("expected drsupdate usage error")
	}

	if !strings.Contains(output, "drscmd drsupdate") {
		t.Fatalf("expected drsupdate help content on usage error, got %q", output)
	}

	if !strings.Contains(err.Error(), "a value is required for this item") {
		t.Fatalf("expected drsupdate usage error message, got %v", err)
	}
}

func TestDrsUpdateConflictingFlagsShowHelp(t *testing.T) {
	cmd := getCommand()

	output, err := runCommandAllowError(t, cmd, []string{APP_NAME, "drsupdate", "--path", "--id", "value", "description", "x"})
	if err == nil {
		t.Fatal("expected drsupdate conflicting flags error")
	}

	if !strings.Contains(output, "drscmd drsupdate") {
		t.Fatalf("expected drsupdate help content on conflicting flags, got %q", output)
	}

	if !strings.Contains(err.Error(), "--path and --id cannot be used together") {
		t.Fatalf("expected drsupdate conflicting flags message, got %v", err)
	}
}

func TestDrsUpdateFailsForUnsupportedItem(t *testing.T) {
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
	if _, err := drs_support.CreateDrsObjectFromDataObject(fakeFS, objectPath, "text/plain", "initial description", nil); err != nil {
		t.Fatalf("create DRS object: %v", err)
	}

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()
	_, err = runCommandAllowError(t, cmd, []string{APP_NAME, "drsupdate", "--path", objectPath, "bogus", "x"})
	if err == nil {
		t.Fatal("expected drsupdate unsupported item error")
	}

	if !strings.Contains(err.Error(), "unsupported DRS metadata field") {
		t.Fatalf("expected unsupported item error, got %v", err)
	}
}

func TestDrsUpdateAliasRejectsPositionalValue(t *testing.T) {
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
	if _, err := drs_support.CreateDrsObjectFromDataObject(fakeFS, objectPath, "text/plain", "initial description", nil); err != nil {
		t.Fatalf("create DRS object: %v", err)
	}

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()
	output, err := runCommandAllowError(t, cmd, []string{APP_NAME, "drsupdate", "--path", objectPath, "alias", "x"})
	if err == nil {
		t.Fatal("expected drsupdate alias positional value error")
	}

	if !strings.Contains(output, "drscmd drsupdate") {
		t.Fatalf("expected drsupdate help content on alias positional value error, got %q", output)
	}

	if !strings.Contains(err.Error(), "alias updates use -a/--alias instead of a positional value") {
		t.Fatalf("expected alias positional value message, got %v", err)
	}
}

func TestDrsUpdateFailsForNonDrsObject(t *testing.T) {
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
	_, err = runCommandAllowError(t, cmd, []string{APP_NAME, "drsupdate", "--path", objectPath, "description", "x"})
	if err == nil {
		t.Fatal("expected drsupdate non-DRS error")
	}

	if !strings.Contains(err.Error(), "is not a DRS object") {
		t.Fatalf("expected non-DRS error, got %v", err)
	}
}

func TestDrsListUsageErrorShowsHelp(t *testing.T) {
	cmd := getCommand()

	output, err := runCommandAllowError(t, cmd, []string{APP_NAME, "drsls", "--limit", "0"})
	if err == nil {
		t.Fatal("expected drsls usage error")
	}

	if !strings.Contains(output, "error") && !strings.Contains(err.Error(), "limit must be greater than zero") {
		t.Fatalf("expected drsls limit error, got output=%q err=%v", output, err)
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

	if !strings.Contains(err.Error(), "a DRS id or iRODS path is required") {
		t.Fatalf("expected drsrm usage error message, got %v", err)
	}
}

func TestDrsRemoveConflictingFlagsShowHelp(t *testing.T) {
	cmd := getCommand()

	output, err := runCommandAllowError(t, cmd, []string{APP_NAME, "drsrm", "--path", "--id", "value"})
	if err == nil {
		t.Fatal("expected drsrm conflicting flags error")
	}

	if !strings.Contains(output, "drscmd drsrm") {
		t.Fatalf("expected drsrm help content on conflicting flags, got %q", output)
	}

	if !strings.Contains(err.Error(), "--path and --id cannot be used together") {
		t.Fatalf("expected drsrm conflicting flags message, got %v", err)
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
	irodsPath = strings.TrimSuffix(irodsPath, "/")
	if irodsPath == "" {
		irodsPath = "/"
	}

	results := []*irodsfs.Entry{}
	for path, entry := range f.entriesByPath {
		if path == irodsPath {
			continue
		}

		parent := filepath.Dir(path)
		if parent == "." {
			parent = "/"
		}

		if strings.ReplaceAll(parent, string(os.PathSeparator), "/") == irodsPath {
			results = append(results, entry)
		}
	}

	return results, nil
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

func (f *fakeFileSystem) addCollection(collectionPath string) {
	f.entriesByPath[collectionPath] = &irodsfs.Entry{
		Type: irodsfs.DirectoryEntry,
		Name: filepath.Base(collectionPath),
		Path: collectionPath,
	}
}

func (f *fakeFileSystem) addDataObject(objectPath string, id int64) {
	f.entriesByPath[objectPath] = &irodsfs.Entry{
		ID:   id,
		Type: irodsfs.FileEntry,
		Name: filepath.Base(objectPath),
		Path: objectPath,
		Size: 128,
	}
	f.metadataByPath[objectPath] = []*irodstypes.IRODSMeta{}
}
