package drs_support

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
)

type compoundTestFilesystem struct {
	account        *irodstypes.IRODSAccount
	entriesByPath  map[string]*irodsfs.Entry
	metadataByPath map[string][]*irodstypes.IRODSMeta
	fileContents   map[string][]byte
}

func newCompoundTestFilesystem(root string) *compoundTestFilesystem {
	return &compoundTestFilesystem{
		account: &irodstypes.IRODSAccount{
			Host:       "irods.example.org",
			Port:       1247,
			ClientZone: "tempZone",
			ClientUser: "test1",
		},
		entriesByPath: map[string]*irodsfs.Entry{
			root: {
				Type: irodsfs.DirectoryEntry,
				Name: filepath.Base(root),
				Path: root,
			},
		},
		metadataByPath: map[string][]*irodstypes.IRODSMeta{
			root: {},
		},
		fileContents: map[string][]byte{},
	}
}

func (f *compoundTestFilesystem) StatFile(irodsPath string) (*irodsfs.Entry, error) {
	return f.Stat(irodsPath)
}

func (f *compoundTestFilesystem) Stat(irodsPath string) (*irodsfs.Entry, error) {
	entry, ok := f.entriesByPath[irodsPath]
	if !ok {
		return nil, os.ErrNotExist
	}
	return entry, nil
}

func (f *compoundTestFilesystem) List(irodsPath string) ([]*irodsfs.Entry, error) {
	irodsPath = strings.TrimSuffix(irodsPath, "/")
	if irodsPath == "" {
		irodsPath = "/"
	}

	children := []*irodsfs.Entry{}
	for candidatePath, entry := range f.entriesByPath {
		if candidatePath == irodsPath {
			continue
		}
		parent := filepath.Dir(candidatePath)
		if parent == "." {
			parent = "/"
		}
		parent = filepath.ToSlash(parent)
		if parent == irodsPath {
			children = append(children, entry)
		}
	}

	sort.Slice(children, func(i int, j int) bool {
		return children[i].Path < children[j].Path
	})
	return children, nil
}

func (f *compoundTestFilesystem) ListMetadata(irodsPath string) ([]*irodstypes.IRODSMeta, error) {
	metas, ok := f.metadataByPath[irodsPath]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]*irodstypes.IRODSMeta(nil), metas...), nil
}

func (f *compoundTestFilesystem) AddMetadata(irodsPath string, attName string, attValue string, attUnits string) error {
	f.metadataByPath[irodsPath] = append(f.metadataByPath[irodsPath], &irodstypes.IRODSMeta{
		Name:  attName,
		Value: attValue,
		Units: attUnits,
	})
	return nil
}

