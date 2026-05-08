package drs_support

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	irodsutil "github.com/cyverse/go-irodsclient/irods/util"
	ignoreext "github.com/michael-conway/go-irodsclient-extensions/ignore"
)

const DrsIgnoreFileName = ".drsignore"

type IRODSReadWriteCloser interface {
	ReadAt(buffer []byte, offset int64) (int, error)
	Write(data []byte) (int, error)
	Close() error
}

type ignoreFilesystemAdapter struct {
	filesystem IRODSFilesystem
}

func (adapter *ignoreFilesystemAdapter) Stat(irodsPath string) (*irodsfs.Entry, error) {
	return compoundStat(adapter.filesystem, irodsPath)
}

func (adapter *ignoreFilesystemAdapter) OpenFile(irodsPath string, resource string, mode string) (ignoreext.IRODSFileHandleReader, error) {
	return compoundOpenFile(adapter.filesystem, irodsPath, resource, mode)
}

type CompoundCreateNodeError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type CompoundCreateResult struct {
	DrsID      string                    `json:"drsId"`
	RootPath   string                    `json:"rootPath"`
	NodeErrors []CompoundCreateNodeError `json:"nodeErrors,omitempty"`
}

type CompoundManifestAVU struct {
	Attribute string `json:"attribute"`
	Value     string `json:"value"`
	Unit      string `json:"unit,omitempty"`
}

type CompoundManifestNode struct {
	Path            string                 `json:"path"`
	RelativePath    string                 `json:"relativePath"`
	NodeType        string                 `json:"nodeType"`
	DrsID           string                 `json:"drsId"`
	Alias           string                 `json:"alias,omitempty"`
	Description     string                 `json:"description,omitempty"`
	WillAssignDrsID bool                   `json:"willAssignDrsId,omitempty"`
	Metadata        []CompoundManifestAVU  `json:"metadata,omitempty"`
	Children        []CompoundManifestNode `json:"children,omitempty"`
}

type CompoundManifestPreflight struct {
	Host          string                `json:"host,omitempty"`
	Port          int                   `json:"port,omitempty"`
	Zone          string                `json:"zone,omitempty"`
	RootPath      string                `json:"rootPath"`
	HasIgnoreFile bool                  `json:"hasIgnoreFile"`
	IgnoreFile    string                `json:"ignoreFile,omitempty"`
	RootDrsID     string                `json:"rootDrsId"`
	Warnings      []string              `json:"warnings,omitempty"`
	ExcludedPaths []string              `json:"excludedPaths,omitempty"`
	Manifest      *CompoundManifestNode `json:"manifest,omitempty"`
}

type compoundTreeNode struct {
	Entry    *irodsfs.Entry
	Metadata []*irodstypes.IRODSMeta
	Children []*compoundTreeNode
}

// HasCompoundIgnoreFile reports whether a .drsignore exists under the target collection path.
func HasCompoundIgnoreFile(filesystem IRODSFilesystem, collectionPath string) (bool, string, error) {
	if filesystem == nil {
		return false, "", fmt.Errorf("no iRODS filesystem provided")
	}

	rootPath := irodsutil.GetCorrectIRODSPath(strings.TrimSpace(collectionPath))
	if rootPath == "" || rootPath == "/" {
		return false, "", fmt.Errorf("a collection path is required")
	}

	ignorePath := path.Clean(path.Join(rootPath, DrsIgnoreFileName))
	ignoreEntry, err := compoundStat(filesystem, ignorePath)
	if err != nil {
		if isFileNotFoundError(err) {
			return false, ignorePath, nil
		}
		return false, "", fmt.Errorf("stat %q: %w", ignorePath, err)
	}

	if ignoreEntry == nil {
		return false, ignorePath, nil
	}

	if ignoreEntry.IsDir() {
		return false, "", fmt.Errorf("%q exists but is a collection, expected data object", ignorePath)
	}

	return true, ignorePath, nil
}

