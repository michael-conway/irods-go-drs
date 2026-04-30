package drs_support

import (
	neturl "net/url"
	"path"
	"path/filepath"
	"sort"
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

const (
	HTTPSProviderIRODSGoREST   = "irods-go-rest"
	HTTPSProviderIRODSHTTPSAPI = "irods-https-api"
)

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

	provider := EffectiveHTTPSAccessProvider(cfg.HttpsAccessImplementation)
	switch provider {
	case HTTPSProviderIRODSGoREST, HTTPSProviderIRODSHTTPSAPI:
		return buildProviderHTTPSAccessMethods(cfg, object, provider)
	default:
		return nil
	}
}

func buildProviderHTTPSAccessMethods(cfg *DrsConfig, object *InternalDrsObject, provider string) []DrsAccessMethod {
	resources := objectReplicaResourceNames(object)
	if len(resources) == 0 {
		return nil
	}

	affinities := normalizedResourceAffinityForLookup(cfg.ResourceAffinity)
	cloud := buildIRODSCloudName(object)
	issuers := buildBearerAuthIssuers(cfg)

	// No configured resource affinity: keep one default https entry for the running server.
	if len(affinities) == 0 {
		region := primaryReplicaResourceName(object)
		if region == "" {
			return nil
		}

		return []DrsAccessMethod{{
			Type:               "https",
			AccessID:           buildHTTPSDefaultAccessID(provider),
			Cloud:              cloud,
			Region:             region,
			Available:          false,
			SupportedAuthTypes: []string{"BasicAuth", "BearerAuth"},
			BearerAuthIssuers:  issuers,
		}}
	}

	dedicated := make([]DrsAccessMethod, 0, len(resources))
	fallback := make([]DrsAccessMethod, 0, len(resources))

	for _, resource := range resources {
		host, matched := hostForResourceAffinity(affinities, resource)
		baseURL := ResolveHTTPSAccessBaseURL(cfg.HttpsAccessMethodBaseURL, host)
		if strings.TrimSpace(baseURL) == "" {
			continue
		}

		accessID := buildHTTPSAffinityAccessID(provider, resource)
		method := DrsAccessMethod{
			Type:               "https",
			AccessID:           accessID,
			Cloud:              cloud,
			Region:             resource,
			Available:          false,
			SupportedAuthTypes: []string{"BasicAuth", "BearerAuth"},
			BearerAuthIssuers:  issuers,
		}

		if matched {
			dedicated = append(dedicated, method)
			continue
		}
		fallback = append(fallback, method)
	}

	return append(dedicated, fallback...)
}

