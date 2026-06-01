package drs_support

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodslowfs "github.com/cyverse/go-irodsclient/irods/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	irodsutil "github.com/cyverse/go-irodsclient/irods/util"
	extmetadata "github.com/michael-conway/go-irodsclient-extensions/metadata"
)

/*
Support for mapping DRS to iRODS.

- Surfacing iRODS Objects as DRS objects
- Query and listing by DRS metadata
*/

const DrsAvuUnit = "iRODS:DRS"
const DrsIdAvuAttrib = "iRODS:DRS:ID"
const DrsAvuVersionAttrib = "iRODS:DRS:VERSION"
const DrsAvuMimeTypeAttrib = "iRODS:DRS:MIME_TYPE"
const DrsAvuCompoundManifestAttrib = "iRODS:DRS:COMPOUND_MANIFEST"
const DrsAvuAliasAttrib = "iRODS:DRS:ALIAS"
const DrsAvuDescriptionAttrib = "iRODS:DRS:DESCRIPTION"

type InternalChecksum struct {
	// md5 | sha256
	Type  string
	Value string
}

type InternalReplica struct {
	ResourceName      string
	ResourceHierarchy string
	Path              string
	Status            string
}

type InternalDrsObject struct {
	// An identifier unique to this DrsObject.
	Id string
	// iRODS logical path to the data object.
	AbsolutePath string
	// Zone that contains object.
	IrodsZone string
	// Resource that currently serves the selected replica for this object.
	ResourceName string
	// All discovered replicas for the data object.
	Replicas []InternalReplica
	// Object size in bytes.
	Size int64
	// Timestamp of content create in RFC3339.
	CreatedTime time.Time
	// Timestamp of content update in RFC3339.
	UpdatedTime time.Time
	// A string representing a version.
	Version string
	// A string providing the mime-type of the DrsObject.
	MimeType string
	// Indicates if this DRS id resolves to a manifest file stored as an iRODS data object.
	IsManifest bool
	// The checksum of the DrsObject.
	Checksum *InternalChecksum
	// Parsed manifest contents when available.
	Contents []DrsManifestEntry
	// A human readable description of the DrsObject.
	Description string
	// A list of alternate identifiers.
	Aliases []string
}

// DrsObjectPage represents one page of DRS object results from a listing operation.
// Offset and Limit reflect the requested page window. Total is populated only when
// TotalKnown is true. HasMore reports whether another page exists after the returned slice.
type DrsObjectPage struct {
	Objects    []*InternalDrsObject
	Offset     int
	Limit      int
	Total      int
	TotalKnown bool
	HasMore    bool
}

type DrsMetadataField string

const (
	DrsMetadataFieldMimeType    DrsMetadataField = "mimeType"
	DrsMetadataFieldVersion     DrsMetadataField = "version"
	DrsMetadataFieldDescription DrsMetadataField = "description"
	DrsMetadataFieldAlias       DrsMetadataField = "alias"
)

type DrsListingScope string

const (
	DrsListingScopeAll      DrsListingScope = "all"
	DrsListingScopeObjects  DrsListingScope = "objects"
	DrsListingScopeCompound DrsListingScope = "compound"
)

const (
	drsListingEntryQueryLimit     = 500
	drsListingAVUAttributePattern = "iRODS:DRS:%"
)

type drsListingEntry struct {
	Entry       *irodsfs.Entry
	MatchedAVUs []extmetadata.AVUStat
}

type IRODSFilesystem interface {
	StatFile(irodsPath string) (*irodsfs.Entry, error)
	List(irodsPath string) ([]*irodsfs.Entry, error)
	ListMetadata(irodsPath string) ([]*irodstypes.IRODSMeta, error)
	AddMetadata(irodsPath string, attName string, attValue string, attUnits string) error
	DeleteMetadataByAVU(irodsPath string, attName string, attValue string, attUnits string) error
	GetAccount() *irodstypes.IRODSAccount
}

type EntryMetadataQuerier interface {
	QueryMetadataEntries(query extmetadata.EntryQuery) (extmetadata.EntryQueryResult, error)
}

type checksumEnsuringFilesystem interface {
	EnsureDataObjectChecksum(irodsPath string) (*irodstypes.IRODSChecksum, error)
}

type allReplicaStatFilesystem interface {
	StatFileWithAllReplicas(irodsPath string) (*irodsfs.Entry, error)
}

func NormalizeDrsListingScope(scope DrsListingScope) (DrsListingScope, error) {
	switch scope {
	case "", DrsListingScopeAll:
		return DrsListingScopeAll, nil
	case DrsListingScopeObjects, DrsListingScopeCompound:
		return scope, nil
	default:
		return "", fmt.Errorf("unsupported DRS listing scope %q", scope)
	}
}

// ApplyDrsMetadata maps DRS AVUs from iRODS metadata onto an InternalDrsObject.
func ApplyDrsMetadata(object *InternalDrsObject, metas []*irodstypes.IRODSMeta) error {
	if object == nil {
		return fmt.Errorf("no internal DRS object provided")
	}

	for _, meta := range metas {
		if meta == nil {
			continue
		}

		if meta.Units != "" && !strings.EqualFold(meta.Units, DrsAvuUnit) {
			continue
		}

		switch meta.Name {
		case DrsIdAvuAttrib:
			object.Id = strings.TrimSpace(meta.Value)
		case DrsAvuVersionAttrib:
			object.Version = strings.TrimSpace(meta.Value)
		case DrsAvuMimeTypeAttrib:
			object.MimeType = strings.TrimSpace(meta.Value)
		case DrsAvuDescriptionAttrib:
			object.Description = strings.TrimSpace(meta.Value)
		case DrsAvuAliasAttrib:
			alias := strings.TrimSpace(meta.Value)
			if alias != "" {
				object.Aliases = append(object.Aliases, alias)
			}
		case DrsAvuCompoundManifestAttrib:
			object.IsManifest = true
		}
	}

	return nil
}