// AddDRSIgnoreTemplate creates a sample .drsignore file in the target collection.
func AddDRSIgnoreTemplate(filesystem IRODSFilesystem, collectionPath string) (string, error) {
	if filesystem == nil {
		return "", fmt.Errorf("no iRODS filesystem provided")
	}

	rootPath := irodsutil.GetCorrectIRODSPath(strings.TrimSpace(collectionPath))
	if rootPath == "" || rootPath == "/" {
		return "", fmt.Errorf("a collection path is required")
	}

	rootEntry, err := compoundStat(filesystem, rootPath)
	if err != nil {
		return "", fmt.Errorf("stat collection %q: %w", rootPath, err)
	}
	if rootEntry == nil || !rootEntry.IsDir() {
		return "", fmt.Errorf("path %q is not a collection", rootPath)
	}

	ignorePath := path.Clean(path.Join(rootPath, DrsIgnoreFileName))
	if _, err := compoundStat(filesystem, ignorePath); err == nil {
		return "", fmt.Errorf("%s already exists at %q", DrsIgnoreFileName, ignorePath)
	} else if !isFileNotFoundError(err) {
		return "", fmt.Errorf("stat %q: %w", ignorePath, err)
	}

	handle, err := compoundCreateFile(filesystem, ignorePath, "", "w")
	if err != nil {
		return "", fmt.Errorf("create %q: %w", ignorePath, err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = handle.Close()
		}
	}()

	sample := SampleDRSIgnore()
	if _, err := handle.Write([]byte(sample)); err != nil {
		return "", fmt.Errorf("write %q: %w", ignorePath, err)
	}

	if err := handle.Close(); err != nil {
		return "", fmt.Errorf("close %q: %w", ignorePath, err)
	}
	closed = true

	return ignorePath, nil
}

// BuildCompoundManifestPreflight creates a no-write compound manifest preview from iRODS state.
func BuildCompoundManifestPreflight(filesystem IRODSFilesystem, collectionPath string) (*CompoundManifestPreflight, error) {
	if filesystem == nil {
		return nil, fmt.Errorf("no iRODS filesystem provided")
	}

	rootPath := irodsutil.GetCorrectIRODSPath(strings.TrimSpace(collectionPath))
	if rootPath == "" || rootPath == "/" {
		return nil, fmt.Errorf("a collection absolute path is required")
	}

	rootEntry, err := compoundStat(filesystem, rootPath)
	if err != nil {
		return nil, fmt.Errorf("stat collection %q: %w", rootPath, err)
	}
	if rootEntry == nil || !rootEntry.IsDir() {
		return nil, fmt.Errorf("path %q is not a collection", rootPath)
	}

	root, err := buildCompoundTree(filesystem, rootPath)
	if err != nil {
		return nil, err
	}

	if err := failIfDescendantCompound(root); err != nil {
		return nil, err
	}

	hasIgnoreFile, ignorePath, err := HasCompoundIgnoreFile(filesystem, rootPath)
	if err != nil {
		return nil, err
	}

	ignores, err := readCompoundIgnores(filesystem, rootPath)
	if err != nil {
		return nil, err
	}

	excludedPaths := []string{}
	if ignores != nil {
		collectExcludedPaths(root, rootPath, ignores, &excludedPaths)
		sort.Strings(excludedPaths)
	}

	warnings := []string{}
	rootDrsID := drsIDFromMetadata(root.Metadata)
	if rootDrsID == "" {
		warnings = append(warnings, "root collection has no DRS id yet; one would be assigned during creation")
	}

	manifestNode, included := buildPreflightNode(root, rootPath, ignores)
	if !included || manifestNode == nil {
		return nil, fmt.Errorf("root collection %q was excluded from manifest preflight", rootPath)
	}

	result := &CompoundManifestPreflight{
		RootPath:      rootPath,
		HasIgnoreFile: hasIgnoreFile,
		IgnoreFile:    ignorePath,
		RootDrsID:     rootDrsID,
		Warnings:      warnings,
		ExcludedPaths: excludedPaths,
		Manifest:      manifestNode,
	}

	if account := filesystem.GetAccount(); account != nil {
		result.Host = strings.TrimSpace(account.Host)
		result.Port = account.Port
		result.Zone = strings.TrimSpace(account.ClientZone)
	}

	return result, nil
}

