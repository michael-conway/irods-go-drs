package drs_support

import (
	"fmt"
	"strings"
	"time"

	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
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

func NewInternalDrsObjectFromDataObject(dataObject *irodstypes.IRODSDataObject, irodsZone string, metas []*irodstypes.IRODSMeta) (*InternalDrsObject, error) {
	if dataObject == nil {
		return nil, fmt.Errorf("no iRODS data object provided")
	}

	object := &InternalDrsObject{
		AbsolutePath: dataObject.Path,
		IrodsZone:    irodsZone,
		Size:         dataObject.Size,
	}

	if err := ApplyDrsMetadata(object, metas); err != nil {
		return nil, err
	}

	if object.Id == "" {
		object.Id = dataObject.Path
	}

	if len(dataObject.Replicas) > 0 && dataObject.Replicas[0] != nil {
		object.CreatedTime = dataObject.Replicas[0].CreateTime
		object.UpdatedTime = dataObject.Replicas[0].ModifyTime
		object.Checksum = checksumFromReplica(dataObject.Replicas[0])
	}

	if object.Version == "" && object.Checksum != nil {
		object.Version = object.Checksum.Value
	}

	return object, nil
}

func checksumFromReplica(replica *irodstypes.IRODSReplica) *InternalChecksum {
	if replica == nil || replica.Checksum == nil || replica.Checksum.IRODSChecksumString == "" {
		return nil
	}

	return &InternalChecksum{
		Type:  string(replica.Checksum.Algorithm),
		Value: replica.Checksum.IRODSChecksumString,
	}
}