// CreateDrsObjectFromDataObject decorates a single existing iRODS data object as a DRS object.
// The caller must provide a filesystem that is already connected to iRODS, the absolute path to
// the data object, a mime type, a human-readable description, and any alternate identifiers to
// store as DRS alias AVUs. The method derives the remaining InternalDrsObject fields from iRODS
// state, generates a new GUID-backed DRS id, rejects objects that are already marked as DRS
// objects, writes the DRS AVU metadata to the data object, and returns the generated DRS id.
func CreateDrsObjectFromDataObject(filesystem IRODSFilesystem, absolutePath string, mimeType string, description string, aliases []string) (string, error) {
	if filesystem == nil {
		return "", fmt.Errorf("no iRODS filesystem provided")
	}

	correctPath := irodsutil.GetCorrectIRODSPath(strings.TrimSpace(absolutePath))
	if correctPath == "/" || correctPath == "" {
		return "", fmt.Errorf("a data object absolute path is required")
	}

	entry, err := filesystem.StatFile(correctPath)
	if err != nil {
		return "", fmt.Errorf("stat data object %q: %w", correctPath, err)
	}

	metas, err := filesystem.ListMetadata(correctPath)
	if err != nil {
		return "", fmt.Errorf("list metadata for %q: %w", correctPath, err)
	}

	existing, err := internalDrsObjectFromEntry(entry, irodsZoneForPath(filesystem, correctPath), metas)
	if err != nil {
		return "", err
	}

	if existing.Id != correctPath || existing.IsManifest {
		return "", fmt.Errorf("iRODS data object %q is already a DRS object", correctPath)
	}

	drsID, err := newGUID()
	if err != nil {
		return "", fmt.Errorf("generate DRS id: %w", err)
	}

	object := &InternalDrsObject{
		Id:           drsID,
		AbsolutePath: correctPath,
		IrodsZone:    irodsZoneForPath(filesystem, correctPath),
		Size:         entry.Size,
		CreatedTime:  entry.CreateTime,
		UpdatedTime:  entry.ModifyTime,
		MimeType:     normalizedMimeType(correctPath, mimeType),
		Description:  strings.TrimSpace(description),
		Aliases:      normalizedAliases(aliases),
	}

	dataObject := entry.ToDataObject()
	object.Checksum, err = ensureDataObjectChecksum(filesystem, correctPath, dataObject.Replicas)
	if err != nil {
		return "", fmt.Errorf("ensure checksum for %q: %w", correctPath, err)
	}
	if object.Checksum != nil {
		object.Version = object.Checksum.Value
	}

	for _, meta := range metadataForObject(object) {
		if err := filesystem.AddMetadata(correctPath, meta.Name, meta.Value, meta.Units); err != nil {
			return "", fmt.Errorf("apply metadata %q to %q: %w", meta.Name, correctPath, err)
		}
	}

	return drsID, nil
}

// CreateCompoundDrsObjectFromDataObject is deprecated. Compound objects are collection-based
// and should be created with CreateCompoundDrsObjectFromCollection.
func CreateCompoundDrsObjectFromDataObject(filesystem IRODSFilesystem, absolutePath string, description string, aliases []string) (string, error) {
	result, err := CreateCompoundDrsObjectFromCollection(filesystem, absolutePath)
	if err != nil {
		return "", err
	}
	return result.DrsID, nil
}

// GetDrsObjectByID resolves one DRS object by its DRS id and returns the hydrated internal model.
// The lookup searches for data objects carrying the DRS id AVU, validates that exactly one object
// matches with DRS-scoped metadata, and then maps the object's current iRODS state and DRS AVUs
// into InternalDrsObject. Both single-object DRS entries and compound manifest objects are
// returned through the same method.
func GetDrsObjectByID(filesystem IRODSFilesystem, metadataQuerier EntryMetadataQuerier, drsID string) (*InternalDrsObject, error) {
	if filesystem == nil {
		return nil, fmt.Errorf("no iRODS filesystem provided")
	}
	if metadataQuerier == nil {
		return nil, fmt.Errorf("no metadata entry querier provided")
	}

	drsID = strings.TrimSpace(drsID)
	if drsID == "" {
		return nil, fmt.Errorf("a DRS id is required")
	}

	var matches []*irodsfs.Entry
	var cursor *extmetadata.EntryQueryCursor
	for {
		query := extmetadata.NewEntryQuery().
			BothKinds().
			AVU(DrsIdAvuAttrib, drsID, extmetadata.AnyUnit).
			IncludeMatchedAVUs(true).
			Limit(100).
			Cursor(cursor).
			Build()

		result, err := metadataQuerier.QueryMetadataEntries(query)
		if err != nil {
			return nil, fmt.Errorf("search DRS object by id %q: %w", drsID, err)
		}

		for _, entry := range result.Entries {
			if entry == nil || entry.Path == "" {
				continue
			}
			if !hasMatchingDrsIDMatchedAVU(result.MatchedAVUs[entry.Path], drsID) {
				continue
			}
			matches = append(matches, entry)
		}

		if len(matches) > 1 || !result.Page.HasMore {
			break
		}
		if result.Page.Next == nil {
			return nil, fmt.Errorf("search DRS object by id %q returned has_more without a cursor", drsID)
		}
		cursor = result.Page.Next
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("DRS object %q not found", drsID)
	}

	if len(matches) > 1 {
		return nil, fmt.Errorf("DRS id %q matched multiple data objects", drsID)
	}

	entry, err := statEntry(filesystem, matches[0].Path)
	if err != nil {
		return nil, fmt.Errorf("stat iRODS entry %q: %w", matches[0].Path, err)
	}

	object, err := drsObjectFromEntry(filesystem, entry)
	if err != nil {
		return nil, err
	}

	return object, nil
}