// CreateCompoundDrsObjectFromCollection bootstraps a collection hierarchy as a compound DRS object.
func CreateCompoundDrsObjectFromCollection(filesystem IRODSFilesystem, collectionPath string) (*CompoundCreateResult, error) {
	if filesystem == nil {
		return nil, fmt.Errorf("no iRODS filesystem provided")
	}

	rootPath := irodsutil.GetCorrectIRODSPath(strings.TrimSpace(collectionPath))
	if rootPath == "" || rootPath == "/" {
		return nil, fmt.Errorf("a collection absolute path is required")
	}

	rootEntry, err := compoundStat(filesystem, rootPath)
	if err != nil {
		return nil, fmt.Errorf("stat collection %q: %w", rootPath, err)
	}
	if rootEntry == nil || !rootEntry.IsDir() {
		return nil, fmt.Errorf("path %q is not a collection", rootPath)
	}

	root, err := buildCompoundTree(filesystem, rootPath)
	if err != nil {
		return nil, err
	}

	if collectionHasCompoundMarker(root.Metadata) {
		return nil, fmt.Errorf("collection %q is already a compound DRS object", rootPath)
	}

	if err := failIfDescendantCompound(root); err != nil {
		return nil, err
	}

	drsID := drsIDFromMetadata(root.Metadata)
	if drsID == "" {
		drsID, err = newGUID()
		if err != nil {
			return nil, fmt.Errorf("generate root DRS id: %w", err)
		}
	}

	ignores, err := readCompoundIgnores(filesystem, rootPath)
	if err != nil {
		return nil, err
	}

	nodes := collectIncludedNodes(root, rootPath, ignores)
	sort.Slice(nodes, func(i int, j int) bool {
		return nodes[i].Entry.Path < nodes[j].Entry.Path
	})

	nodeErrors := make([]CompoundCreateNodeError, 0)
	for _, node := range nodes {
		if node == nil || node.Entry == nil {
			continue
		}

		nodePath := node.Entry.Path
		relativePath := relativePathFromRoot(rootPath, nodePath)
		aliasValue := relativePath
		if aliasValue == "" {
			aliasValue = "."
		}

		if node.Entry.IsDir() {
			if err := upsertDrsMetadata(filesystem, nodePath, DrsAvuAliasAttrib, aliasValue); err != nil {
				nodeErrors = append(nodeErrors, CompoundCreateNodeError{Path: nodePath, Message: err.Error()})
			}
			if err := upsertDrsMetadata(filesystem, nodePath, DrsAvuDescriptionAttrib, aliasValue); err != nil {
				nodeErrors = append(nodeErrors, CompoundCreateNodeError{Path: nodePath, Message: err.Error()})
			}
			continue
		}

		if drsIDFromMetadata(node.Metadata) == "" {
			if _, err := CreateDrsObjectFromDataObject(filesystem, nodePath, "", aliasValue, []string{aliasValue}); err != nil {
				nodeErrors = append(nodeErrors, CompoundCreateNodeError{Path: nodePath, Message: fmt.Sprintf("create data object DRS metadata: %v", err)})
				continue
			}
		}

		if err := ensureDataObjectDrsCompleteness(filesystem, nodePath, aliasValue, aliasValue); err != nil {
			nodeErrors = append(nodeErrors, CompoundCreateNodeError{Path: nodePath, Message: err.Error()})
		}
	}

	if err := upsertDrsMetadata(filesystem, rootPath, DrsIdAvuAttrib, drsID); err != nil {
		nodeErrors = append(nodeErrors, CompoundCreateNodeError{Path: rootPath, Message: err.Error()})
	}
	if err := upsertDrsMetadata(filesystem, rootPath, DrsAvuCompoundManifestAttrib, "true"); err != nil {
		nodeErrors = append(nodeErrors, CompoundCreateNodeError{Path: rootPath, Message: err.Error()})
	}
	if err := upsertDrsMetadata(filesystem, rootPath, DrsAvuAliasAttrib, "."); err != nil {
		nodeErrors = append(nodeErrors, CompoundCreateNodeError{Path: rootPath, Message: err.Error()})
	}
	if err := upsertDrsMetadata(filesystem, rootPath, DrsAvuDescriptionAttrib, "."); err != nil {
		nodeErrors = append(nodeErrors, CompoundCreateNodeError{Path: rootPath, Message: err.Error()})
	}

	return &CompoundCreateResult{
		DrsID:      drsID,
		RootPath:   rootPath,
		NodeErrors: nodeErrors,
	}, nil
}

