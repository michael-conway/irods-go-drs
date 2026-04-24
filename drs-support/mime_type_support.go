package drs_support

// TODO: this is provisional, consider migrating to a general utility package that builds on the go-irodsclient library.
// TODO: consider adding an external configuration file that can help define mime types, potentially as a
// public file stored in irods

import (
	"mime"
	"path/filepath"
	"strings"
)

// MimeTypeSupport resolves MIME types from data object paths using the Go standard library.
type MimeTypeSupport struct{}

var extensionMimeTypes = map[string]string{
	".md": "text/markdown",
}

// DeriveFromDataObjectPath returns the MIME type for a data object path based on its file extension.
// Unknown or extensionless paths return the empty string.
func (s MimeTypeSupport) DeriveFromDataObjectPath(dataObjectPath string) string {
	ext := strings.TrimSpace(filepath.Ext(strings.TrimSpace(dataObjectPath)))
	if ext == "" {
		return ""
	}

	if mimeType, ok := extensionMimeTypes[strings.ToLower(ext)]; ok {
		return mimeType
	}

	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		return ""
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return contentType
	}

	return mediaType
}

// DeriveMimeTypeFromDataObjectPath is the package-level helper for MIME type resolution.
func DeriveMimeTypeFromDataObjectPath(dataObjectPath string) string {
	return MimeTypeSupport{}.DeriveFromDataObjectPath(dataObjectPath)
}