// GetDrsObjectByIRODSPath resolves the DRS object metadata currently attached to one iRODS data
// object path. If the target path exists but does not carry DRS metadata, the method returns a
// not-found style error.
func GetDrsObjectByIRODSPath(filesystem IRODSFilesystem, absolutePath string) (*InternalDrsObject, error) {
	if filesystem == nil {
		return nil, fmt.Errorf("no iRODS filesystem provided")
	}

	correctPath := irodsutil.GetCorrectIRODSPath(strings.TrimSpace(absolutePath))
	if correctPath == "/" || correctPath == "" {
		return nil, fmt.Errorf("an iRODS data object absolute path is required")
	}

	entry, err := statEntry(filesystem, correctPath)
	if err != nil {
		return nil, fmt.Errorf("stat iRODS path %q: %w", correctPath, err)
	}

	object, err := drsObjectFromEntry(filesystem, entry)
	if err != nil {
		return nil, err
	}

	if object == nil {
		return nil, fmt.Errorf("iRODS path %q is not a DRS object", correctPath)
	}

	return object, nil
}

// ListDrsObjectsUnderCollection lists DRS-decorated entries contained by an iRODS collection.
// When recursive is false, only direct children of the collection are inspected. When recursive is
// true, subcollections are searched using explicit child and descendant AVU queries. Returned
// objects are sorted by absolute path for stable output.
func ListDrsObjectsUnderCollection(filesystem IRODSFilesystem, metadataQuerier EntryMetadataQuerier, collectionPath string, recursive bool, scope DrsListingScope) ([]*InternalDrsObject, error) {
	if filesystem == nil {
		return nil, fmt.Errorf("no iRODS filesystem provided")
	}
	if metadataQuerier == nil {
		return nil, fmt.Errorf("no metadata entry querier provided")
	}

	correctPath := irodsutil.GetCorrectIRODSPath(strings.TrimSpace(collectionPath))
	if correctPath == "" {
		return nil, fmt.Errorf("an iRODS collection path is required")
	}

	normalizedScope, err := NormalizeDrsListingScope(scope)
	if err != nil {
		return nil, err
	}

	objects, err := listDrsObjectsUnderCollection(filesystem, metadataQuerier, correctPath, recursive, normalizedScope)
	if err != nil {
		return nil, err
	}

	sortDrsObjects(objects)
	return objects, nil
}

// ListDrsObjectsUnderCollectionPage lists DRS-decorated entries under one collection and
// applies zero-based offset/limit paging after sorting by absolute path. Limit must be positive.
func ListDrsObjectsUnderCollectionPage(filesystem IRODSFilesystem, metadataQuerier EntryMetadataQuerier, collectionPath string, recursive bool, scope DrsListingScope, offset int, limit int) (*DrsObjectPage, error) {
	if filesystem == nil {
		return nil, fmt.Errorf("no iRODS filesystem provided")
	}

	if offset < 0 {
		return nil, fmt.Errorf("offset must be zero or greater")
	}

	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	objects, err := ListDrsObjectsUnderCollection(filesystem, metadataQuerier, collectionPath, recursive, scope)
	if err != nil {
		return nil, err
	}

	page := &DrsObjectPage{
		Offset:     offset,
		Limit:      limit,
		Total:      len(objects),
		TotalKnown: true,
	}

	if offset >= len(objects) {
		page.Objects = []*InternalDrsObject{}
		return page, nil
	}

	end := offset + limit
	if end > len(objects) {
		end = len(objects)
	}

	page.Objects = objects[offset:end]
	page.HasMore = end < len(objects)
	return page, nil
}

// ListDrsObjectsUnderCollectionPageFast lists a bounded page of DRS entries under one collection.
// It avoids scanning the entire result set, so Total is intentionally unknown. HasMore is derived
// from one extra fetched row and the metadata query cursor state.
func ListDrsObjectsUnderCollectionPageFast(filesystem IRODSFilesystem, metadataQuerier EntryMetadataQuerier, collectionPath string, recursive bool, scope DrsListingScope, offset int, limit int) (*DrsObjectPage, error) {
	if filesystem == nil {
		return nil, fmt.Errorf("no iRODS filesystem provided")
	}

	if offset < 0 {
		return nil, fmt.Errorf("offset must be zero or greater")
	}

	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	correctPath := irodsutil.GetCorrectIRODSPath(strings.TrimSpace(collectionPath))
	if correctPath == "" {
		return nil, fmt.Errorf("an iRODS collection path is required")
	}

	fetchLimit := offset + limit + 1
	objects, queryHasMore, err := listDrsObjectsUnderCollectionBounded(filesystem, metadataQuerier, correctPath, recursive, scope, fetchLimit)
	if err != nil {
		return nil, err
	}

	sortDrsObjects(objects)

	page := &DrsObjectPage{
		Offset: offset,
		Limit:  limit,
	}

	if offset >= len(objects) {
		page.Objects = []*InternalDrsObject{}
		page.HasMore = queryHasMore
		return page, nil
	}

	end := offset + limit
	if end > len(objects) {
		end = len(objects)
	}

	page.Objects = objects[offset:end]
	page.HasMore = queryHasMore || end < len(objects)
	return page, nil
}