func buildIRODSAccessMethods(cfg *DrsConfig, object *InternalDrsObject) []DrsAccessMethod {
	if cfg == nil || object == nil || !cfg.IrodsAccessMethodSupported {
		return nil
	}

	if strings.TrimSpace(object.AbsolutePath) == "" {
		return nil
	}

	return []DrsAccessMethod{{
		Type:      "irods",
		AccessID:  "irods",
		Cloud:     buildIRODSCloudName(object),
		Region:    primaryReplicaResourceName(object),
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

func objectReplicaResourceNames(object *InternalDrsObject) []string {
	if object == nil {
		return nil
	}

	seen := map[string]struct{}{}
	result := make([]string, 0, len(object.Replicas)+1)

	for _, replica := range object.Replicas {
		resourceName := strings.TrimSpace(replica.ResourceName)
		if resourceName == "" {
			continue
		}
		if _, exists := seen[resourceName]; exists {
			continue
		}
		seen[resourceName] = struct{}{}
		result = append(result, resourceName)
	}

	resourceName := strings.TrimSpace(object.ResourceName)
	if resourceName != "" {
		if _, exists := seen[resourceName]; !exists {
			seen[resourceName] = struct{}{}
			result = append(result, resourceName)
		}
	}

	return result
}

func hostForResourceAffinity(affinities []ResourceAffinityEntry, resource string) (string, bool) {
	resource = strings.TrimSpace(resource)
	if resource == "" {
		return "", false
	}

	for _, entry := range affinities {
		if !containsResourceAffinityMatch(entry.Resources, resource) {
			continue
		}
		return entry.Host, true
	}

	for _, entry := range affinities {
		if isDefaultResourceAffinity(entry.Resources) {
			return entry.Host, false
		}
	}

	return "", false
}

func normalizedResourceAffinityForLookup(entries []ResourceAffinityEntry) []ResourceAffinityEntry {
	if len(entries) == 0 {
		return nil
	}

	normalized := make([]ResourceAffinityEntry, 0, len(entries))
	for _, entry := range entries {
		host := strings.TrimSpace(entry.Host)
		resources := normalizeStringSlice(entry.Resources)
		if host == "" {
			continue
		}

		normalized = append(normalized, ResourceAffinityEntry{
			Host:      host,
			Resources: resources,
		})
	}

	return normalized
}

func containsResourceAffinityMatch(resources []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, resource := range resources {
		resource = strings.TrimSpace(resource)
		if resource == "" || resource == "*" {
			continue
		}
		if resource == target {
			return true
		}
	}

	return false
}

func containsResourceAffinityWildcard(resources []string) bool {
	for _, resource := range resources {
		if strings.TrimSpace(resource) == "*" {
			return true
		}
	}
	return false
}

func isDefaultResourceAffinity(resources []string) bool {
	// Preferred default marker: empty resources list.
	if len(resources) == 0 {
		return true
	}

	// Backward-compatible default marker.
	return containsResourceAffinityWildcard(resources)
}

type ParsedHTTPSAccessID struct {
	Provider string
	Resource string
}

func NormalizeHTTPSAccessProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func EffectiveHTTPSAccessProvider(provider string) string {
	provider = NormalizeHTTPSAccessProvider(provider)
	if provider == "" {
		return HTTPSProviderIRODSGoREST
	}
	return provider
}

func IsSupportedHTTPSAccessProvider(provider string) bool {
	switch NormalizeHTTPSAccessProvider(provider) {
	case HTTPSProviderIRODSGoREST, HTTPSProviderIRODSHTTPSAPI:
		return true
	default:
		return false
	}
}

func buildHTTPSDefaultAccessID(provider string) string {
	return EffectiveHTTPSAccessProvider(provider) + "-https"
}

func buildHTTPSAffinityAccessID(provider string, resource string) string {
	return buildHTTPSDefaultAccessID(provider) + "-" + strings.TrimSpace(resource)
}

func ParseHTTPSAccessID(accessID string) (ParsedHTTPSAccessID, bool) {
	id := strings.TrimSpace(accessID)
	if id == "" {
		return ParsedHTTPSAccessID{}, false
	}
	if strings.HasPrefix(id, "irods-go-rest-https-affinity-") {
		return ParsedHTTPSAccessID{}, false
	}

	providers := []string{HTTPSProviderIRODSGoREST, HTTPSProviderIRODSHTTPSAPI}
	for _, provider := range providers {
		defaultID := buildHTTPSDefaultAccessID(provider)
		if id == defaultID {
			return ParsedHTTPSAccessID{
				Provider: provider,
			}, true
		}

		prefix := defaultID + "-"
		if !strings.HasPrefix(id, prefix) {
			continue
		}

		resource := strings.TrimSpace(strings.TrimPrefix(id, prefix))
		if resource == "" {
			return ParsedHTTPSAccessID{}, false
		}

		return ParsedHTTPSAccessID{
			Provider: provider,
			Resource: resource,
		}, true
	}

	return ParsedHTTPSAccessID{}, false
}

func ResolveAffinityHostForResource(cfg *DrsConfig, resource string) string {
	affinities := normalizedResourceAffinityForLookup(cfg.ResourceAffinity)
	host, _ := hostForResourceAffinity(affinities, resource)
	return strings.TrimSpace(host)
}

func ResolveHTTPSAccessBaseURL(configuredBaseURL string, preferredHost string) string {
	configuredBaseURL = strings.TrimSpace(configuredBaseURL)
	preferredHost = strings.TrimSpace(preferredHost)
	if configuredBaseURL == "" {
		return ""
	}
	if preferredHost == "" {
		return configuredBaseURL
	}

	cfgURL, cfgErr := neturl.Parse(configuredBaseURL)
	hostURL, hostErr := neturl.Parse(preferredHost)
	if cfgErr != nil || hostErr != nil || hostURL.Host == "" {
		return configuredBaseURL
	}

	if hostURL.Path == "" && hostURL.RawQuery == "" && hostURL.Fragment == "" {
		merged := *cfgURL
		if hostURL.Scheme != "" {
			merged.Scheme = hostURL.Scheme
		}
		merged.Host = hostURL.Host
		merged.User = hostURL.User
		return merged.String()
	}

	return preferredHost
}

func SortedAccessMethodRegions(methods []DrsAccessMethod) []string {
	regions := make([]string, 0, len(methods))
	for _, method := range methods {
		if strings.TrimSpace(method.Region) == "" {
			continue
		}
		regions = append(regions, method.Region)
	}

	sort.Strings(regions)
	return regions
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