func buildCompoundTree(filesystem IRODSFilesystem, rootPath string) (*compoundTreeNode, error) {
	rootEntry, err := statEntry(filesystem, rootPath)
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", rootPath, err)
	}

	metadata, err := filesystem.ListMetadata(rootPath)
	if err != nil && !isFileNotFoundError(err) {
		return nil, fmt.Errorf("list metadata for %q: %w", rootPath, err)
	}

	node := &compoundTreeNode{
		Entry:    rootEntry,
		Metadata: metadata,
		Children: []*compoundTreeNode{},
	}

	if !rootEntry.IsDir() {
		return node, nil
	}

	children, err := filesystem.List(rootPath)
	if err != nil {
		return nil, fmt.Errorf("list collection %q: %w", rootPath, err)
	}

	for _, child := range children {
		if child == nil {
			continue
		}
		childNode, childErr := buildCompoundTree(filesystem, child.Path)
		if childErr != nil {
			return nil, childErr
		}
		node.Children = append(node.Children, childNode)
	}

	return node, nil
}

func statEntry(filesystem IRODSFilesystem, irodsPath string) (*irodsfs.Entry, error) {
	return compoundStat(filesystem, irodsPath)
}

func failIfDescendantCompound(root *compoundTreeNode) error {
	if root == nil {
		return nil
	}

	for _, child := range root.Children {
		if child == nil || child.Entry == nil || !child.Entry.IsDir() {
			continue
		}
		if collectionHasCompoundMarker(child.Metadata) {
			return fmt.Errorf("descendant collection %q is already a compound DRS object", child.Entry.Path)
		}
		if err := failIfDescendantCompound(child); err != nil {
			return err
		}
	}
	return nil
}

func readCompoundIgnores(filesystem IRODSFilesystem, rootPath string) (*ignoreext.Ignores, error) {
	ignorePath := path.Clean(path.Join(rootPath, DrsIgnoreFileName))
	ignores, err := ignoreext.ReadIgnoreFileFromIRODS(&ignoreFilesystemAdapter{filesystem: filesystem}, ignorePath, rootPath)
	if err == nil {
		return ignores, nil
	}
	if isFileNotFoundError(err) {
		return ignoreext.NewIgnores(rootPath, []string{})
	}
	return nil, fmt.Errorf("read %s at %q: %w", DrsIgnoreFileName, ignorePath, err)
}

func collectIncludedNodes(root *compoundTreeNode, rootPath string, ignores *ignoreext.Ignores) []*compoundTreeNode {
	if root == nil {
		return []*compoundTreeNode{}
	}

	included := []*compoundTreeNode{}
	stack := []*compoundTreeNode{root}
	for len(stack) > 0 {
		node := stack[0]
		stack = stack[1:]
		if node == nil || node.Entry == nil {
			continue
		}
		if isDefaultExcludedCompoundEntry(node.Entry) {
			continue
		}

		isRoot := node.Entry.Path == rootPath
		ignored := false
		if !isRoot && ignores != nil {
			ignored = ignores.IsIgnored(node.Entry.Path, node.Entry.IsDir())
		}
		if !ignored {
			included = append(included, node)
		}

		for _, child := range node.Children {
			stack = append(stack, child)
		}
	}

	return included
}

