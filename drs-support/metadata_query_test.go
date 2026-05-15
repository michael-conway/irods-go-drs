package drs_support

import (
	"path"
	"strings"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	extmetadata "github.com/michael-conway/go-irodsclient-extensions/metadata"
)

func (f *compoundTestFilesystem) QueryMetadataEntries(query extmetadata.EntryQuery) (extmetadata.EntryQueryResult, error) {
	return queryMetadataEntriesFromMaps(query, f.entriesByPath, f.metadataByPath)
}

func (f *accessMethodsTestFilesystem) QueryMetadataEntries(query extmetadata.EntryQuery) (extmetadata.EntryQueryResult, error) {
	return queryMetadataEntriesFromMaps(query, nil, f.metadataByPath)
}

func queryMetadataEntriesFromMaps(query extmetadata.EntryQuery, entries map[string]*irodsfs.Entry, metadataByPath map[string][]*irodstypes.IRODSMeta) (extmetadata.EntryQueryResult, error) {
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

	for irodsPath, metas := range metadataByPath {
		entry := entries[irodsPath]
		if entry == nil {
			continue
		}
		if !queryIncludesEntryKind(normalized, entry) || !queryEntryInScope(normalized, entry) {
			continue
		}

		matched := matchedAVUsForQuery(normalized, metas)
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

func queryIncludesEntryKind(query extmetadata.EntryQuery, entry *irodsfs.Entry) bool {
	if entry == nil {
		return false
	}
	if entry.IsDir() {
		return extmetadata.EntryQueryHasKind(query, extmetadata.EntryKindCollection)
	}
	return extmetadata.EntryQueryHasKind(query, extmetadata.EntryKindDataObject)
}

func matchedAVUsForQuery(query extmetadata.EntryQuery, metas []*irodstypes.IRODSMeta) []extmetadata.AVUStat {
	matched := []extmetadata.AVUStat{}
	for _, meta := range metas {
		if meta == nil {
			continue
		}
		if !metaMatchesConditions(meta, query.Conditions) {
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

func queryEntryInScope(query extmetadata.EntryQuery, entry *irodsfs.Entry) bool {
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

func metaMatchesConditions(meta *irodstypes.IRODSMeta, conditions []extmetadata.EntryCondition) bool {
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