// ListDrsObjects returns one page of DRS entries discovered under the connected zone root.
// The search traverses the zone recursively, sorts results by absolute path, and then applies
// zero-based offset/limit paging. Limit must be positive.
func ListDrsObjects(filesystem IRODSFilesystem, metadataQuerier EntryMetadataQuerier, scope DrsListingScope, offset int, limit int) (*DrsObjectPage, error) {
	if filesystem == nil {
		return nil, fmt.Errorf("no iRODS filesystem provided")
	}

	if offset < 0 {
		return nil, fmt.Errorf("offset must be zero or greater")
	}

	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	objects, err := listDrsObjectsUnderCollection(filesystem, metadataQuerier, rootCollectionPath(filesystem), true, scope)
	if err != nil {
		return nil, err
	}

	sortDrsObjects(objects)

	page := &DrsObjectPage{
		Offset:     offset,
		Limit:      limit,
		Total:      len(objects),
		TotalKnown: true,
	}

	if offset >= len(objects) {
		page.Objects = []*InternalDrsObject{}
		return page, nil
	}

	end := offset + limit
	if end > len(objects) {
		end = len(objects)
	}

	page.Objects = objects[offset:end]
	page.HasMore = end < len(objects)
	return page, nil
}

// UpdateDrsObjectMetadataField updates one supported DRS metadata AVU on an existing DRS object.
// The target must already resolve to a DRS object by iRODS data object path.
func UpdateDrsObjectMetadataField(filesystem IRODSFilesystem, absolutePath string, field DrsMetadataField, value string) error {
	if filesystem == nil {
		return fmt.Errorf("no iRODS filesystem provided")
	}

	object, err := GetDrsObjectByIRODSPath(filesystem, absolutePath)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(field)) == string(DrsMetadataFieldAlias) {
		return fmt.Errorf("alias updates require UpdateDrsObjectAliases")
	}

	attrName, err := drsMetadataAttributeName(field)
	if err != nil {
		return err
	}

	metas, err := filesystem.ListMetadata(object.AbsolutePath)
	if err != nil {
		return fmt.Errorf("list metadata for %q: %w", object.AbsolutePath, err)
	}

	for _, meta := range metas {
		if meta == nil {
			continue
		}

		if meta.Name != attrName {
			continue
		}

		if meta.Units != "" && !strings.EqualFold(meta.Units, DrsAvuUnit) {
			continue
		}

		if err := filesystem.DeleteMetadataByAVU(object.AbsolutePath, meta.Name, meta.Value, meta.Units); err != nil {
			return fmt.Errorf("remove metadata %q from %q: %w", meta.Name, object.AbsolutePath, err)
		}
	}

	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return nil
	}

	if err := filesystem.AddMetadata(object.AbsolutePath, attrName, trimmedValue, DrsAvuUnit); err != nil {
		return fmt.Errorf("apply metadata %q to %q: %w", attrName, object.AbsolutePath, err)
	}

	return nil
}

// UpdateDrsObjectAliases replaces the alias AVUs for an existing DRS object with the provided set.
// Any alias not present in the new set is removed. Duplicate and blank aliases are ignored.
func UpdateDrsObjectAliases(filesystem IRODSFilesystem, absolutePath string, aliases []string) error {
	if filesystem == nil {
		return fmt.Errorf("no iRODS filesystem provided")
	}

	object, err := GetDrsObjectByIRODSPath(filesystem, absolutePath)
	if err != nil {
		return err
	}

	metas, err := filesystem.ListMetadata(object.AbsolutePath)
	if err != nil {
		return fmt.Errorf("list metadata for %q: %w", object.AbsolutePath, err)
	}

	for _, meta := range metas {
		if meta == nil {
			continue
		}

		if meta.Name != DrsAvuAliasAttrib {
			continue
		}

		if meta.Units != "" && !strings.EqualFold(meta.Units, DrsAvuUnit) {
			continue
		}

		if err := filesystem.DeleteMetadataByAVU(object.AbsolutePath, meta.Name, meta.Value, meta.Units); err != nil {
			return fmt.Errorf("remove alias metadata from %q: %w", object.AbsolutePath, err)
		}
	}

	for _, alias := range normalizedAliases(aliases) {
		if err := filesystem.AddMetadata(object.AbsolutePath, DrsAvuAliasAttrib, alias, DrsAvuUnit); err != nil {
			return fmt.Errorf("apply alias metadata to %q: %w", object.AbsolutePath, err)
		}
	}

	return nil
}

// RemoveSingleDrsObjectFromDataObject strips DRS-related AVUs from a single-object DRS data object
// without deleting the underlying iRODS data object.
//
// User guide:
// Call this when an existing iRODS data object was previously decorated as one atomic DRS object
// via CreateDrsObjectFromDataObject and you want to remove only the DRS registration metadata.
// This method is idempotent: if the target object is not currently marked as a DRS object, it
// returns success without making changes. This method must not be used for compound DRS objects;
// if the target carries the compound-manifest AVU, the method returns an error and leaves metadata
// unchanged.
func RemoveSingleDrsObjectFromDataObject(filesystem IRODSFilesystem, absolutePath string) error {
	if filesystem == nil {
		return fmt.Errorf("no iRODS filesystem provided")
	}

	correctPath := irodsutil.GetCorrectIRODSPath(strings.TrimSpace(absolutePath))
	if correctPath == "/" || correctPath == "" {
		return fmt.Errorf("a data object absolute path is required")
	}

	entry, err := filesystem.StatFile(correctPath)
	if err != nil {
		return fmt.Errorf("stat data object %q: %w", correctPath, err)
	}

	metas, err := filesystem.ListMetadata(correctPath)
	if err != nil {
		return fmt.Errorf("list metadata for %q: %w", correctPath, err)
	}

	object, err := internalDrsObjectFromEntry(entry, irodsZoneForPath(filesystem, correctPath), metas)
	if err != nil {
		return err
	}

	if object.IsManifest {
		return fmt.Errorf("iRODS data object %q is a compound DRS manifest and cannot be removed with RemoveSingleDrsObjectFromDataObject", correctPath)
	}

	for _, meta := range metas {
		if !isDrsMetadata(meta) {
			continue
		}

		if err := filesystem.DeleteMetadataByAVU(correctPath, meta.Name, meta.Value, meta.Units); err != nil {
			return fmt.Errorf("remove metadata %q from %q: %w", meta.Name, correctPath, err)
		}
	}

	return nil
}

