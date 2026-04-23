package drs_support

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	irodsutil "github.com/cyverse/go-irodsclient/irods/util"
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

type InternalDrsObject struct {
	// An identifier unique to this DrsObject.
	Id string
	// iRODS logical path to the data object.
	AbsolutePath string
	// Zone that contains object.
	IrodsZone string
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

type IRODSFilesystem interface {
	StatFile(irodsPath string) (*irodsfs.Entry, error)
	ListMetadata(irodsPath string) ([]*irodstypes.IRODSMeta, error)
	AddMetadata(irodsPath string, attName string, attValue string, attUnits string) error
	GetAccount() *irodstypes.IRODSAccount
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
		MimeType:     strings.TrimSpace(mimeType),
		Description:  strings.TrimSpace(description),
		Aliases:      normalizedAliases(aliases),
	}

	dataObject := entry.ToDataObject()
	if len(dataObject.Replicas) > 0 && dataObject.Replicas[0] != nil {
		object.Checksum = checksumFromReplica(dataObject.Replicas[0])
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

// CreateCompoundDrsObjectFromDataObject is the placeholder entry point for manifest-backed
// compound DRS objects and currently validates only the minimal required inputs.
func CreateCompoundDrsObjectFromDataObject(filesystem IRODSFilesystem, absolutePath string, description string, aliases []string) (string, error) {
	if filesystem == nil {
		return "", fmt.Errorf("no iRODS filesystem provided")
	}

	if strings.TrimSpace(absolutePath) == "" {
		return "", fmt.Errorf("a data object absolute path is required")
	}

	return "", fmt.Errorf("compound DRS object creation is not implemented")
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
	if len(dataObject.Replicas) > 0 && dataObject.Replicas[0] != nil {
		object.Checksum = checksumFromReplica(dataObject.Replicas[0])
	}

	if object.Version == "" && object.Checksum != nil {
		object.Version = object.Checksum.Value
	}

	return object, nil
}

// checksumFromReplica converts a go-irodsclient replica checksum into the internal checksum model,
// preserving both the checksum algorithm type and the checksum value.
func checksumFromReplica(replica *irodstypes.IRODSReplica) *InternalChecksum {
	if replica == nil || replica.Checksum == nil || replica.Checksum.IRODSChecksumString == "" {
		return nil
	}

	return &InternalChecksum{
		Type:  strings.ToLower(string(replica.Checksum.Algorithm)),
		Value: replica.Checksum.IRODSChecksumString,
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
