package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	irodsclientconfig "github.com/cyverse/go-irodsclient/config"
	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	extmetadata "github.com/michael-conway/go-irodsclient-extensions/metadata"
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

func TestDrsInfoIncludesManifestForCompoundObject(t *testing.T) {
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

	fakeFS := newFakeFileSystem("/tempZone/home/rods/file.txt")
	fakeFS.addCollection("/tempZone/home/rods/compound")
	fakeFS.addCollection("/tempZone/home/rods/compound/sub")
	fakeFS.addDataObject("/tempZone/home/rods/compound/sub/object.txt", 401)
	fakeFS.addDataObject("/tempZone/home/rods/compound/.drsignore", 402)
	fakeFS.fileContents["/tempZone/home/rods/compound/.drsignore"] = []byte("sub/**\n")

	if err := fakeFS.AddMetadata("/tempZone/home/rods/compound", drs_support.DrsIdAvuAttrib, "compound-id", drs_support.DrsAvuUnit); err != nil {
		t.Fatalf("seed compound root drs id: %v", err)
	}
	if err := fakeFS.AddMetadata("/tempZone/home/rods/compound", drs_support.DrsAvuCompoundManifestAttrib, "true", drs_support.DrsAvuUnit); err != nil {
		t.Fatalf("seed compound marker: %v", err)
	}
	if err := fakeFS.AddMetadata("/tempZone/home/rods/compound/sub/object.txt", drs_support.DrsIdAvuAttrib, "child-id", drs_support.DrsAvuUnit); err != nil {
		t.Fatalf("seed child drs id: %v", err)
	}

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()
	output := runCommand(t, cmd, []string{APP_NAME, "drsinfo", "--id", "compound-id"})

	if !strings.Contains(output, "\"isManifest\": true") {
		t.Fatalf("expected compound drsinfo result, got %q", output)
	}
	if !strings.Contains(output, "\"manifest\"") {
		t.Fatalf("expected embedded generated manifest for compound object, got %q", output)
	}
	if !strings.Contains(output, "\"rootPath\": \"/tempZone/home/rods/compound\"") {
		t.Fatalf("expected manifest root path, got %q", output)
	}
	if !strings.Contains(output, "\"path\": \"/tempZone/home/rods/compound/sub/object.txt\"") {
		t.Fatalf("expected manifest data object entry, got %q", output)
	}
	if strings.Contains(output, "/tempZone/home/rods/compound/.drsignore") {
		t.Fatalf("expected .drsignore excluded from generated manifest, got %q", output)
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

func TestDrsListScopeFlags(t *testing.T) {
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
	fakeFS.addCollection("/tempZone/home/rods/projects/bundle")
	fakeFS.addDataObject("/tempZone/home/rods/projects/alpha.txt", 106)

	if _, err := drs_support.CreateDrsObjectFromDataObject(fakeFS, "/tempZone/home/rods/projects/alpha.txt", "", "alpha object", nil); err != nil {
		t.Fatalf("create alpha DRS object: %v", err)
	}
	if err := fakeFS.AddMetadata("/tempZone/home/rods/projects/bundle", drs_support.DrsIdAvuAttrib, "compound-id", drs_support.DrsAvuUnit); err != nil {
		t.Fatalf("add compound DRS id: %v", err)
	}
	if err := fakeFS.AddMetadata("/tempZone/home/rods/projects/bundle", drs_support.DrsAvuCompoundManifestAttrib, "true", drs_support.DrsAvuUnit); err != nil {
		t.Fatalf("add compound marker: %v", err)
	}

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	allOutput := runCommand(t, getCommand(), []string{APP_NAME, "drsls", "/tempZone/home/rods/projects"})
	if !strings.Contains(allOutput, "\"total\": 2") ||
		!strings.Contains(allOutput, "\"path\": \"/tempZone/home/rods/projects/alpha.txt\"") ||
		!strings.Contains(allOutput, "\"path\": \"/tempZone/home/rods/projects/bundle\"") ||
		!strings.Contains(allOutput, "\"isBundle\": true") {
		t.Fatalf("expected default drsls scope to include objects and compound collections, got %q", allOutput)
	}

	objectsOutput := runCommand(t, getCommand(), []string{APP_NAME, "drsls", "--scope_objects", "/tempZone/home/rods/projects"})
	if !strings.Contains(objectsOutput, "\"total\": 1") ||
		!strings.Contains(objectsOutput, "\"path\": \"/tempZone/home/rods/projects/alpha.txt\"") ||
		strings.Contains(objectsOutput, "\"path\": \"/tempZone/home/rods/projects/bundle\"") {
		t.Fatalf("expected --scope_objects to include only data objects, got %q", objectsOutput)
	}

	compoundOutput := runCommand(t, getCommand(), []string{APP_NAME, "drsls", "--scope_compound", "/tempZone/home/rods/projects"})
	if !strings.Contains(compoundOutput, "\"total\": 1") ||
		!strings.Contains(compoundOutput, "\"path\": \"/tempZone/home/rods/projects/bundle\"") ||
		strings.Contains(compoundOutput, "\"path\": \"/tempZone/home/rods/projects/alpha.txt\"") {
		t.Fatalf("expected --scope_compound to include only compound collections, got %q", compoundOutput)
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

func TestAddDrsIgnoreHelp(t *testing.T) {
	cmd := getCommand()

	helpOutput := runCommand(t, cmd, []string{APP_NAME, "add_drsignore", "--help"})
	if !strings.Contains(helpOutput, "drscmd add_drsignore") {
		t.Fatalf("expected add_drsignore help header, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "<irods-collection-path>") {
		t.Fatalf("expected add_drsignore help to include argument usage, got %q", helpOutput)
	}
}

func TestDrsRemoveRemovesDrsAVUsFromSingleObject(t *testing.T) {
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
	if _, err := drs_support.CreateDrsObjectFromDataObject(fakeFS, "/tempZone/home/rods/file.txt", "text/plain", "single", []string{"a1"}); err != nil {
		t.Fatalf("create single drs object: %v", err)
	}
	if err := fakeFS.AddMetadata("/tempZone/home/rods/file.txt", "user:note", "keep", "custom"); err != nil {
		t.Fatalf("add custom metadata: %v", err)
	}

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()
	output := runCommand(t, cmd, []string{APP_NAME, "drsrm", "--path", "/tempZone/home/rods/file.txt"})

	if !strings.Contains(output, "\"path\": \"/tempZone/home/rods/file.txt\"") {
		t.Fatalf("expected drsrm output path, got %q", output)
	}
	if !strings.Contains(output, "\"pathsVisited\": 1") {
		t.Fatalf("expected pathsVisited=1, got %q", output)
	}

	metas, err := fakeFS.ListMetadata("/tempZone/home/rods/file.txt")
	if err != nil {
		t.Fatalf("list metadata after strip: %v", err)
	}
	for _, meta := range metas {
		if meta == nil {
			continue
		}
		if meta.Units == drs_support.DrsAvuUnit && strings.HasPrefix(meta.Name, "iRODS:DRS") {
			t.Fatalf("expected drs metadata removed from single object, got %+v", metas)
		}
	}
}

func TestDrsRemoveRemovesDrsAVUsFromCompoundCollection(t *testing.T) {
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
	fakeFS.addCollection("/tempZone/home/rods/compound")
	fakeFS.addCollection("/tempZone/home/rods/compound/sub")
	fakeFS.addDataObject("/tempZone/home/rods/compound/sub/object.txt", 301)
	fakeFS.addDataObject("/tempZone/home/rods/compound/.drsignore", 302)

	// root + child drs semantics
	if err := fakeFS.AddMetadata("/tempZone/home/rods/compound", drs_support.DrsIdAvuAttrib, "compound-id", drs_support.DrsAvuUnit); err != nil {
		t.Fatalf("seed root drs id: %v", err)
	}
	if err := fakeFS.AddMetadata("/tempZone/home/rods/compound", drs_support.DrsAvuCompoundManifestAttrib, "true", drs_support.DrsAvuUnit); err != nil {
		t.Fatalf("seed root compound marker: %v", err)
	}
	if _, err := drs_support.CreateDrsObjectFromDataObject(fakeFS, "/tempZone/home/rods/compound/sub/object.txt", "text/plain", "child", []string{"c1"}); err != nil {
		t.Fatalf("seed child drs object: %v", err)
	}
	if err := fakeFS.AddMetadata("/tempZone/home/rods/compound/.drsignore", "user:note", "keep", "custom"); err != nil {
		t.Fatalf("seed .drsignore custom metadata: %v", err)
	}

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()
	output := runCommand(t, cmd, []string{APP_NAME, "drsrm", "--id", "compound-id"})

	if !strings.Contains(output, "\"path\": \"/tempZone/home/rods/compound\"") {
		t.Fatalf("expected drsrm output root path, got %q", output)
	}
	if !strings.Contains(output, "\"pathsVisited\":") || !strings.Contains(output, "\"avusRemoved\":") {
		t.Fatalf("expected strip counters in output, got %q", output)
	}

	rootMetas, _ := fakeFS.ListMetadata("/tempZone/home/rods/compound")
	childMetas, _ := fakeFS.ListMetadata("/tempZone/home/rods/compound/sub/object.txt")
	for _, meta := range rootMetas {
		if meta != nil && meta.Units == drs_support.DrsAvuUnit && strings.HasPrefix(meta.Name, "iRODS:DRS") {
			t.Fatalf("expected root drs metadata removed, got %+v", rootMetas)
		}
	}
	for _, meta := range childMetas {
		if meta != nil && meta.Units == drs_support.DrsAvuUnit && strings.HasPrefix(meta.Name, "iRODS:DRS") {
			t.Fatalf("expected child drs metadata removed, got %+v", childMetas)
		}
	}

	ignoreMetas, _ := fakeFS.ListMetadata("/tempZone/home/rods/compound/.drsignore")
	foundCustom := false
	for _, meta := range ignoreMetas {
		if meta != nil && meta.Name == "user:note" && meta.Value == "keep" {
			foundCustom = true
			break
		}
	}
	if !foundCustom {
		t.Fatalf("expected .drsignore data object to remain with custom metadata, got %+v", ignoreMetas)
	}
}

func TestDrsMakeCompoundHelp(t *testing.T) {
	cmd := getCommand()

	helpOutput := runCommand(t, cmd, []string{APP_NAME, "drsmakecompound", "--help"})
	if !strings.Contains(helpOutput, "drscmd drsmakecompound") {
		t.Fatalf("expected drsmakecompound help header, got %q", helpOutput)
	}

	if !strings.Contains(helpOutput, "<irods-collection-path>") {
		t.Fatalf("expected drsmakecompound help to include argument usage, got %q", helpOutput)
	}
}

func TestDrsMakeCompoundRequiresOverrideWhenIgnoreMissing(t *testing.T) {
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
	fakeFS.addCollection("/tempZone/home/rods/compound")
	fakeFS.addDataObject("/tempZone/home/rods/compound/object.txt", 201)

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()
	_, err = runCommandAllowError(t, cmd, []string{APP_NAME, "drsmakecompound", "/tempZone/home/rods/compound"})
	if err == nil {
		t.Fatal("expected drsmakecompound to fail when .drsignore is missing")
	}

	if !strings.Contains(err.Error(), "--allow-no-ignore") {
		t.Fatalf("expected missing ignore override message, got %v", err)
	}
}

func TestDrsMakeCompoundPreflightWithoutIgnoreUsesOverride(t *testing.T) {
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
	fakeFS.addCollection("/tempZone/home/rods/compound")
	fakeFS.addDataObject("/tempZone/home/rods/compound/object.txt", 202)

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()
	output := runCommand(t, cmd, []string{
		APP_NAME, "drsmakecompound", "/tempZone/home/rods/compound",
		"--preflight", "--allow-no-ignore",
	})

	if !strings.Contains(output, "\"preflight\": true") {
		t.Fatalf("expected preflight response, got %q", output)
	}
	if !strings.Contains(output, "\"warning\": \"no .drsignore found at /tempZone/home/rods/compound/.drsignore\"") {
		t.Fatalf("expected missing ignore warning, got %q", output)
	}
	if !strings.Contains(output, "\"manifest\"") || !strings.Contains(output, "\"nodeType\": \"collection\"") {
		t.Fatalf("expected manifest output for preflight, got %q", output)
	}
	if !strings.Contains(output, "\"drsId\": \"\"") {
		t.Fatalf("expected blank drsId fields when ids are not assigned yet, got %q", output)
	}

	rootMetas := fakeFS.metadataByPath["/tempZone/home/rods/compound"]
	for _, meta := range rootMetas {
		if meta != nil && meta.Name == drs_support.DrsAvuCompoundManifestAttrib {
			t.Fatalf("expected preflight to avoid writes, got root metadata %+v", rootMetas)
		}
	}
}

func TestDrsMakeCompoundCreatesCompoundObjectWithIgnore(t *testing.T) {
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
	fakeFS.addCollection("/tempZone/home/rods/compound")
	fakeFS.addDataObject("/tempZone/home/rods/compound/object.txt", 203)

	fakeFS.entriesByPath["/tempZone/home/rods/compound/.drsignore"] = &irodsfs.Entry{
		Type: irodsfs.FileEntry,
		Name: ".drsignore",
		Path: "/tempZone/home/rods/compound/.drsignore",
		Size: 4,
	}
	fakeFS.metadataByPath["/tempZone/home/rods/compound/.drsignore"] = []*irodstypes.IRODSMeta{}
	fakeFS.fileContents["/tempZone/home/rods/compound/.drsignore"] = []byte("\n")

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()
	output := runCommand(t, cmd, []string{APP_NAME, "drsmakecompound", "/tempZone/home/rods/compound"})

	if !strings.Contains(output, "\"preflight\": false") {
		t.Fatalf("expected non-preflight response, got %q", output)
	}
	if !strings.Contains(output, "\"drsId\"") {
		t.Fatalf("expected drs id in response, got %q", output)
	}
	if strings.Contains(output, "\"excludedPaths\"") {
		t.Fatalf("expected excludedPaths to be preflight-only, got %q", output)
	}
	if strings.Contains(output, "\"manifest\"") {
		t.Fatalf("expected manifest payload to be preflight-only, got %q", output)
	}

	rootMetas := fakeFS.metadataByPath["/tempZone/home/rods/compound"]
	if len(rootMetas) == 0 {
		t.Fatalf("expected root metadata after drsmakecompound")
	}
	foundCompound := false
	for _, meta := range rootMetas {
		if meta != nil && meta.Name == drs_support.DrsAvuCompoundManifestAttrib {
			foundCompound = true
			break
		}
	}
	if !foundCompound {
		t.Fatalf("expected compound marker on root metadata, got %+v", rootMetas)
	}
}

func TestDrsMakeCompoundPreflightShowsExcludedPathsWhenIgnored(t *testing.T) {
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
	fakeFS.addCollection("/tempZone/home/rods/compound")
	fakeFS.addCollection("/tempZone/home/rods/compound/skip")
	fakeFS.addDataObject("/tempZone/home/rods/compound/keep.txt", 204)
	fakeFS.addDataObject("/tempZone/home/rods/compound/skip/ignored.txt", 205)
	fakeFS.entriesByPath["/tempZone/home/rods/compound/.drsignore"] = &irodsfs.Entry{
		Type: irodsfs.FileEntry,
		Name: ".drsignore",
		Path: "/tempZone/home/rods/compound/.drsignore",
		Size: 8,
	}
	fakeFS.metadataByPath["/tempZone/home/rods/compound/.drsignore"] = []*irodstypes.IRODSMeta{}
	fakeFS.fileContents["/tempZone/home/rods/compound/.drsignore"] = []byte("skip/**\n")

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()
	output := runCommand(t, cmd, []string{
		APP_NAME, "drsmakecompound", "/tempZone/home/rods/compound",
		"--preflight",
	})

	if !strings.Contains(output, "\"preflight\": true") {
		t.Fatalf("expected preflight response, got %q", output)
	}
	if !strings.Contains(output, "\"excludedPaths\"") {
		t.Fatalf("expected excluded paths in preflight when ignore rules exclude content, got %q", output)
	}
	if !strings.Contains(output, "/tempZone/home/rods/compound/skip") {
		t.Fatalf("expected excluded skip path in output, got %q", output)
	}
}

func TestAddDrsIgnoreCreatesTemplateFile(t *testing.T) {
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

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()
	output := runCommand(t, cmd, []string{APP_NAME, "add_drsignore", "/tempZone/home/rods/projects"})

	if !strings.Contains(output, "\"ignoreFile\": \"/tempZone/home/rods/projects/.drsignore\"") {
		t.Fatalf("expected add_drsignore output to include created path, got %q", output)
	}

	content := string(fakeFS.fileContents["/tempZone/home/rods/projects/.drsignore"])
	if !strings.Contains(content, "*.tmp") {
		t.Fatalf("expected created .drsignore to contain sample content, got %q", content)
	}
}

func TestAddDrsIgnoreFailsWhenTemplateExists(t *testing.T) {
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
	fakeFS.entriesByPath["/tempZone/home/rods/projects/.drsignore"] = &irodsfs.Entry{
		Type: irodsfs.FileEntry,
		Name: ".drsignore",
		Path: "/tempZone/home/rods/projects/.drsignore",
		Size: 10,
	}
	fakeFS.metadataByPath["/tempZone/home/rods/projects/.drsignore"] = []*irodstypes.IRODSMeta{}
	fakeFS.fileContents["/tempZone/home/rods/projects/.drsignore"] = []byte("existing")

	oldCreateFileSystem := createFileSystem
	createFileSystem = func(account *irodstypes.IRODSAccount, applicationName string) (FileSystem, error) {
		return fakeFS, nil
	}
	defer func() { createFileSystem = oldCreateFileSystem }()

	cmd := getCommand()
	_, err = runCommandAllowError(t, cmd, []string{APP_NAME, "add_drsignore", "/tempZone/home/rods/projects"})
	if err == nil {
		t.Fatal("expected add_drsignore to fail when .drsignore exists")
	}

	if !strings.Contains(err.Error(), ".drsignore already exists") {
		t.Fatalf("expected duplicate .drsignore error, got %v", err)
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
	fileContents   map[string][]byte
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
		fileContents: map[string][]byte{},
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

func (f *fakeFileSystem) Stat(irodsPath string) (*irodsfs.Entry, error) {
	return f.StatFile(irodsPath)
}

func (f *fakeFileSystem) QueryMetadataEntries(query extmetadata.EntryQuery) (extmetadata.EntryQueryResult, error) {
	normalized, err := extmetadata.NormalizeEntryQuery(query)
	if err != nil {
		return extmetadata.EntryQueryResult{}, err
	}

	result := extmetadata.EntryQueryResult{
		Entries:     []*extmetadata.Entry{},
		MatchedAVUs: map[string][]extmetadata.AVUStat{},
		Page: extmetadata.EntryQueryPage{
			Limit: normalized.Limit,
		},
	}

	for irodsPath, metas := range f.metadataByPath {
		entry := f.entriesByPath[irodsPath]
		if entry == nil || !fakeQueryIncludesEntryKind(normalized, entry) || !fakeQueryEntryInScope(normalized, entry) {
			continue
		}

		matched := fakeMatchedAVUsForQuery(normalized, metas)
		if len(matched) == 0 {
			continue
		}

		result.Entries = append(result.Entries, entry)
		result.MatchedAVUs[entry.Path] = matched
		if entry.IsDir() {
			result.Page.Returned.Collections++
			continue
		}
		result.Page.Returned.DataObjects++
	}

	return result, nil
}

func fakeQueryIncludesEntryKind(query extmetadata.EntryQuery, entry *irodsfs.Entry) bool {
	if entry.IsDir() {
		return extmetadata.EntryQueryHasKind(query, extmetadata.EntryKindCollection)
	}
	return extmetadata.EntryQueryHasKind(query, extmetadata.EntryKindDataObject)
}

func fakeMatchedAVUsForQuery(query extmetadata.EntryQuery, metas []*irodstypes.IRODSMeta) []extmetadata.AVUStat {
	matched := []extmetadata.AVUStat{}
	for _, meta := range metas {
		if meta == nil || !fakeMetaMatchesConditions(meta, query.Conditions) {
			continue
		}
		matched = append(matched, extmetadata.AVUStat{
			Name:  meta.Name,
			Value: meta.Value,
			Units: meta.Units,
		})
	}
	return matched
}

func fakeQueryEntryInScope(query extmetadata.EntryQuery, entry *irodsfs.Entry) bool {
	if query.Scope == nil || query.Scope.Mode == extmetadata.EntryQueryScopeAbsolute {
		return true
	}

	root := strings.TrimRight(query.Scope.Root, "/")
	if root == "" {
		root = "/"
	}
	entryPath := path.Clean(entry.Path)

	switch query.Scope.Mode {
	case extmetadata.EntryQueryScopeSelf:
		return entry.IsDir() && entryPath == root
	case extmetadata.EntryQueryScopeChildren:
		return path.Dir(entryPath) == root
	case extmetadata.EntryQueryScopeDescendants:
		if entry.IsDir() {
			return strings.HasPrefix(entryPath, root+"/")
		}
		return strings.HasPrefix(path.Dir(entryPath), root+"/")
	default:
		return true
	}
}

func fakeMetaMatchesConditions(meta *irodstypes.IRODSMeta, conditions []extmetadata.EntryCondition) bool {
	for _, condition := range conditions {
		var candidate string
		switch condition.Field {
		case extmetadata.FieldAVUAttrib:
			candidate = meta.Name
		case extmetadata.FieldAVUValue:
			candidate = meta.Value
		case extmetadata.FieldAVUUnit:
			candidate = meta.Units
		default:
			continue
		}

		if condition.Op == extmetadata.OpLike {
			pattern := strings.TrimSuffix(extmetadata.NormalizeLikePattern(condition.Value), "%")
			if !strings.HasPrefix(candidate, pattern) {
				return false
			}
			continue
		}
		if candidate != condition.Value {
			return false
		}
	}
	return true
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

func (f *fakeFileSystem) OpenFile(irodsPath string, resource string, mode string) (drs_support.IRODSReadWriteCloser, error) {
	_ = resource
	readOnly := strings.Contains(strings.ToLower(mode), "r") && !strings.Contains(strings.ToLower(mode), "w")
	if _, ok := f.entriesByPath[irodsPath]; !ok {
		return nil, os.ErrNotExist
	}
	return &fakeFileHandle{
		filesystem: f,
		path:       irodsPath,
		readOnly:   readOnly,
	}, nil
}

func (f *fakeFileSystem) CreateFile(irodsPath string, resource string, mode string) (drs_support.IRODSReadWriteCloser, error) {
	_ = resource
	_ = mode
	if _, exists := f.entriesByPath[irodsPath]; exists {
		return nil, os.ErrExist
	}
	parent := filepath.Dir(irodsPath)
	if parent == "." {
		parent = "/"
	}
	parentEntry, ok := f.entriesByPath[parent]
	if !ok || !parentEntry.IsDir() {
		return nil, os.ErrNotExist
	}
	f.entriesByPath[irodsPath] = &irodsfs.Entry{
		Type: irodsfs.FileEntry,
		Name: filepath.Base(irodsPath),
		Path: irodsPath,
		Size: 0,
	}
	f.metadataByPath[irodsPath] = []*irodstypes.IRODSMeta{}
	f.fileContents[irodsPath] = []byte{}
	return &fakeFileHandle{
		filesystem: f,
		path:       irodsPath,
	}, nil
}

func (f *fakeFileSystem) addCollection(collectionPath string) {
	f.entriesByPath[collectionPath] = &irodsfs.Entry{
		Type: irodsfs.DirectoryEntry,
		Name: filepath.Base(collectionPath),
		Path: collectionPath,
	}
	f.metadataByPath[collectionPath] = []*irodstypes.IRODSMeta{}
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

type fakeFileHandle struct {
	filesystem *fakeFileSystem
	path       string
	readOnly   bool
	closed     bool
}

func (h *fakeFileHandle) ReadAt(buffer []byte, offset int64) (int, error) {
	if h.closed {
		return 0, os.ErrClosed
	}
	content := h.filesystem.fileContents[h.path]
	if offset >= int64(len(content)) {
		return 0, io.EOF
	}
	read := copy(buffer, content[offset:])
	if int(offset)+read >= len(content) {
		return read, io.EOF
	}
	return read, nil
}

func (h *fakeFileHandle) Write(data []byte) (int, error) {
	if h.closed {
		return 0, os.ErrClosed
	}
	if h.readOnly {
		return 0, os.ErrPermission
	}
	h.filesystem.fileContents[h.path] = append(h.filesystem.fileContents[h.path], data...)
	if entry, ok := h.filesystem.entriesByPath[h.path]; ok {
		entry.Size = int64(len(h.filesystem.fileContents[h.path]))
	}
	return len(data), nil
}

func (h *fakeFileHandle) Close() error {
	h.closed = true
	return nil
}