func listDrsObjectsUnderCollection(filesystem IRODSFilesystem, metadataQuerier EntryMetadataQuerier, collectionPath string, recursive bool, scope DrsListingScope) ([]*InternalDrsObject, error) {
	objects, _, err := listDrsObjectsUnderCollectionBounded(filesystem, metadataQuerier, collectionPath, recursive, scope, 0)
	return objects, err
}

func listDrsObjectsUnderCollectionBounded(filesystem IRODSFilesystem, metadataQuerier EntryMetadataQuerier, collectionPath string, recursive bool, scope DrsListingScope, maxEntries int) ([]*InternalDrsObject, bool, error) {
	if metadataQuerier == nil {
		return nil, false, fmt.Errorf("no metadata entry querier provided")
	}

	normalizedScope, err := NormalizeDrsListingScope(scope)
	if err != nil {
		return nil, false, err
	}

	entriesByPath, queryHasMore, err := queryDrsListingEntriesBounded(metadataQuerier, collectionPath, recursive, normalizedScope, maxEntries)
	if err != nil {
		return nil, false, err
	}

	paths := make([]string, 0, len(entriesByPath))
	for entryPath := range entriesByPath {
		paths = append(paths, entryPath)
	}
	sort.Strings(paths)

	objects := make([]*InternalDrsObject, 0, len(paths))
	for _, entryPath := range paths {
		listingEntry := entriesByPath[entryPath]
		entry := listingEntry.Entry
		if entry == nil {
			continue
		}

		object, err := drsObjectFromListingEntry(filesystem, entry, listingEntry.MatchedAVUs)
		if err != nil {
			return nil, false, err
		}
		if object == nil || !drsListingScopeIncludesObject(normalizedScope, entry, object) {
			continue
		}

		objects = append(objects, object)
	}

	return objects, queryHasMore, nil
}

func queryDrsListingEntries(metadataQuerier EntryMetadataQuerier, collectionPath string, recursive bool, scope DrsListingScope) (map[string]drsListingEntry, error) {
	entriesByPath, _, err := queryDrsListingEntriesBounded(metadataQuerier, collectionPath, recursive, scope, 0)
	return entriesByPath, err
}

func queryDrsListingEntriesBounded(metadataQuerier EntryMetadataQuerier, collectionPath string, recursive bool, scope DrsListingScope, maxEntries int) (map[string]drsListingEntry, bool, error) {
	entriesByPath := map[string]drsListingEntry{}
	hasMore := false
	for _, kind := range drsListingEntryKinds(scope) {
		var cursor *extmetadata.EntryQueryCursor
		for {
			queryBuilder := extmetadata.NewEntryQuery().
				AVUAttrib(drsListingAVUAttributePattern).
				IncludeMatchedAVUs(true).
				Limit(drsListingQueryLimit(maxEntries)).
				Cursor(cursor)

			switch kind {
			case extmetadata.EntryKindCollection:
				queryBuilder.Collections()
				if recursive {
					queryBuilder.Like(extmetadata.FieldPath, collectionPath+"/%")
				} else {
					queryBuilder.Scope(collectionPath, extmetadata.EntryQueryScopeChildren)
				}
			case extmetadata.EntryKindDataObject:
				queryBuilder.DataObjects()
				if recursive {
					queryBuilder.Like(extmetadata.FieldPath, collectionPath+"%/%")
				} else {
					queryBuilder.Scope(collectionPath, extmetadata.EntryQueryScopeChildren)
				}
			}

			result, err := metadataQuerier.QueryMetadataEntries(queryBuilder.Build())
			if err != nil {
				return nil, false, fmt.Errorf("query DRS listing metadata under %q: %w", collectionPath, err)
			}

			for _, entry := range result.Entries {
				if entry == nil || strings.TrimSpace(entry.Path) == "" {
					continue
				}
				if !drsListingEntryWithinScope(entry.Path, collectionPath, recursive) {
					continue
				}
				matchedAVUs := result.MatchedAVUs[entry.Path]
				if !hasAnyDrsIDMatchedAVU(matchedAVUs) {
					continue
				}
				listingEntry := entriesByPath[entry.Path]
				if listingEntry.Entry == nil {
					listingEntry.Entry = entry
				}
				listingEntry.MatchedAVUs = appendUniqueAVUStats(listingEntry.MatchedAVUs, matchedAVUs)
				entriesByPath[entry.Path] = listingEntry
			}

			if maxEntries > 0 && len(entriesByPath) > maxEntries {
				hasMore = true
				break
			}

			if !result.Page.HasMore {
				break
			}
			if result.Page.Next == nil {
				return nil, false, fmt.Errorf("query DRS listing metadata under %q returned has_more without a cursor", collectionPath)
			}
			if maxEntries > 0 && len(entriesByPath) >= maxEntries {
				hasMore = true
				break
			}
			cursor = result.Page.Next
		}
	}

	return entriesByPath, hasMore, nil
}

func drsListingQueryLimit(maxEntries int) int {
	if maxEntries <= 0 || maxEntries > drsListingEntryQueryLimit {
		return drsListingEntryQueryLimit
	}
	return maxEntries
}

