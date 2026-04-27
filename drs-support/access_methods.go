package drs_support

import (
	"path"
	"path/filepath"
	"strings"
)

type DrsAccessMethod struct {
	Type               string
	URL                string
	AccessID           string
	Cloud              string
	Region             string
	Available          bool
	SupportedAuthTypes []string
	BearerAuthIssuers  []string
}

func BuildAccessMethods(cfg *DrsConfig, object *InternalDrsObject) []DrsAccessMethod {
	if cfg == nil || object == nil {
		return nil
	}

	if usesStructuredAccessMethodConfig(cfg) {
		methods := make([]DrsAccessMethod, 0, 3)
		if accessMethod, ok := buildHTTPSAccessMethod(cfg, object); ok {
			methods = append(methods, accessMethod)
		}
		if accessMethod, ok := buildIRODSAccessMethod(cfg, object); ok {
			methods = append(methods, accessMethod)
		}
		if accessMethod, ok := buildFileAccessMethod(cfg, object); ok {
			methods = append(methods, accessMethod)
		}
		return methods
	}

	methods := make([]DrsAccessMethod, 0, len(cfg.AccessMethods))
	for _, method := range cfg.AccessMethods {
		switch strings.ToLower(strings.TrimSpace(method)) {
		case "http":
			if accessMethod, ok := buildLegacyHTTPAccessMethod(cfg, object); ok {
				methods = append(methods, accessMethod)
			}
		case "irods":
			if accessMethod, ok := buildIRODSAccessMethod(cfg, object); ok {
				methods = append(methods, accessMethod)
			}
		case "local":
			if accessMethod, ok := buildLocalAccessMethod(cfg, object); ok {
				methods = append(methods, accessMethod)
			}
		case "s3":
			if accessMethod, ok := buildS3AccessMethod(cfg, object); ok {
				methods = append(methods, accessMethod)
			}
		}
	}

	return methods
}

func usesStructuredAccessMethodConfig(cfg *DrsConfig) bool {
	if cfg == nil {
		return false
	}

	return cfg.HttpsAccessMethodSupported || cfg.IrodsAccessMethodSupported || cfg.FileAccessMethodSupported || strings.TrimSpace(cfg.HttpsAccessMethodBaseURL) != ""
}

func buildHTTPSAccessMethod(cfg *DrsConfig, object *InternalDrsObject) (DrsAccessMethod, bool) {
	if cfg == nil || !cfg.HttpsAccessMethodSupported {
		return DrsAccessMethod{}, false
	}

	baseURL := strings.TrimSpace(cfg.HttpsAccessMethodBaseURL)
	if baseURL == "" || strings.TrimSpace(object.AbsolutePath) == "" {
		return DrsAccessMethod{}, false
	}

	return DrsAccessMethod{
		Type:               "https",
		AccessID:           "irods-go-rest-https",
		Cloud:              buildIRODSCloudName(object),
		Region:             strings.TrimSpace(object.ResourceName),
		Available:          false,
		SupportedAuthTypes: []string{"BasicAuth", "BearerAuth"},
		BearerAuthIssuers:  buildBearerAuthIssuers(cfg),
	}, true
}

func buildLegacyHTTPAccessMethod(cfg *DrsConfig, object *InternalDrsObject) (DrsAccessMethod, bool) {
	if strings.TrimSpace(cfg.HTTPAccessBaseURL) == "" {
		return DrsAccessMethod{}, false
	}

	return DrsAccessMethod{
		Type:      "http",
		AccessID:  "http:" + object.Id,
		Available: false,
	}, true
}

func buildIRODSAccessMethod(cfg *DrsConfig, object *InternalDrsObject) (DrsAccessMethod, bool) {
	if usesStructuredAccessMethodConfig(cfg) && !cfg.IrodsAccessMethodSupported {
		return DrsAccessMethod{}, false
	}

	host := strings.TrimSpace(cfg.IRODSAccessHost)
	if host == "" {
		host = strings.TrimSpace(cfg.IrodsHost)
	}

	port := cfg.IRODSAccessPort
	if port == 0 {
		port = cfg.IrodsPort
	}

	if host == "" || port <= 0 || strings.TrimSpace(object.AbsolutePath) == "" {
		return DrsAccessMethod{}, false
	}

	return DrsAccessMethod{
		Type:      "irods",
		AccessID:  "irods:" + object.Id,
		Available: false,
	}, true
}

func buildFileAccessMethod(cfg *DrsConfig, object *InternalDrsObject) (DrsAccessMethod, bool) {
	if cfg == nil || !cfg.FileAccessMethodSupported {
		return DrsAccessMethod{}, false
	}

	return buildLocalAccessMethod(cfg, object)
}

func buildLocalAccessMethod(cfg *DrsConfig, object *InternalDrsObject) (DrsAccessMethod, bool) {
	root := strings.TrimSpace(cfg.LocalAccessRootPath)
	if root == "" || strings.TrimSpace(object.AbsolutePath) == "" {
		return DrsAccessMethod{}, false
	}

	mappedPath := filepath.Join(root, strings.TrimPrefix(path.Clean(object.AbsolutePath), "/"))
	return DrsAccessMethod{
		Type:      "local",
		URL:       "local://" + filepath.ToSlash(mappedPath),
		Available: true,
	}, true
}

func buildS3AccessMethod(cfg *DrsConfig, object *InternalDrsObject) (DrsAccessMethod, bool) {
	_ = cfg
	if strings.TrimSpace(object.Id) == "" {
		return DrsAccessMethod{}, false
	}

	return DrsAccessMethod{
		Type:      "s3",
		AccessID:  "s3:" + object.Id,
		Available: false,
	}, true
}

func buildIRODSCloudName(object *InternalDrsObject) string {
	if object == nil {
		return ""
	}

	zone := strings.TrimSpace(object.IrodsZone)
	if zone == "" {
		return ""
	}

	return "irods:" + zone
}

func buildBearerAuthIssuers(cfg *DrsConfig) []string {
	if cfg == nil {
		return nil
	}

	issuer := strings.TrimSpace(cfg.OidcUrl)
	if issuer == "" {
		return nil
	}

	return []string{issuer}
}
