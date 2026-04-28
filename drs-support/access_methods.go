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

	methods := make([]DrsAccessMethod, 0, 4)
	methods = append(methods, buildHTTPSAccessMethods(cfg, object)...)
	methods = append(methods, buildIRODSAccessMethods(cfg, object)...)
	methods = append(methods, buildFileAccessMethods(cfg, object)...)
	methods = append(methods, buildS3AccessMethods(cfg, object)...)
	return methods
}

func buildHTTPSAccessMethods(cfg *DrsConfig, object *InternalDrsObject) []DrsAccessMethod {
	if cfg == nil || object == nil || !cfg.HttpsAccessMethodSupported {
		return nil
	}

	baseURL := strings.TrimSpace(cfg.HttpsAccessMethodBaseURL)
	if baseURL == "" || strings.TrimSpace(object.AbsolutePath) == "" {
		return nil
	}

	switch implementation := strings.TrimSpace(cfg.HttpsAccessImplementation); implementation {
	case "", "irods-go-rest":
		region := primaryReplicaResourceName(object)
		if region == "" {
			return nil
		}

		return []DrsAccessMethod{{
			Type:               "https",
			AccessID:           "irods-go-rest-https",
			Cloud:              buildIRODSCloudName(object),
			Region:             region,
			Available:          false,
			SupportedAuthTypes: []string{"BasicAuth", "BearerAuth"},
			BearerAuthIssuers:  buildBearerAuthIssuers(cfg),
		}}
	case "irods-https-api":
		// Stub for future provider support.
		return nil
	default:
		return nil
	}
}

func buildIRODSAccessMethods(cfg *DrsConfig, object *InternalDrsObject) []DrsAccessMethod {
	if cfg == nil || object == nil || !cfg.IrodsAccessMethodSupported {
		return nil
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
		return nil
	}

	return []DrsAccessMethod{{
		Type:      "irods",
		AccessID:  "irods:" + object.Id,
		Available: false,
	}}
}

func buildFileAccessMethods(cfg *DrsConfig, object *InternalDrsObject) []DrsAccessMethod {
	if cfg == nil || object == nil || !cfg.FileAccessMethodSupported {
		return nil
	}

	if accessMethod, ok := buildLocalAccessMethod(cfg, object); ok {
		return []DrsAccessMethod{accessMethod}
	}

	return nil
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

func buildS3AccessMethods(cfg *DrsConfig, object *InternalDrsObject) []DrsAccessMethod {
	_ = cfg
	_ = object
	return nil
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

func primaryReplicaResourceName(object *InternalDrsObject) string {
	if object == nil {
		return ""
	}

	for _, replica := range object.Replicas {
		resourceName := strings.TrimSpace(replica.ResourceName)
		if resourceName != "" {
			return resourceName
		}
	}

	return strings.TrimSpace(object.ResourceName)
}