func drsListingEntryKinds(scope DrsListingScope) []extmetadata.EntryKind {
	switch scope {
	case DrsListingScopeObjects:
		return []extmetadata.EntryKind{extmetadata.EntryKindDataObject}
	case DrsListingScopeCompound:
		return []extmetadata.EntryKind{extmetadata.EntryKindCollection}
	default:
		return []extmetadata.EntryKind{extmetadata.EntryKindCollection, extmetadata.EntryKindDataObject}
	}
}

func drsListingEntryWithinScope(entryPath string, collectionPath string, recursive bool) bool {
	entryPath = path.Clean(strings.TrimSpace(entryPath))
	collectionPath = path.Clean(strings.TrimSpace(collectionPath))
	if entryPath == "" || entryPath == "." || collectionPath == "" || collectionPath == "." {
		return false
	}
	if recursive {
		return strings.HasPrefix(entryPath, strings.TrimRight(collectionPath, "/")+"/")
	}
	return path.Dir(entryPath) == collectionPath
}

func drsListingScopeIncludesObject(scope DrsListingScope, entry *irodsfs.Entry, object *InternalDrsObject) bool {
	if entry == nil || object == nil {
		return false
	}

	if entry.IsDir() {
		return scope != DrsListingScopeObjects && object.IsManifest
	}

	return scope != DrsListingScopeCompound
}

// internalDrsObjectFromEntry builds an InternalDrsObject from an iRODS file entry plus any DRS AVUs
// already attached to the object.
func internalDrsObjectFromEntry(entry *irodsfs.Entry, irodsZone string, metas []*irodstypes.IRODSMeta) (*InternalDrsObject, error) {
	if entry == nil {
		return nil, fmt.Errorf("no iRODS entry provided")
	}

	object := &InternalDrsObject{
		AbsolutePath: entry.Path,
		IrodsZone:    irodsZone,
		Size:         entry.Size,
		CreatedTime:  entry.CreateTime,
		UpdatedTime:  entry.ModifyTime,
	}

	if err := ApplyDrsMetadata(object, metas); err != nil {
		return nil, err
	}

	if object.Id == "" {
		object.Id = entry.Path
	}

	dataObject := entry.ToDataObject()
	object.Replicas = replicasFromDataObject(dataObject)
	if len(dataObject.Replicas) > 0 && dataObject.Replicas[0] != nil {
		object.ResourceName = strings.TrimSpace(dataObject.Replicas[0].ResourceName)
		object.Checksum = checksumFromReplica(dataObject.Replicas[0])
	}

	if object.Version == "" && object.Checksum != nil {
		object.Version = object.Checksum.Value
	}

	return object, nil
}

func replicasFromDataObject(dataObject *irodstypes.IRODSDataObject) []InternalReplica {
	if dataObject == nil || len(dataObject.Replicas) == 0 {
		return nil
	}

	replicas := make([]InternalReplica, 0, len(dataObject.Replicas))
	for _, replica := range dataObject.Replicas {
		if replica == nil {
			continue
		}

		replicas = append(replicas, InternalReplica{
			ResourceName:      strings.TrimSpace(replica.ResourceName),
			ResourceHierarchy: strings.TrimSpace(replica.ResourceHierarchy),
			Path:              strings.TrimSpace(replica.Path),
			Status:            strings.TrimSpace(replica.Status),
		})
	}

	return replicas
}

// checksumFromReplica converts a go-irodsclient replica checksum into the internal checksum model,
// preserving both the checksum algorithm type and the checksum value.
func checksumFromReplica(replica *irodstypes.IRODSReplica) *InternalChecksum {
	if replica == nil || replica.Checksum == nil || replica.Checksum.IRODSChecksumString == "" {
		return nil
	}

	return &InternalChecksum{
		Type:  normalizeChecksumType(replica.Checksum.Algorithm),
		Value: normalizeChecksumValue(replica.Checksum.IRODSChecksumString),
	}
}

func checksumFromIRODSChecksum(checksum *irodstypes.IRODSChecksum) *InternalChecksum {
	if checksum == nil || checksum.IRODSChecksumString == "" {
		return nil
	}

	return &InternalChecksum{
		Type:  normalizeChecksumType(checksum.Algorithm),
		Value: normalizeChecksumValue(checksum.IRODSChecksumString),
	}
}

func normalizeChecksumType(algorithm irodstypes.ChecksumAlgorithm) string {
	switch algorithm {
	case irodstypes.ChecksumAlgorithmSHA1:
		return "sha-1"
	case irodstypes.ChecksumAlgorithmSHA256:
		return "sha-256"
	case irodstypes.ChecksumAlgorithmSHA512:
		return "sha-512"
	case irodstypes.ChecksumAlgorithmADLER32:
		return "adler32"
	case irodstypes.ChecksumAlgorithmMD5:
		return "md5"
	default:
		return strings.ToLower(string(algorithm))
	}
}

func normalizeChecksumValue(irodsChecksum string) string {
	trimmed := strings.TrimSpace(irodsChecksum)
	if trimmed == "" {
		return ""
	}

	// iRODS checksums often have algorithm prefixes like 'sha2:', 'md5:', etc.
	// Strip the prefix if present.
	parts := strings.SplitN(trimmed, ":", 2)
	checksumPart := trimmed
	if len(parts) == 2 {
		checksumPart = parts[1]
	}

	// GA4GH DRS expects hex-encoded checksums for many algorithms.
	// iRODS typically uses base64 for SHA-256 and MD5.
	// We try to decode as base64. If it works and it's not already hex-like,
	// we convert it to hex.

	// If it's already a valid hex string of common digest lengths (32, 64, etc),
	// we assume it is already normalized.
	if isHexString(checksumPart) {
		return strings.ToLower(checksumPart)
	}

	// Attempt base64 decoding.
	decoded, err := base64.StdEncoding.DecodeString(checksumPart)
	if err == nil {
		// Successfully decoded from base64. Re-encode as hex.
		return hex.EncodeToString(decoded)
	}

	// Fallback to original stripped value if not base64 or hex.
	return checksumPart
}

