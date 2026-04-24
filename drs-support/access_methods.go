package drs_support

import (
	"path"
	"path/filepath"
	"strings"
)

type DrsAccessMethod struct {
	Type      string
	URL       string
	AccessID  string
	Cloud     string
	Region    string
	Available bool
}

func BuildAccessMethods(cfg *DrsConfig, object *InternalDrsObject) []DrsAccessMethod {
	if cfg == nil || object == nil {
		return nil
	}

	methods := make([]DrsAccessMethod, 0, len(cfg.AccessMethods))
	for _, method := range cfg.AccessMethods {
		switch strings.ToLower(strings.TrimSpace(method)) {
		case "http":
			if accessMethod, ok := buildHTTPAccessMethod(cfg, object); ok {
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

func buildHTTPAccessMethod(cfg *DrsConfig, object *InternalDrsObject) (DrsAccessMethod, bool) {
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