func collectionHasCompoundMarker(metas []*irodstypes.IRODSMeta) bool {
	for _, meta := range metas {
		if meta == nil {
			continue
		}
		if meta.Name != DrsAvuCompoundManifestAttrib {
			continue
		}
		if meta.Units != "" && !strings.EqualFold(meta.Units, DrsAvuUnit) {
			continue
		}
		return true
	}
	return false
}

func drsIDFromMetadata(metas []*irodstypes.IRODSMeta) string {
	for _, meta := range metas {
		if meta == nil {
			continue
		}
		if meta.Name != DrsIdAvuAttrib {
			continue
		}
		if meta.Units != "" && !strings.EqualFold(meta.Units, DrsAvuUnit) {
			continue
		}
		return strings.TrimSpace(meta.Value)
	}
	return ""
}

func upsertDrsMetadata(filesystem IRODSFilesystem, irodsPath string, attributeName string, attributeValue string) error {
	metas, err := filesystem.ListMetadata(irodsPath)
	if err != nil && !isFileNotFoundError(err) {
		return fmt.Errorf("list metadata for %q: %w", irodsPath, err)
	}

	for _, meta := range metas {
		if meta == nil {
			continue
		}
		if meta.Name != attributeName {
			continue
		}
		if meta.Units != "" && !strings.EqualFold(meta.Units, DrsAvuUnit) {
			continue
		}
		if err := filesystem.DeleteMetadataByAVU(irodsPath, meta.Name, meta.Value, meta.Units); err != nil {
			return fmt.Errorf("remove metadata %q from %q: %w", attributeName, irodsPath, err)
		}
	}

	if strings.TrimSpace(attributeValue) == "" {
		return nil
	}
	if err := filesystem.AddMetadata(irodsPath, attributeName, strings.TrimSpace(attributeValue), DrsAvuUnit); err != nil {
		return fmt.Errorf("set metadata %q on %q: %w", attributeName, irodsPath, err)
	}
	return nil
}

func relativePathFromRoot(rootPath string, targetPath string) string {
	rootPath = path.Clean(rootPath)
	targetPath = path.Clean(targetPath)
	if rootPath == targetPath {
		return ""
	}
	trimmed := strings.TrimPrefix(targetPath, rootPath)
	trimmed = strings.TrimPrefix(trimmed, "/")
	return trimmed
}

func isFileNotFoundError(err error) bool {
	return err != nil &&
		(errors.Is(err, os.ErrNotExist) ||
			irodstypes.IsFileNotFoundError(err))
}

func buildPreflightNode(node *compoundTreeNode, rootPath string, ignores *ignoreext.Ignores) (*CompoundManifestNode, bool) {
	if node == nil || node.Entry == nil {
		return nil, false
	}
	if isDefaultExcludedCompoundEntry(node.Entry) {
		return nil, false
	}

	if node.Entry.Path != rootPath && ignores != nil && ignores.IsIgnored(node.Entry.Path, node.Entry.IsDir()) {
		return nil, false
	}

	relativePath := relativePathFromRoot(rootPath, node.Entry.Path)
	alias := aliasFromMetadata(node.Metadata)
	description := descriptionFromMetadata(node.Metadata)

	if node.Entry.IsDir() {
		if strings.TrimSpace(alias) == "" {
			if relativePath == "" {
				alias = "."
			} else {
				alias = relativePath
			}
		}
		if strings.TrimSpace(description) == "" {
			if relativePath == "" {
				description = "."
			} else {
				description = relativePath
			}
		}
	}

	preflightNode := &CompoundManifestNode{
		Path:         node.Entry.Path,
		RelativePath: relativePath,
		NodeType:     compoundNodeType(node.Entry),
		DrsID:        drsIDFromMetadata(node.Metadata),
		Alias:        strings.TrimSpace(alias),
		Description:  strings.TrimSpace(description),
		Metadata:     nonDrsMetadataFromMetas(node.Metadata),
		Children:     []CompoundManifestNode{},
	}

	if !node.Entry.IsDir() && preflightNode.DrsID == "" {
		preflightNode.WillAssignDrsID = true
	}

	for _, child := range node.Children {
		childNode, included := buildPreflightNode(child, rootPath, ignores)
		if !included || childNode == nil {
			continue
		}
		preflightNode.Children = append(preflightNode.Children, *childNode)
	}

	return preflightNode, true
}