// isHexString reports whether a string represents a valid hex digest.
func isHexString(s string) bool {
	if len(s) == 0 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func ensureDataObjectChecksum(filesystem IRODSFilesystem, absolutePath string, replicas []*irodstypes.IRODSReplica) (*InternalChecksum, error) {
	for _, replica := range replicas {
		if checksum := checksumFromReplica(replica); checksum != nil {
			return checksum, nil
		}
	}

	switch fs := any(filesystem).(type) {
	case checksumEnsuringFilesystem:
		checksum, err := fs.EnsureDataObjectChecksum(absolutePath)
		if err != nil {
			return nil, err
		}
		return checksumFromIRODSChecksum(checksum), nil
	case *irodsfs.FileSystem:
		conn, err := fs.GetMetadataConnection(true)
		if err != nil {
			return nil, fmt.Errorf("get metadata connection: %w", err)
		}
		defer func() {
			_ = fs.ReturnMetadataConnection(conn)
		}()

		checksum, err := irodslowfs.GetDataObjectChecksum(conn, absolutePath, "")
		if err != nil {
			return nil, err
		}
		return checksumFromIRODSChecksum(checksum), nil
	default:
		return nil, nil
	}
}

// metadataForObject renders an InternalDrsObject into the AVUs that should be stored on the iRODS object.
func metadataForObject(object *InternalDrsObject) []*irodstypes.IRODSMeta {
	metas := []*irodstypes.IRODSMeta{
		{Name: DrsIdAvuAttrib, Value: object.Id, Units: DrsAvuUnit},
	}

	if object.Version != "" {
		metas = append(metas, &irodstypes.IRODSMeta{Name: DrsAvuVersionAttrib, Value: object.Version, Units: DrsAvuUnit})
	}

	if object.MimeType != "" {
		metas = append(metas, &irodstypes.IRODSMeta{Name: DrsAvuMimeTypeAttrib, Value: object.MimeType, Units: DrsAvuUnit})
	}

	if object.Description != "" {
		metas = append(metas, &irodstypes.IRODSMeta{Name: DrsAvuDescriptionAttrib, Value: object.Description, Units: DrsAvuUnit})
	}

	for _, alias := range normalizedAliases(object.Aliases) {
		metas = append(metas, &irodstypes.IRODSMeta{Name: DrsAvuAliasAttrib, Value: alias, Units: DrsAvuUnit})
	}

	if object.IsManifest {
		metas = append(metas, &irodstypes.IRODSMeta{Name: DrsAvuCompoundManifestAttrib, Value: "true", Units: DrsAvuUnit})
	}

	return metas
}

func normalizedMimeType(dataObjectPath string, mimeType string) string {
	mimeType = strings.TrimSpace(mimeType)
	if mimeType != "" {
		return mimeType
	}

	return MimeTypeSupport{}.DeriveFromDataObjectPath(dataObjectPath)
}

func drsMetadataAttributeName(field DrsMetadataField) (string, error) {
	switch strings.TrimSpace(string(field)) {
	case string(DrsMetadataFieldMimeType), "mime-type", "mime":
		return DrsAvuMimeTypeAttrib, nil
	case string(DrsMetadataFieldVersion):
		return DrsAvuVersionAttrib, nil
	case string(DrsMetadataFieldDescription):
		return DrsAvuDescriptionAttrib, nil
	case string(DrsMetadataFieldAlias):
		return DrsAvuAliasAttrib, nil
	default:
		return "", fmt.Errorf("unsupported DRS metadata field %q", field)
	}
}

func isDrsMetadata(meta *irodstypes.IRODSMeta) bool {
	if meta == nil {
		return false
	}

	switch meta.Name {
	case DrsIdAvuAttrib, DrsAvuVersionAttrib, DrsAvuMimeTypeAttrib, DrsAvuCompoundManifestAttrib, DrsAvuAliasAttrib, DrsAvuDescriptionAttrib:
		return meta.Units == "" || strings.EqualFold(meta.Units, DrsAvuUnit)
	default:
		return false
	}
}

func drsObjectFromEntry(filesystem IRODSFilesystem, entry *irodsfs.Entry) (*InternalDrsObject, error) {
	if entry == nil {
		return nil, fmt.Errorf("no iRODS entry provided")
	}

	metas, err := filesystem.ListMetadata(entry.Path)
	if err != nil {
		return nil, fmt.Errorf("list metadata for %q: %w", entry.Path, err)
	}

	return drsObjectFromEntryWithMetadata(filesystem, entry, metas)
}

func drsObjectFromEntryWithMetadata(filesystem IRODSFilesystem, entry *irodsfs.Entry, metas []*irodstypes.IRODSMeta) (*InternalDrsObject, error) {
	if entry == nil {
		return nil, fmt.Errorf("no iRODS entry provided")
	}

	entry, err := entryWithAllReplicas(filesystem, entry)
	if err != nil {
		return nil, err
	}

	object, err := internalDrsObjectFromEntry(entry, irodsZoneForPath(filesystem, entry.Path), metas)
	if err != nil {
		return nil, err
	}

	if !hasMatchingDrsIDMetadata(metas, object.Id) {
		return nil, nil
	}

	if object.IsManifest && entry.IsDir() {
		checksum, err := BuildCompoundRuntimeManifestChecksum(filesystem, object.AbsolutePath)
		if err != nil {
			return nil, fmt.Errorf("build compound runtime manifest checksum for %q: %w", object.AbsolutePath, err)
		}
		object.Checksum = checksum
	}

	return object, nil
}

func drsObjectFromListingEntry(filesystem IRODSFilesystem, entry *irodsfs.Entry, avus []extmetadata.AVUStat) (*InternalDrsObject, error) {
	if entry == nil {
		return nil, fmt.Errorf("no iRODS entry provided")
	}

	metas := irodsMetasFromAVUStats(avus)
	object, err := internalDrsObjectFromEntry(entry, irodsZoneForPath(filesystem, entry.Path), metas)
	if err != nil {
		return nil, err
	}

	if !hasMatchingDrsIDMetadata(metas, object.Id) {
		return nil, nil
	}

	return object, nil
}

func irodsMetasFromAVUStats(avus []extmetadata.AVUStat) []*irodstypes.IRODSMeta {
	metas := make([]*irodstypes.IRODSMeta, 0, len(avus))
	for _, avu := range avus {
		metas = append(metas, &irodstypes.IRODSMeta{
			Name:  avu.Name,
			Value: avu.Value,
			Units: avu.Units,
		})
	}
	return metas
}

func entryWithAllReplicas(filesystem IRODSFilesystem, entry *irodsfs.Entry) (*irodsfs.Entry, error) {
	if entry == nil || strings.TrimSpace(entry.Path) == "" {
		return entry, nil
	}
	if entry.IsDir() {
		return entry, nil
	}

	switch fs := any(filesystem).(type) {
	case allReplicaStatFilesystem:
		fullEntry, err := fs.StatFileWithAllReplicas(entry.Path)
		if err != nil {
			return nil, fmt.Errorf("stat data object with all replicas %q: %w", entry.Path, err)
		}
		if fullEntry != nil {
			return fullEntry, nil
		}
	case *irodsfs.FileSystem:
		conn, err := fs.GetMetadataConnection(true)
		if err != nil {
			return nil, fmt.Errorf("get metadata connection: %w", err)
		}
		defer func() {
			_ = fs.ReturnMetadataConnection(conn)
		}()

		dataObject, err := irodslowfs.GetDataObject(conn, entry.Path)
		if err != nil {
			return nil, fmt.Errorf("stat data object with all replicas %q: %w", entry.Path, err)
		}
		if dataObject != nil && len(dataObject.Replicas) > 0 {
			return irodsfs.NewEntryFromDataObject(dataObject), nil
		}
	}

	return entry, nil
}

func hasMatchingDrsIDMetadata(metas []*irodstypes.IRODSMeta, drsID string) bool {
	for _, meta := range metas {
		if meta == nil {
			continue
		}

		if meta.Name != DrsIdAvuAttrib {
			continue
		}

		if strings.TrimSpace(meta.Value) != drsID {
			continue
		}

		if meta.Units == "" || strings.EqualFold(meta.Units, DrsAvuUnit) {
			return true
		}
	}

	return false
}

func hasMatchingDrsIDMatchedAVU(avus []extmetadata.AVUStat, drsID string) bool {
	for _, avu := range avus {
		if avu.Name != DrsIdAvuAttrib {
			continue
		}

		if strings.TrimSpace(avu.Value) != drsID {
			continue
		}

		if avu.Units == "" || strings.EqualFold(avu.Units, DrsAvuUnit) {
			return true
		}
	}

	return false
}

func hasAnyDrsIDMatchedAVU(avus []extmetadata.AVUStat) bool {
	for _, avu := range avus {
		if avu.Name != DrsIdAvuAttrib {
			continue
		}
		if strings.TrimSpace(avu.Value) == "" {
			continue
		}
		if avu.Units == "" || strings.EqualFold(avu.Units, DrsAvuUnit) {
			return true
		}
	}

	return false
}

func appendUniqueAVUStats(existing []extmetadata.AVUStat, additions []extmetadata.AVUStat) []extmetadata.AVUStat {
	for _, avu := range additions {
		found := false
		for _, candidate := range existing {
			if candidate.Name == avu.Name && candidate.Value == avu.Value && candidate.Units == avu.Units {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, avu)
		}
	}
	return existing
}

func rootCollectionPath(filesystem IRODSFilesystem) string {
	if filesystem != nil {
		if account := filesystem.GetAccount(); account != nil {
			zone := strings.TrimSpace(account.ClientZone)
			if zone != "" {
				return "/" + strings.TrimPrefix(zone, "/")
			}
		}
	}

	return "/"
}

func sortDrsObjects(objects []*InternalDrsObject) {
	sort.Slice(objects, func(i int, j int) bool {
		if objects[i].AbsolutePath == objects[j].AbsolutePath {
			return objects[i].Id < objects[j].Id
		}

		return objects[i].AbsolutePath < objects[j].AbsolutePath
	})
}

// normalizedAliases trims aliases and drops empty values before they are persisted as AVUs.
func normalizedAliases(aliases []string) []string {
	normalized := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		trimmed := strings.TrimSpace(alias)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}

	return normalized
}

// irodsZoneForPath returns the zone from the connected filesystem account when available and falls
// back to extracting the zone from the iRODS absolute path.
func irodsZoneForPath(filesystem IRODSFilesystem, absolutePath string) string {
	if filesystem != nil {
		if account := filesystem.GetAccount(); account != nil && account.ClientZone != "" {
			return account.ClientZone
		}
	}

	zone, err := irodsutil.GetIRODSZone(absolutePath)
	if err != nil {
		return ""
	}

	return zone
}

// newGUID generates a random RFC 4122 version 4 identifier string for use as a DRS id.
func newGUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	), nil
}
