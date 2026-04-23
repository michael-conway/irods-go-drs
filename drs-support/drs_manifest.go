package drs_support

import (
	"encoding/json"
	"fmt"
	"strings"
)

const DefaultDrsManifestSchema = "irods-drs-manifest/v1"

type DrsManifest struct {
	Schema      string             `json:"schema,omitempty"`
	Type        string             `json:"type,omitempty"`
	Description string             `json:"description,omitempty"`
	Contents    []DrsManifestEntry `json:"contents,omitempty"`
}

type DrsManifestEntry struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	Role string `json:"role,omitempty"`
}

func ParseDrsManifest(data []byte) (*DrsManifest, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("manifest content is empty")
	}

	var manifest DrsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}

	if manifest.Schema == "" {
		manifest.Schema = DefaultDrsManifestSchema
	}

	return &manifest, nil
}

func (m *DrsManifest) Validate() []string {
	if m == nil {
		return []string{"manifest is nil"}
	}

	var issues []string
	if strings.TrimSpace(m.Schema) == "" {
		issues = append(issues, "manifest schema is empty")
	}

	for i, entry := range m.Contents {
		if strings.TrimSpace(entry.ID) == "" {
			issues = append(issues, fmt.Sprintf("manifest entry %d is missing id", i))
		}
	}

	return issues
}
