//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"testing"
	"time"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

type e2eObjectFixture struct {
	rootPath     string
	objectPath   string
	objectID     string
	missingID    string
	description  string
	aliases      []string
	objectName   string
	expectedUser string
}

type e2eBulkObjectFixture struct {
	rootPath     string
	objectPaths  []string
	objectIDs    []string
	missingID    string
	expectedUser string
}

func requireE2EObjectFixture(t *testing.T) *e2eObjectFixture {
	t.Helper()
	fixture, err := buildE2EObjectFixture(t)
	if err != nil {
		t.Fatalf("build e2e object fixture: %v", err)
	}
	return fixture
}

func buildE2EObjectFixture(t *testing.T) (*e2eObjectFixture, error) {
	t.Helper()

	filesystem := newE2EIRODSFilesystem(t, requireE2EEffectiveUser(t))

	cfg := requireE2EIRODSConfig(t)
	testUser := filesystem.GetAccount().ClientUser
	rootPath := fmt.Sprintf("/%s/home/%s/drs-e2e-%d", cfg.IrodsZone, testUser, time.Now().UnixNano())
	if err := filesystem.MakeDir(rootPath, true); err != nil {
		filesystem.Release()
		return nil, fmt.Errorf("create e2e fixture root %q: %w", rootPath, err)
	}

	objectPath := path.Join(rootPath, "object.txt")
	content := []byte("drs e2e object\n")
	if _, err := filesystem.UploadFileFromBuffer(bytes.NewBuffer(content), objectPath, "", false, true, nil); err != nil {
		filesystem.Release()
		return nil, fmt.Errorf("upload e2e fixture object %q: %w", objectPath, err)
	}

	description := "e2e description"
	aliases := []string{"e2e-alias-one", "e2e-alias-two"}
	objectID, err := drs_support.CreateDrsObjectFromDataObject(filesystem, objectPath, "", description, aliases)
	if err != nil {
		filesystem.Release()
		return nil, fmt.Errorf("create DRS fixture object for %q: %w", objectPath, err)
	}

	t.Cleanup(func() {
		defer filesystem.Release()
		if err := filesystem.RemoveDir(rootPath, true, true); err != nil && filesystem.Exists(rootPath) {
			t.Errorf("cleanup e2e fixture root %q: %v", rootPath, err)
		}
	})

	return &e2eObjectFixture{
		rootPath:     rootPath,
		objectPath:   objectPath,
		objectID:     objectID,
		missingID:    fmt.Sprintf("missing-e2e-%d", time.Now().UnixNano()),
		description:  description,
		aliases:      aliases,
		objectName:   path.Base(objectPath),
		expectedUser: testUser,
	}, nil
}

func newE2EIRODSFilesystem(t *testing.T, effectiveUser string) *irodsfs.FileSystem {
	t.Helper()

	cfg := requireE2EIRODSConfig(t)
	effectiveUser = strings.TrimSpace(effectiveUser)
	if effectiveUser == "" {
		effectiveUser = strings.TrimSpace(cfg.IrodsPrimaryTestUser)
	}

	account, err := irodstypes.CreateIRODSProxyAccount(
		cfg.IrodsHost,
		cfg.IrodsPort,
		effectiveUser,
		cfg.IrodsZone,
		cfg.IrodsAdminUser,
		cfg.IrodsZone,
		irodstypes.GetAuthScheme(cfg.IrodsAuthScheme),
		cfg.IrodsAdminPassword,
		cfg.IrodsDefaultResource,
	)
	if err != nil {
		t.Fatalf("create e2e iRODS proxy account: %v", err)
	}

	filesystem, err := irodsfs.NewFileSystemWithDefault(account, "irods-go-drs-e2e-fixture")
	if err != nil {
		t.Fatalf("connect to iRODS for e2e fixture setup: %v", err)
	}

	return filesystem
}

func requireE2EBulkObjectFixture(t *testing.T) *e2eBulkObjectFixture {
	t.Helper()
	fixture, err := buildE2EBulkObjectFixture(t)
	if err != nil {
		t.Fatalf("build bulk e2e object fixture: %v", err)
	}
	return fixture
}

func buildE2EBulkObjectFixture(t *testing.T) (*e2eBulkObjectFixture, error) {
	t.Helper()

	filesystem := newE2EIRODSFilesystem(t, requireE2EEffectiveUser(t))

	cfg := requireE2EIRODSConfig(t)
	testUser := filesystem.GetAccount().ClientUser
	rootPath := fmt.Sprintf("/%s/home/%s/drs-bulk-e2e-%d", cfg.IrodsZone, testUser, time.Now().UnixNano())
	if err := filesystem.MakeDir(rootPath, true); err != nil {
		filesystem.Release()
		return nil, fmt.Errorf("create bulk e2e fixture root %q: %w", rootPath, err)
	}

	objectPaths := make([]string, 0, 3)
	objectIDs := make([]string, 0, 3)
	for i := 1; i <= 3; i++ {
		objectPath := path.Join(rootPath, fmt.Sprintf("object-%d.txt", i))
		content := []byte(fmt.Sprintf("drs bulk e2e object %d\n", i))
		if _, err := filesystem.UploadFileFromBuffer(bytes.NewBuffer(content), objectPath, "", false, true, nil); err != nil {
			filesystem.Release()
			return nil, fmt.Errorf("upload bulk e2e fixture object %q: %w", objectPath, err)
		}

		objectID, err := drs_support.CreateDrsObjectFromDataObject(filesystem, objectPath, "", fmt.Sprintf("bulk e2e description %d", i), []string{fmt.Sprintf("bulk-e2e-alias-%d", i)})
		if err != nil {
			filesystem.Release()
			return nil, fmt.Errorf("create bulk DRS fixture object for %q: %w", objectPath, err)
		}

		objectPaths = append(objectPaths, objectPath)
		objectIDs = append(objectIDs, objectID)
	}

	t.Cleanup(func() {
		defer filesystem.Release()
		if err := filesystem.RemoveDir(rootPath, true, true); err != nil && filesystem.Exists(rootPath) {
			t.Errorf("cleanup bulk e2e fixture root %q: %v", rootPath, err)
		}
	})

	return &e2eBulkObjectFixture{
		rootPath:     rootPath,
		objectPaths:  objectPaths,
		objectIDs:    objectIDs,
		missingID:    fmt.Sprintf("missing-bulk-e2e-%d", time.Now().UnixNano()),
		expectedUser: testUser,
	}, nil
}
