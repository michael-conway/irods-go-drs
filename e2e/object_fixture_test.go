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
	extmetadata "github.com/michael-conway/go-irodsclient-extensions/metadata"
	extmetadatairodsfs "github.com/michael-conway/go-irodsclient-extensions/metadata/irodsfs"
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

type e2eBasicObjectFixture struct {
	rootPath     string
	objectPath   string
	objectID     string
	description  string
	objectName   string
	expectedUser string
}

type e2eCompoundObjectFixture struct {
	rootPath        string
	compoundPath    string
	compoundID      string
	ignoreFilePath  string
	childObjectPath string
	childObjectID   string
	childAVUName    string
	childAVUValue   string
	childAVUUnit    string
	ignoredPath     string
	expectedUser    string
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
		objectName:   objectPath,
		expectedUser: testUser,
	}, nil
}

type e2eIRODSFilesystem struct {
	*irodsfs.FileSystem
}

func (f *e2eIRODSFilesystem) QueryMetadataEntries(query extmetadata.EntryQuery) (extmetadata.EntryQueryResult, error) {
	return extmetadatairodsfs.NewAdapter(f.FileSystem).QueryEntries(query)
}

func newE2EIRODSFilesystem(t *testing.T, effectiveUser string) *e2eIRODSFilesystem {
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

	return &e2eIRODSFilesystem{FileSystem: filesystem}
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

func requireE2EBasicObjectFixture(t *testing.T) *e2eBasicObjectFixture {
	t.Helper()
	fixture, err := buildE2EBasicObjectFixture(t)
	if err != nil {
		t.Fatalf("build basic e2e object fixture: %v", err)
	}
	return fixture
}

func requireE2ECompoundObjectFixture(t *testing.T) *e2eCompoundObjectFixture {
	t.Helper()
	fixture, err := buildE2ECompoundObjectFixture(t)
	if err != nil {
		t.Fatalf("build compound e2e object fixture: %v", err)
	}
	return fixture
}

func buildE2EBasicObjectFixture(t *testing.T) (*e2eBasicObjectFixture, error) {
	t.Helper()

	cfg := requireE2EIRODSConfig(t)
	testUser := strings.TrimSpace(cfg.IrodsPrimaryTestUser)
	filesystem := newE2EIRODSFilesystem(t, testUser)

	rootPath := fmt.Sprintf("/%s/home/%s/drs-basic-e2e", cfg.IrodsZone, testUser)
	if err := filesystem.MakeDir(rootPath, true); err != nil {
		filesystem.Release()
		return nil, fmt.Errorf("create basic e2e fixture root %q: %w", rootPath, err)
	}

	objectPath := path.Join(rootPath, "object.txt")
	if !filesystem.Exists(objectPath) {
		content := []byte("drs basic e2e object\n")
		if _, err := filesystem.UploadFileFromBuffer(bytes.NewBuffer(content), objectPath, "", false, true, nil); err != nil {
			filesystem.Release()
			return nil, fmt.Errorf("upload basic e2e fixture object %q: %w", objectPath, err)
		}
	}

	description := "basic e2e description"
	var objectID string
	if object, err := drs_support.GetDrsObjectByIRODSPath(filesystem, objectPath); err == nil && object != nil && strings.TrimSpace(object.Id) != "" {
		objectID = strings.TrimSpace(object.Id)
	} else {
		objectID, err = drs_support.CreateDrsObjectFromDataObject(filesystem, objectPath, "", description, []string{"basic-e2e-alias"})
		if err != nil {
			filesystem.Release()
			return nil, fmt.Errorf("create basic DRS fixture object for %q: %w", objectPath, err)
		}
	}

	filesystem.Release()

	return &e2eBasicObjectFixture{
		rootPath:     rootPath,
		objectPath:   objectPath,
		objectID:     objectID,
		description:  description,
		objectName:   objectPath,
		expectedUser: testUser,
	}, nil
}

func buildE2ECompoundObjectFixture(t *testing.T) (*e2eCompoundObjectFixture, error) {
	t.Helper()

	filesystem := newE2EIRODSFilesystem(t, requireE2EEffectiveUser(t))

	cfg := requireE2EIRODSConfig(t)
	testUser := filesystem.GetAccount().ClientUser
	rootPath := fmt.Sprintf("/%s/home/%s/drs-compound-e2e-%d", cfg.IrodsZone, testUser, time.Now().UnixNano())
	compoundPath := path.Join(rootPath, "compound-root")
	subCollectionPath := path.Join(compoundPath, "series-a")
	ignoredCollectionPath := path.Join(compoundPath, "ignored")
	childObjectPath := path.Join(subCollectionPath, "child.txt")
	ignoredObjectPath := path.Join(ignoredCollectionPath, "ignored.txt")
	ignoreFilePath := path.Join(compoundPath, drs_support.DrsIgnoreFileName)
	childAVUName := "user:note"
	childAVUValue := "compound child metadata"
	childAVUUnit := "custom"

	if err := filesystem.MakeDir(subCollectionPath, true); err != nil {
		filesystem.Release()
		return nil, fmt.Errorf("create compound fixture collection %q: %w", subCollectionPath, err)
	}
	if err := filesystem.MakeDir(ignoredCollectionPath, true); err != nil {
		filesystem.Release()
		return nil, fmt.Errorf("create ignored fixture collection %q: %w", ignoredCollectionPath, err)
	}

	if _, err := filesystem.UploadFileFromBuffer(bytes.NewBuffer([]byte("compound e2e child object\n")), childObjectPath, "", false, true, nil); err != nil {
		filesystem.Release()
		return nil, fmt.Errorf("upload compound fixture object %q: %w", childObjectPath, err)
	}
	if err := filesystem.AddMetadata(childObjectPath, childAVUName, childAVUValue, childAVUUnit); err != nil {
		filesystem.Release()
		return nil, fmt.Errorf("add custom metadata to child object %q: %w", childObjectPath, err)
	}
	if _, err := filesystem.UploadFileFromBuffer(bytes.NewBuffer([]byte("compound e2e ignored object\n")), ignoredObjectPath, "", false, true, nil); err != nil {
		filesystem.Release()
		return nil, fmt.Errorf("upload ignored fixture object %q: %w", ignoredObjectPath, err)
	}
	ignoreFileContents := "# exclude ignored subtree from compound DRS\nignored/\n"
	if _, err := filesystem.UploadFileFromBuffer(bytes.NewBuffer([]byte(ignoreFileContents)), ignoreFilePath, "", false, true, nil); err != nil {
		filesystem.Release()
		return nil, fmt.Errorf("upload ignore file %q: %w", ignoreFilePath, err)
	}

	createResult, err := drs_support.CreateCompoundDrsObjectFromCollection(filesystem, compoundPath)
	if err != nil {
		filesystem.Release()
		return nil, fmt.Errorf("create compound DRS object at %q: %w", compoundPath, err)
	}
	if createResult == nil {
		filesystem.Release()
		return nil, fmt.Errorf("create compound DRS object at %q returned nil result", compoundPath)
	}
	if len(createResult.NodeErrors) > 0 {
		filesystem.Release()
		return nil, fmt.Errorf("create compound DRS object at %q reported node errors: %+v", compoundPath, createResult.NodeErrors)
	}

	childObject, err := drs_support.GetDrsObjectByIRODSPath(filesystem, childObjectPath)
	if err != nil {
		filesystem.Release()
		return nil, fmt.Errorf("resolve child DRS object at %q: %w", childObjectPath, err)
	}

	t.Cleanup(func() {
		defer filesystem.Release()
		if err := filesystem.RemoveDir(rootPath, true, true); err != nil && filesystem.Exists(rootPath) {
			t.Errorf("cleanup compound e2e fixture root %q: %v", rootPath, err)
		}
	})

	return &e2eCompoundObjectFixture{
		rootPath:        rootPath,
		compoundPath:    compoundPath,
		compoundID:      strings.TrimSpace(createResult.DrsID),
		ignoreFilePath:  ignoreFilePath,
		childObjectPath: childObjectPath,
		childObjectID:   strings.TrimSpace(childObject.Id),
		childAVUName:    childAVUName,
		childAVUValue:   childAVUValue,
		childAVUUnit:    childAVUUnit,
		ignoredPath:     ignoredObjectPath,
		expectedUser:    testUser,
	}, nil
}