func collectExcludedPaths(node *compoundTreeNode, rootPath string, ignores *ignoreext.Ignores, excludedPaths *[]string) {
	if node == nil || node.Entry == nil || excludedPaths == nil {
		return
	}
	if isDefaultExcludedCompoundEntry(node.Entry) {
		return
	}

	if node.Entry.Path != rootPath && ignores != nil && ignores.IsIgnored(node.Entry.Path, node.Entry.IsDir()) {
		*excludedPaths = append(*excludedPaths, node.Entry.Path)
	}

	for _, child := range node.Children {
		collectExcludedPaths(child, rootPath, ignores, excludedPaths)
	}
}

func aliasFromMetadata(metas []*irodstypes.IRODSMeta) string {
	for _, meta := range metas {
		if meta == nil || meta.Name != DrsAvuAliasAttrib {
			continue
		}
		if meta.Units != "" && !strings.EqualFold(meta.Units, DrsAvuUnit) {
			continue
		}
		value := strings.TrimSpace(meta.Value)
		if value != "" {
			return value
		}
	}
	return ""
}

func descriptionFromMetadata(metas []*irodstypes.IRODSMeta) string {
	for _, meta := range metas {
		if meta == nil || meta.Name != DrsAvuDescriptionAttrib {
			continue
		}
		if meta.Units != "" && !strings.EqualFold(meta.Units, DrsAvuUnit) {
			continue
		}
		value := strings.TrimSpace(meta.Value)
		if value != "" {
			return value
		}
	}
	return ""
}

func nonDrsMetadataFromMetas(metas []*irodstypes.IRODSMeta) []CompoundManifestAVU {
	if len(metas) == 0 {
		return nil
	}

	manifestMetas := make([]CompoundManifestAVU, 0, len(metas))
	for _, meta := range metas {
		if meta == nil || isDrsMetadata(meta) {
			continue
		}
		manifestMetas = append(manifestMetas, CompoundManifestAVU{
			Attribute: strings.TrimSpace(meta.Name),
			Value:     strings.TrimSpace(meta.Value),
			Unit:      strings.TrimSpace(meta.Units),
		})
	}
	return manifestMetas
}

func compoundNodeType(entry *irodsfs.Entry) string {
	if entry == nil {
		return ""
	}
	if entry.IsDir() {
		return "collection"
	}
	return "data_object"
}

func ensureDataObjectDrsCompleteness(filesystem IRODSFilesystem, dataObjectPath string, fallbackAlias string, fallbackDescription string) error {
	metas, err := filesystem.ListMetadata(dataObjectPath)
	if err != nil && !isFileNotFoundError(err) {
		return fmt.Errorf("list metadata for %q: %w", dataObjectPath, err)
	}

	if !hasDrsMetadataWithValue(metas, DrsAvuMimeTypeAttrib) {
		mimeType := normalizedMimeType(dataObjectPath, "")
		if mimeType != "" {
			if err := filesystem.AddMetadata(dataObjectPath, DrsAvuMimeTypeAttrib, mimeType, DrsAvuUnit); err != nil {
				return fmt.Errorf("set mime type metadata on %q: %w", dataObjectPath, err)
			}
		}
	}

	if !hasDrsMetadataWithValue(metas, DrsAvuVersionAttrib) {
		entry, err := statEntry(filesystem, dataObjectPath)
		if err != nil {
			return fmt.Errorf("stat data object %q: %w", dataObjectPath, err)
		}
		entry, err = entryWithAllReplicas(filesystem, entry)
		if err != nil {
			return err
		}

		dataObject := entry.ToDataObject()
		checksum, err := ensureDataObjectChecksum(filesystem, dataObjectPath, dataObject.Replicas)
		if err != nil {
			return fmt.Errorf("ensure checksum for %q: %w", dataObjectPath, err)
		}
		if checksum != nil && strings.TrimSpace(checksum.Value) != "" {
			if err := filesystem.AddMetadata(dataObjectPath, DrsAvuVersionAttrib, checksum.Value, DrsAvuUnit); err != nil {
				return fmt.Errorf("set version metadata on %q: %w", dataObjectPath, err)
			}
		}
	}

	if err := ensureDrsMetadataValue(filesystem, dataObjectPath, DrsAvuAliasAttrib, fallbackAlias); err != nil {
		return err
	}
	if err := ensureDrsMetadataValue(filesystem, dataObjectPath, DrsAvuDescriptionAttrib, fallbackDescription); err != nil {
		return err
	}

	return nil
}

