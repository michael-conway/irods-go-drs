//go:build integration
// +build integration

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
)

type drscmdMakeOutput struct {
	DRSID string `json:"drsId"`
	Path  string `json:"path"`
}

type drscmdInfoOutput struct {
	DRSID       string   `json:"drsId"`
	Path        string   `json:"path"`
	Zone        string   `json:"zone,omitempty"`
	Size        int64    `json:"size,omitempty"`
	Description string   `json:"description,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
}

type drscmdRemoveOutput struct {
	Path string `json:"path"`
}

func TestDrsCmdIntegrationHarness(t *testing.T) {
	if _, err := exec.LookPath("gocmd"); err != nil {
		t.Skip("gocmd is not installed or not on PATH")
	}

	repoRoot := integrationRepoRoot(t)
	tempHome := t.TempDir()
	gocache := filepath.Join(t.TempDir(), "gocache")
	if err := os.MkdirAll(gocache, 0700); err != nil {
		t.Fatalf("make gocache dir: %v", err)
	}

	drscmdBinary := filepath.Join(t.TempDir(), "drscmd")
	buildCmd := exec.Command("go", "build", "-o", drscmdBinary, "./tools/drs-console")
	buildCmd.Dir = repoRoot
	buildCmd.Env = integrationCommandEnv(tempHome, gocache)
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build drscmd: %v\n%s", err, string(buildOutput))
	}

	filesystem := newToolIntegrationIRODSFilesystem(t)
	defer filesystem.Release()

	testDir := makeToolIntegrationTestDir(t, filesystem)
	objectPath := testDir + "/USERGUIDE.md"
	content := []byte("# drscmd cli integration fixture\n")
	if _, err := filesystem.UploadFileFromBuffer(bytes.NewBuffer(content), objectPath, "", false, true, nil); err != nil {
		t.Fatalf("upload object %q: %v", objectPath, err)
	}

	runIntegrationCommand(t, tempHome, gocache, drscmdBinary,
		"iinit",
		"-h", integrationIRODSHost(),
		"-o", fmt.Sprintf("%d", integrationIRODSPort(t)),
		"-u", integrationIRODSUser(),
		"-z", integrationIRODSZone(),
		"-p", integrationIRODSPassword(),
		"-t", "native",
	)

	runIntegrationCommand(t, tempHome, gocache, "gocmd", "cd", testDir)

	makeOutputText := runIntegrationCommand(t, tempHome, gocache, drscmdBinary, "drsmake", "USERGUIDE.md", "-d", "test drs1 object")
	var makeOutput drscmdMakeOutput
	decodeCommandJSON(t, makeOutputText, &makeOutput)
	if makeOutput.DRSID == "" {
		t.Fatal("expected drsmake to return a drs id")
	}
	if makeOutput.Path != objectPath {
		t.Fatalf("expected drsmake path %q, got %q", objectPath, makeOutput.Path)
	}

	infoByIDText := runIntegrationCommand(t, tempHome, gocache, drscmdBinary, "drsinfo", makeOutput.DRSID)
	var infoByID drscmdInfoOutput
	decodeCommandJSON(t, infoByIDText, &infoByID)
	if infoByID.DRSID != makeOutput.DRSID {
		t.Fatalf("expected drsinfo by id to return drs id %q, got %q", makeOutput.DRSID, infoByID.DRSID)
	}
	if infoByID.Path != objectPath {
		t.Fatalf("expected drsinfo by id path %q, got %q", objectPath, infoByID.Path)
	}

	infoByPathText := runIntegrationCommand(t, tempHome, gocache, drscmdBinary, "drsinfo", "--path", "USERGUIDE.md")
	var infoByPath drscmdInfoOutput
	decodeCommandJSON(t, infoByPathText, &infoByPath)
	if infoByPath.DRSID != makeOutput.DRSID {
		t.Fatalf("expected drsinfo by path to return drs id %q, got %q", makeOutput.DRSID, infoByPath.DRSID)
	}
	if infoByPath.Path != objectPath {
		t.Fatalf("expected drsinfo by path %q, got %q", objectPath, infoByPath.Path)
	}

	removeOutputText := runIntegrationCommand(t, tempHome, gocache, drscmdBinary, "drsrm", "USERGUIDE.md")
	var removeOutput drscmdRemoveOutput
	decodeCommandJSON(t, removeOutputText, &removeOutput)
	if removeOutput.Path != objectPath {
		t.Fatalf("expected drsrm path %q, got %q", objectPath, removeOutput.Path)
	}
}

func integrationRepoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}

	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func integrationCommandEnv(home string, gocache string) []string {
	env := os.Environ()
	env = append(env, "HOME="+home)
	env = append(env, "GOCACHE="+gocache)
	return env
}

func runIntegrationCommand(t *testing.T, home string, gocache string, binary string, args ...string) string {
	t.Helper()

	cmd := exec.Command(binary, args...)
	cmd.Env = integrationCommandEnv(home, gocache)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run %s %s: %v\n%s", binary, strings.Join(args, " "), err, string(output))
	}

	return string(output)
}

func decodeCommandJSON(t *testing.T, output string, target any) {
	t.Helper()

	start := strings.Index(output, "{")
	if start < 0 {
		t.Fatalf("expected JSON object in output, got %q", output)
	}

	if err := json.Unmarshal([]byte(output[start:]), target); err != nil {
		t.Fatalf("decode command json from %q: %v", output, err)
	}
}

func newToolIntegrationIRODSFilesystem(t *testing.T) *irodsfs.FileSystem {
	t.Helper()

	account, err := irodstypes.CreateIRODSAccount(
		integrationIRODSHost(),
		integrationIRODSPort(t),
		integrationIRODSUser(),
		integrationIRODSZone(),
		irodstypes.AuthSchemeNative,
		integrationIRODSPassword(),
		"",
	)
	if err != nil {
		t.Fatalf("create iRODS account: %v", err)
	}

	filesystem, err := irodsfs.NewFileSystemWithDefault(account, "irods-go-drs-drscmd-integration-test")
	if err != nil {
		t.Fatalf("connect to iRODS: %v", err)
	}

	return filesystem
}

func makeToolIntegrationTestDir(t *testing.T, filesystem *irodsfs.FileSystem) string {
	t.Helper()

	testDir := fmt.Sprintf("/%s/home/%s/drscmd-integration-%d", integrationIRODSZone(), integrationIRODSUser(), time.Now().UnixNano())
	if err := filesystem.MakeDir(testDir, true); err != nil {
		t.Fatalf("make dir %q: %v", testDir, err)
	}

	t.Cleanup(func() {
		if err := filesystem.RemoveDir(testDir, true, true); err != nil && filesystem.Exists(testDir) {
			t.Errorf("cleanup dir %q: %v", testDir, err)
		}
	})

	return testDir
}

func integrationIRODSHost() string {
	return getenvDefault("DRS_TEST_IRODS_HOST", "localhost")
}

func integrationIRODSPort(t *testing.T) int {
	t.Helper()

	raw := getenvDefault("DRS_TEST_IRODS_PORT", "1247")
	port, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("invalid DRS_TEST_IRODS_PORT %q: %v", raw, err)
	}

	return port
}

func integrationIRODSZone() string {
	return getenvDefault("DRS_TEST_IRODS_ZONE", "tempZone")
}

func integrationIRODSUser() string {
	return getenvDefault("DRS_TEST_IRODS_USER", "test1")
}

func integrationIRODSPassword() string {
	return getenvDefault("DRS_TEST_IRODS_PASSWORD", "test")
}

func getenvDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}