func (f *compoundTestFilesystem) DeleteMetadataByAVU(irodsPath string, attName string, attValue string, attUnits string) error {
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

func (f *compoundTestFilesystem) GetAccount() *irodstypes.IRODSAccount {
	return f.account
}

func (f *compoundTestFilesystem) EnsureDataObjectChecksum(irodsPath string) (*irodstypes.IRODSChecksum, error) {
	if _, ok := f.entriesByPath[irodsPath]; !ok {
		return nil, os.ErrNotExist
	}
	return irodstypes.CreateIRODSChecksum("d41d8cd98f00b204e9800998ecf8427e")
}

func (f *compoundTestFilesystem) OpenFile(irodsPath string, resource string, mode string) (IRODSReadWriteCloser, error) {
	_ = resource
	readOnly := strings.Contains(strings.ToLower(mode), "r") && !strings.Contains(strings.ToLower(mode), "w")
	if _, ok := f.entriesByPath[irodsPath]; !ok {
		return nil, os.ErrNotExist
	}
	return &compoundTestFileHandle{
		filesystem: f,
		path:       irodsPath,
		readOnly:   readOnly,
	}, nil
}

func (f *compoundTestFilesystem) CreateFile(irodsPath string, resource string, mode string) (IRODSReadWriteCloser, error) {
	_ = resource
	_ = mode
	if _, exists := f.entriesByPath[irodsPath]; exists {
		return nil, os.ErrExist
	}
	parent := filepath.Dir(irodsPath)
	if parent == "." {
		parent = "/"
	}
	parent = filepath.ToSlash(parent)
	parentEntry, ok := f.entriesByPath[parent]
	if !ok || !parentEntry.IsDir() {
		return nil, os.ErrNotExist
	}

	f.entriesByPath[irodsPath] = &irodsfs.Entry{
		Type: irodsfs.FileEntry,
		Name: filepath.Base(irodsPath),
		Path: irodsPath,
	}
	f.metadataByPath[irodsPath] = []*irodstypes.IRODSMeta{}
	f.fileContents[irodsPath] = []byte{}

	return &compoundTestFileHandle{
		filesystem: f,
		path:       irodsPath,
	}, nil
}

func (f *compoundTestFilesystem) addCollection(collectionPath string) {
	f.entriesByPath[collectionPath] = &irodsfs.Entry{
		Type: irodsfs.DirectoryEntry,
		Name: filepath.Base(collectionPath),
		Path: collectionPath,
	}
	f.metadataByPath[collectionPath] = []*irodstypes.IRODSMeta{}
}

func (f *compoundTestFilesystem) addDataObject(dataObjectPath string) {
	f.entriesByPath[dataObjectPath] = &irodsfs.Entry{
		Type: irodsfs.FileEntry,
		Name: filepath.Base(dataObjectPath),
		Path: dataObjectPath,
		Size: 12,
	}
	f.metadataByPath[dataObjectPath] = []*irodstypes.IRODSMeta{}
	f.fileContents[dataObjectPath] = []byte("hello world")
}

type compoundTestFileHandle struct {
	filesystem *compoundTestFilesystem
	path       string
	readOnly   bool
	closed     bool
}

func (h *compoundTestFileHandle) ReadAt(buffer []byte, offset int64) (int, error) {
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

func (h *compoundTestFileHandle) Write(data []byte) (int, error) {
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

func (h *compoundTestFileHandle) Close() error {
	h.closed = true
	return nil
}

func TestCreateCompoundDrsObjectChecksDescendantBeforeIgnore(t *testing.T) {
	rootPath := "/tempZone/home/test1/compound"
	filesystem := newCompoundTestFilesystem(rootPath)
	filesystem.addCollection(rootPath + "/child")
	filesystem.addDataObject(rootPath + "/" + DrsIgnoreFileName)
	filesystem.fileContents[rootPath+"/"+DrsIgnoreFileName] = []byte("child/**\n")

	if err := filesystem.AddMetadata(rootPath+"/child", DrsAvuCompoundManifestAttrib, "true", DrsAvuUnit); err != nil {
		t.Fatalf("seed child compound marker: %v", err)
	}

	_, err := CreateCompoundDrsObjectFromCollection(filesystem, rootPath)
	if err == nil {
		t.Fatal("expected create compound to fail when descendant is already compound")
	}

	if !strings.Contains(err.Error(), "descendant collection") {
		t.Fatalf("expected descendant compound error, got %v", err)
	}
}

func TestHasCompoundIgnoreFile(t *testing.T) {
	rootPath := "/tempZone/home/test1/compound"
	filesystem := newCompoundTestFilesystem(rootPath)

	hasIgnore, ignorePath, err := HasCompoundIgnoreFile(filesystem, rootPath)
	if err != nil {
		t.Fatalf("has compound ignore: %v", err)
	}
	if hasIgnore {
		t.Fatalf("expected no ignore file in fresh collection")
	}
	if ignorePath != rootPath+"/"+DrsIgnoreFileName {
		t.Fatalf("expected ignore path %q, got %q", rootPath+"/"+DrsIgnoreFileName, ignorePath)
	}

	filesystem.addDataObject(rootPath + "/" + DrsIgnoreFileName)
	hasIgnore, ignorePath, err = HasCompoundIgnoreFile(filesystem, rootPath)
	if err != nil {
		t.Fatalf("has compound ignore after create: %v", err)
	}
	if !hasIgnore || ignorePath != rootPath+"/"+DrsIgnoreFileName {
		t.Fatalf("expected ignore file presence, got has=%v path=%q", hasIgnore, ignorePath)
	}
}

func TestBuildCompoundManifestPreflightIsNoWrite(t *testing.T) {
	rootPath := "/tempZone/home/test1/compound"
	filesystem := newCompoundTestFilesystem(rootPath)
	filesystem.addCollection(rootPath + "/sub")
	filesystem.addDataObject(rootPath + "/sub/object.txt")
	filesystem.addDataObject(rootPath + "/" + DrsIgnoreFileName)
	filesystem.fileContents[rootPath+"/"+DrsIgnoreFileName] = []byte("*.tmp\n")

	result, err := BuildCompoundManifestPreflight(filesystem, rootPath)
	if err != nil {
		t.Fatalf("build preflight: %v", err)
	}

	if result == nil || result.Manifest == nil {
		t.Fatalf("expected non-empty preflight manifest")
	}
	if result.RootDrsID != "" {
		t.Fatalf("expected blank rootDrsID when not assigned, got %q", result.RootDrsID)
	}

	if result.Manifest.NodeType != "collection" || result.Manifest.Path != rootPath {
		t.Fatalf("unexpected preflight root manifest %+v", result.Manifest)
	}
	if result.Manifest.DrsID != "" {
		t.Fatalf("expected blank root manifest drsId when not assigned, got %q", result.Manifest.DrsID)
	}

	if len(result.Manifest.Children) == 0 {
		t.Fatalf("expected preflight manifest children, got %+v", result.Manifest)
	}
	if manifestContainsPath(result.Manifest, rootPath+"/"+DrsIgnoreFileName) {
		t.Fatalf("expected preflight manifest to exclude .drsignore")
	}
	subNode := manifestFindPath(result.Manifest, rootPath+"/sub")
	if subNode == nil {
		t.Fatalf("expected subcollection node in preflight manifest")
	}
	if subNode.DrsID != "" {
		t.Fatalf("expected blank drsId for intermediary subcollection, got %q", subNode.DrsID)
	}
	objectNode := manifestFindPath(result.Manifest, rootPath+"/sub/object.txt")
	if objectNode == nil {
		t.Fatalf("expected data object node in preflight manifest")
	}
	if objectNode.DrsID != "" {
		t.Fatalf("expected blank drsId for unassigned data object in preflight, got %q", objectNode.DrsID)
	}

	rootMetas := filesystem.metadataByPath[rootPath]
	for _, meta := range rootMetas {
		if meta != nil && meta.Name == DrsAvuCompoundManifestAttrib {
			t.Fatalf("expected preflight to avoid writes, got %+v", rootMetas)
		}
	}
}

func TestCreateCompoundDrsObjectAppliesIgnoreAndScaffolding(t *testing.T) {
	rootPath := "/tempZone/home/test1/compound"
	filesystem := newCompoundTestFilesystem(rootPath)
	filesystem.addCollection(rootPath + "/a")
	filesystem.addCollection(rootPath + "/skip")
	filesystem.addDataObject(rootPath + "/a/new.txt")
	filesystem.addDataObject(rootPath + "/a/existing.txt")
	filesystem.addDataObject(rootPath + "/skip/ignored.txt")
	filesystem.addDataObject(rootPath + "/" + DrsIgnoreFileName)
	filesystem.fileContents[rootPath+"/"+DrsIgnoreFileName] = []byte("skip/**\n")

	const existingDrsID = "11111111-2222-3333-4444-555555555555"
	if err := filesystem.AddMetadata(rootPath+"/a/existing.txt", DrsIdAvuAttrib, existingDrsID, DrsAvuUnit); err != nil {
		t.Fatalf("seed existing data object drs id: %v", err)
	}

	result, err := CreateCompoundDrsObjectFromCollection(filesystem, rootPath)
	if err != nil {
		t.Fatalf("create compound object: %v", err)
	}

	if result == nil || strings.TrimSpace(result.DrsID) == "" {
		t.Fatal("expected non-empty root DRS id")
	}

	if len(result.NodeErrors) != 0 {
		t.Fatalf("expected no node errors, got %+v", result.NodeErrors)
	}

	rootMetas, err := filesystem.ListMetadata(rootPath)
	if err != nil {
		t.Fatalf("list root metadata: %v", err)
	}
	if !hasMetadata(rootMetas, DrsAvuCompoundManifestAttrib, "true", DrsAvuUnit) {
		t.Fatalf("expected root compound marker metadata")
	}
	if !hasMetadata(rootMetas, DrsAvuAliasAttrib, ".", DrsAvuUnit) {
		t.Fatalf("expected root alias metadata")
	}
	if !hasMetadata(rootMetas, DrsAvuDescriptionAttrib, ".", DrsAvuUnit) {
		t.Fatalf("expected root description metadata")
	}
	if !hasMetadata(rootMetas, DrsIdAvuAttrib, result.DrsID, DrsAvuUnit) {
		t.Fatalf("expected root DRS id metadata")
	}

	subcollectionMetas, err := filesystem.ListMetadata(rootPath + "/a")
	if err != nil {
		t.Fatalf("list subcollection metadata: %v", err)
	}
	if !hasMetadata(subcollectionMetas, DrsAvuAliasAttrib, "a", DrsAvuUnit) {
		t.Fatalf("expected subcollection alias metadata")
	}
	if !hasMetadata(subcollectionMetas, DrsAvuDescriptionAttrib, "a", DrsAvuUnit) {
		t.Fatalf("expected subcollection description metadata")
	}

	newObjectMetas, err := filesystem.ListMetadata(rootPath + "/a/new.txt")
	if err != nil {
		t.Fatalf("list new object metadata: %v", err)
	}
	if drsIDFromMetadata(newObjectMetas) == "" {
		t.Fatalf("expected generated DRS id for non-ignored data object")
	}
	if !hasMetadataNameWithValue(newObjectMetas, DrsAvuAliasAttrib) {
		t.Fatalf("expected alias metadata for new data object, got %+v", newObjectMetas)
	}
	if !hasMetadataNameWithValue(newObjectMetas, DrsAvuDescriptionAttrib) {
		t.Fatalf("expected description metadata for new data object, got %+v", newObjectMetas)
	}
	if !hasMetadataNameWithValue(newObjectMetas, DrsAvuMimeTypeAttrib) {
		t.Fatalf("expected mime type metadata for new data object, got %+v", newObjectMetas)
	}
	if !hasMetadataNameWithValue(newObjectMetas, DrsAvuVersionAttrib) {
		t.Fatalf("expected version metadata for new data object, got %+v", newObjectMetas)
	}

	existingObjectMetas, err := filesystem.ListMetadata(rootPath + "/a/existing.txt")
	if err != nil {
		t.Fatalf("list existing object metadata: %v", err)
	}
	if drsIDFromMetadata(existingObjectMetas) != existingDrsID {
		t.Fatalf("expected existing DRS id to be preserved, got %q", drsIDFromMetadata(existingObjectMetas))
	}

	ignoredObjectMetas, err := filesystem.ListMetadata(rootPath + "/skip/ignored.txt")
	if err != nil {
		t.Fatalf("list ignored object metadata: %v", err)
	}
	if drsIDFromMetadata(ignoredObjectMetas) != "" {
		t.Fatalf("expected ignored data object to remain without DRS id")
	}

	ignoreObjectMetas, err := filesystem.ListMetadata(rootPath + "/" + DrsIgnoreFileName)
	if err != nil {
		t.Fatalf("list .drsignore metadata: %v", err)
	}
	if drsIDFromMetadata(ignoreObjectMetas) != "" {
		t.Fatalf("expected .drsignore to remain outside compound bundle and without DRS id")
	}
}

func hasMetadata(metas []*irodstypes.IRODSMeta, name string, value string, units string) bool {
	for _, meta := range metas {
		if meta == nil {
			continue
		}
		if meta.Name == name && meta.Value == value && meta.Units == units {
			return true
		}
	}
	return false
}

func hasMetadataNameWithValue(metas []*irodstypes.IRODSMeta, name string) bool {
	for _, meta := range metas {
		if meta == nil {
			continue
		}
		if meta.Name == name && strings.TrimSpace(meta.Value) != "" && (meta.Units == "" || meta.Units == DrsAvuUnit) {
			return true
		}
	}
	return false
}

func manifestContainsPath(node *CompoundManifestNode, targetPath string) bool {
	if node == nil {
		return false
	}
	if node.Path == targetPath {
		return true
	}
	for _, child := range node.Children {
		childNode := child
		if manifestContainsPath(&childNode, targetPath) {
			return true
		}
	}
	return false
}

func manifestFindPath(node *CompoundManifestNode, targetPath string) *CompoundManifestNode {
	if node == nil {
		return nil
	}
	if node.Path == targetPath {
		return node
	}
	for _, child := range node.Children {
		childNode := child
		found := manifestFindPath(&childNode, targetPath)
		if found != nil {
			return found
		}
	}
	return nil
}