func hasDrsMetadataWithValue(metas []*irodstypes.IRODSMeta, attributeName string) bool {
	for _, meta := range metas {
		if meta == nil {
			continue
		}
		if meta.Name != attributeName {
			continue
		}
		if meta.Units != "" && !strings.EqualFold(meta.Units, DrsAvuUnit) {
			continue
		}
		if strings.TrimSpace(meta.Value) != "" {
			return true
		}
	}
	return false
}

func ensureDrsMetadataValue(filesystem IRODSFilesystem, irodsPath string, attributeName string, fallbackValue string) error {
	fallbackValue = strings.TrimSpace(fallbackValue)
	if fallbackValue == "" {
		return nil
	}

	metas, err := filesystem.ListMetadata(irodsPath)
	if err != nil && !isFileNotFoundError(err) {
		return fmt.Errorf("list metadata for %q: %w", irodsPath, err)
	}
	if hasDrsMetadataWithValue(metas, attributeName) {
		return nil
	}

	if err := filesystem.AddMetadata(irodsPath, attributeName, fallbackValue, DrsAvuUnit); err != nil {
		return fmt.Errorf("set metadata %q on %q: %w", attributeName, irodsPath, err)
	}
	return nil
}

func isDefaultExcludedCompoundEntry(entry *irodsfs.Entry) bool {
	if entry == nil {
		return false
	}
	// .drsignore is a creation/preflight control file and is never part of the
	// compound object bundle.
	return strings.EqualFold(strings.TrimSpace(entry.Name), DrsIgnoreFileName)
}

func compoundStat(filesystem any, irodsPath string) (*irodsfs.Entry, error) {
	switch fs := filesystem.(type) {
	case interface {
		Stat(irodsPath string) (*irodsfs.Entry, error)
	}:
		return fs.Stat(irodsPath)
	case interface {
		StatFile(irodsPath string) (*irodsfs.Entry, error)
	}:
		return fs.StatFile(irodsPath)
	default:
		return nil, fmt.Errorf("filesystem does not support stat")
	}
}

func compoundOpenFile(filesystem any, irodsPath string, resource string, mode string) (IRODSReadWriteCloser, error) {
	switch fs := filesystem.(type) {
	case interface {
		OpenFile(irodsPath string, resource string, mode string) (*irodsfs.FileHandle, error)
	}:
		return fs.OpenFile(irodsPath, resource, mode)
	case interface {
		OpenFile(irodsPath string, resource string, mode string) (IRODSReadWriteCloser, error)
	}:
		return fs.OpenFile(irodsPath, resource, mode)
	default:
		return nil, fmt.Errorf("filesystem does not support file open")
	}
}

func compoundCreateFile(filesystem any, irodsPath string, resource string, mode string) (IRODSReadWriteCloser, error) {
	switch fs := filesystem.(type) {
	case interface {
		CreateFile(irodsPath string, resource string, mode string) (*irodsfs.FileHandle, error)
	}:
		return fs.CreateFile(irodsPath, resource, mode)
	case interface {
		CreateFile(irodsPath string, resource string, mode string) (IRODSReadWriteCloser, error)
	}:
		return fs.CreateFile(irodsPath, resource, mode)
	default:
		return nil, fmt.Errorf("filesystem does not support file creation")
	}
}
