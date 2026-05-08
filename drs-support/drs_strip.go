package drs_support

import (
	"fmt"
	"strings"

	irodsutil "github.com/cyverse/go-irodsclient/irods/util"
)

type StripDrsResult struct {
	RootPath             string `json:"rootPath"`
	PathsVisited         int    `json:"pathsVisited"`
	PathsWithDrsMetadata int    `json:"pathsWithDrsMetadata"`
	AvusRemoved          int    `json:"avusRemoved"`
}

// StripDrsSemantics removes DRS AVUs from a single iRODS data object path or a
// collection subtree. The target data objects/collections are preserved; only
// DRS metadata is removed.
func StripDrsSemantics(filesystem IRODSFilesystem, irodsPath string) (*StripDrsResult, error) {
	if filesystem == nil {
		return nil, fmt.Errorf("no iRODS filesystem provided")
	}

	correctPath := irodsutil.GetCorrectIRODSPath(strings.TrimSpace(irodsPath))
	if correctPath == "" || correctPath == "/" {
		return nil, fmt.Errorf("an iRODS path is required")
	}

	entry, err := statEntry(filesystem, correctPath)
	if err != nil {
		return nil, fmt.Errorf("stat iRODS path %q: %w", correctPath, err)
	}

	result := &StripDrsResult{
		RootPath: correctPath,
	}
	if err := stripDrsSemanticsRecursive(filesystem, entry.Path, entry.IsDir(), result); err != nil {
		return nil, err
	}
	return result, nil
}

func stripDrsSemanticsRecursive(filesystem IRODSFilesystem, irodsPath string, isDir bool, result *StripDrsResult) error {
	result.PathsVisited++

	metas, err := filesystem.ListMetadata(irodsPath)
	if err != nil {
		return fmt.Errorf("list metadata for %q: %w", irodsPath, err)
	}

	removedFromPath := 0
	for _, meta := range metas {
		if !isDrsMetadata(meta) {
			continue
		}
		if err := filesystem.DeleteMetadataByAVU(irodsPath, meta.Name, meta.Value, meta.Units); err != nil {
			return fmt.Errorf("remove metadata %q from %q: %w", meta.Name, irodsPath, err)
		}
		removedFromPath++
	}

	if removedFromPath > 0 {
		result.PathsWithDrsMetadata++
		result.AvusRemoved += removedFromPath
	}

	if !isDir {
		return nil
	}

	children, err := filesystem.List(irodsPath)
	if err != nil {
		return fmt.Errorf("list collection %q: %w", irodsPath, err)
	}
	for _, child := range children {
		if child == nil || strings.TrimSpace(child.Path) == "" {
			continue
		}
		if err := stripDrsSemanticsRecursive(filesystem, child.Path, child.IsDir(), result); err != nil {
			return err
		}
	}
	return nil
}
