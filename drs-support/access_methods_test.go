package drs_support

import (
	"testing"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	extension_s3admin "github.com/michael-conway/go-irodsclient-extensions/s3admin"
)

func TestBuildAccessMethodsBuildsConfiguredStubs(t *testing.T) {
	cfg := &DrsConfig{
		IrodsAccessMethodSupported: false,
		FileAccessMethodSupported:  false,
		HttpsAccessMethodSupported: true,
		HttpsAccessImplementation:  "irods-go-rest",
		HttpsAccessMethodBaseURL:   "/api/v1/path/contents?irods_path=",
		HttpsResourceAffinity: []ResourceAffinityEntry{
			{
				Host:      "https://dedicated.example.org",
				Resources: []string{"demoResc"},
			},
			{
				Host:      "https://default.example.org",
				Resources: []string{},
			},
		},
		OidcUrl: "https://issuer.example.org",
	}

	object := &InternalDrsObject{
		Id:           "object-123",
		AbsolutePath: "/tempZone/home/test1/file.txt",
		IrodsZone:    "tempZone",
		ResourceName: "demoResc",
		Replicas: []InternalReplica{
			{ResourceName: "demoResc"},
			{ResourceName: "archiveResc"},
		},
	}

	methods := BuildAccessMethods(cfg, object)
	if len(methods) != 2 {
		t.Fatalf("expected 2 access methods, got %d", len(methods))
	}

	if methods[0].Type != "https" || methods[0].URL != "" || methods[0].Available {
		t.Fatalf("unexpected https access method: %+v", methods[0])
	}
	if methods[0].AccessID == "irods-go-rest-https" {
		t.Fatalf("expected affinity access id for multi-method response, got %+v", methods[0])
	}
	if methods[0].Cloud != "irods:tempZone" {
		t.Fatalf("expected irods cloud name, got %+v", methods[0])
	}
	if methods[0].Region != "demoResc" {
		t.Fatalf("expected resource-backed region, got %+v", methods[0])
	}
	if methods[1].Region != "archiveResc" {
		t.Fatalf("expected fallback method for archiveResc, got %+v", methods[1])
	}
	if len(methods[0].SupportedAuthTypes) != 2 || methods[0].SupportedAuthTypes[0] != "BasicAuth" || methods[0].SupportedAuthTypes[1] != "BearerAuth" {
		t.Fatalf("expected supported auth types, got %+v", methods[0])
	}
	if len(methods[0].BearerAuthIssuers) != 1 || methods[0].BearerAuthIssuers[0] != "https://issuer.example.org" {
		t.Fatalf("expected bearer auth issuers from oidc url, got %+v", methods[0])
	}
}

func TestBuildAccessMethodsBuildsIRODSStub(t *testing.T) {
	cfg := &DrsConfig{
		IrodsAccessMethodSupported: true,
		IRODSAccessHost:            "icat.example.org",
		IRODSAccessPort:            1247,
		OidcUrl:                    "https://issuer.example.org",
	}

	object := &InternalDrsObject{
		Id:           "object-123",
		AbsolutePath: "/tempZone/home/test1/file.txt",
		IrodsZone:    "tempZone",
		ResourceName: "demoResc",
		Replicas: []InternalReplica{
			{ResourceName: "demoResc"},
		},
	}

	methods := BuildAccessMethods(cfg, object)
	if len(methods) != 1 {
		t.Fatalf("expected 1 access method, got %d", len(methods))
	}

	if methods[0].Type != "irods" || methods[0].AccessID != "irods" || methods[0].URL != "" || methods[0].Available {
		t.Fatalf("unexpected irods access method: %+v", methods[0])
	}
	if methods[0].Cloud != "irods:tempZone" {
		t.Fatalf("expected irods cloud name, got %+v", methods[0])
	}
	if methods[0].Region != "demoResc" {
		t.Fatalf("expected resource-backed region, got %+v", methods[0])
	}
	if len(methods[0].SupportedAuthTypes) != 0 {
		t.Fatalf("expected no supported auth types for embedded-ticket irods uri, got %+v", methods[0])
	}
	if len(methods[0].BearerAuthIssuers) != 0 {
		t.Fatalf("expected no bearer auth issuers for embedded-ticket irods uri, got %+v", methods[0])
	}
}

func TestBuildAccessMethodsBuildsIRODSStubWithoutConfiguredHostPort(t *testing.T) {
	cfg := &DrsConfig{
		IrodsAccessMethodSupported: true,
	}

	object := &InternalDrsObject{
		Id:           "object-123",
		AbsolutePath: "/tempZone/home/test1/file.txt",
		IrodsZone:    "tempZone",
		ResourceName: "demoResc",
		Replicas: []InternalReplica{
			{ResourceName: "demoResc"},
		},
	}

	methods := BuildAccessMethods(cfg, object)
	if len(methods) != 1 {
		t.Fatalf("expected 1 access method, got %d", len(methods))
	}

	if methods[0].Type != "irods" || methods[0].AccessID != "irods" || methods[0].URL != "" || methods[0].Available {
		t.Fatalf("unexpected irods access method: %+v", methods[0])
	}
}

func TestBuildAccessMethodsKeepsAffinityAccessIDForSingleAffinityMethod(t *testing.T) {
	cfg := &DrsConfig{
		HttpsAccessMethodSupported: true,
		HttpsAccessImplementation:  "irods-go-rest",
		HttpsAccessMethodBaseURL:   "/api/v1/path/contents?irods_path=",
		HttpsResourceAffinity: []ResourceAffinityEntry{
			{
				Host:      "https://default.example.org",
				Resources: []string{},
			},
		},
	}

	object := &InternalDrsObject{
		Id:           "object-123",
		AbsolutePath: "/tempZone/home/test1/file.txt",
		IrodsZone:    "tempZone",
		Replicas: []InternalReplica{
			{ResourceName: "demoResc"},
		},
	}

	methods := BuildAccessMethods(cfg, object)
	if len(methods) != 1 {
		t.Fatalf("expected 1 access method, got %d", len(methods))
	}
	if methods[0].AccessID == "irods-go-rest-https" {
		t.Fatalf("expected affinity access id for single-method affinity response, got %+v", methods[0])
	}
}

func TestBuildAccessMethodsBuildsProviderDefaultHTTPSAccessIDWithoutAffinity(t *testing.T) {
	cfg := &DrsConfig{
		HttpsAccessMethodSupported: true,
		HttpsAccessImplementation:  "irods-go-rest",
		HttpsAccessMethodBaseURL:   "/api/v1/path/contents?irods_path=",
	}

	object := &InternalDrsObject{
		Id:           "object-123",
		AbsolutePath: "/tempZone/home/test1/file.txt",
		IrodsZone:    "tempZone",
		Replicas: []InternalReplica{
			{ResourceName: "demoResc"},
			{ResourceName: "archiveResc"},
		},
	}

	methods := BuildAccessMethods(cfg, object)
	if len(methods) != 1 {
		t.Fatalf("expected 1 default access method without affinity, got %d", len(methods))
	}
	if methods[0].AccessID != "irods-go-rest-https" {
		t.Fatalf("expected provider-default https access id for default method, got %+v", methods[0])
	}
	if methods[0].Region != "demoResc" {
		t.Fatalf("expected primary replica region on default method, got %+v", methods[0])
	}
}

func TestBuildAccessMethodsSkipsUnconfiguredMethods(t *testing.T) {
	cfg := &DrsConfig{
		HttpsAccessMethodSupported: true,
	}

	object := &InternalDrsObject{
		Id:           "object-123",
		AbsolutePath: "/tempZone/home/test1/file.txt",
	}

	methods := BuildAccessMethods(cfg, object)
	if len(methods) != 0 {
		t.Fatalf("expected no usable access methods, got %+v", methods)
	}
}

func TestBuildAccessMethodsSkipsUnsupportedHTTPSImplementation(t *testing.T) {
	cfg := &DrsConfig{
		HttpsAccessMethodSupported: true,
		HttpsAccessImplementation:  "unsupported-provider",
		HttpsAccessMethodBaseURL:   "/api/v1/path/contents?irods_path=",
	}

	object := &InternalDrsObject{
		Id:           "object-123",
		AbsolutePath: "/tempZone/home/test1/file.txt",
		Replicas: []InternalReplica{
			{ResourceName: "demoResc"},
		},
	}

	methods := BuildAccessMethods(cfg, object)
	if len(methods) != 0 {
		t.Fatalf("expected no methods for unsupported https implementation stub, got %+v", methods)
	}
}

func TestBuildAccessMethodsBuildsIRODSHTTPSAPIProviderPrefix(t *testing.T) {
	cfg := &DrsConfig{
		HttpsAccessMethodSupported: true,
		HttpsAccessImplementation:  "irods-https-api",
		HttpsAccessMethodBaseURL:   "/api/v1/path/contents?irods_path=",
		HttpsResourceAffinity: []ResourceAffinityEntry{
			{
				Host:      "https://dedicated.example.org",
				Resources: []string{"demoResc"},
			},
		},
	}

	object := &InternalDrsObject{
		Id:           "object-123",
		AbsolutePath: "/tempZone/home/test1/file.txt",
		Replicas: []InternalReplica{
			{ResourceName: "demoResc"},
		},
	}

	methods := BuildAccessMethods(cfg, object)
	if len(methods) != 1 {
		t.Fatalf("expected 1 https method, got %+v", methods)
	}

	if methods[0].AccessID != "irods-https-api-https-demoResc" {
		t.Fatalf("expected irods-https-api access-id prefix, got %+v", methods[0])
	}
}

func TestParseHTTPSAccessID(t *testing.T) {
	parsed, ok := ParseHTTPSAccessID("irods-go-rest-https")
	if !ok || parsed.Provider != HTTPSProviderIRODSGoREST || parsed.Resource != "" {
		t.Fatalf("expected parsed default go-rest access-id, got parsed=%+v ok=%v", parsed, ok)
	}

	parsed, ok = ParseHTTPSAccessID("irods-go-rest-https-demoResc")
	if !ok || parsed.Provider != HTTPSProviderIRODSGoREST || parsed.Resource != "demoResc" {
		t.Fatalf("expected parsed go-rest affinity access-id, got parsed=%+v ok=%v", parsed, ok)
	}

	parsed, ok = ParseHTTPSAccessID("irods-https-api-https-demoResc")
	if !ok || parsed.Provider != HTTPSProviderIRODSHTTPSAPI || parsed.Resource != "demoResc" {
		t.Fatalf("expected parsed https-api affinity access-id, got parsed=%+v ok=%v", parsed, ok)
	}
}

func TestParseHTTPSAccessIDRejectsLegacyAndUnknownFormats(t *testing.T) {
	cases := []string{
		"rods-go-rest-https-demoResc",
		"irods-go-rest-https-affinity-aHR0cDovL2xvY2FsaG9zdDo4MDgwfGRlbW9SZXNj",
		"irods-http-api-https-demoResc",
		"unsupported-https-demoResc",
	}

	for _, accessID := range cases {
		if parsed, ok := ParseHTTPSAccessID(accessID); ok {
			t.Fatalf("expected invalid access-id to be rejected: %q parsed=%+v", accessID, parsed)
		}
	}
}

func TestResolveHTTPSAccessBaseURLAppendsPathToAffinityHost(t *testing.T) {
	actual := ResolveHTTPSAccessBaseURL("/api/v1/path/contents?irods_path=", "https://download.example.org")
	expected := "https://download.example.org/api/v1/path/contents?irods_path="
	if actual != expected {
		t.Fatalf("expected appended affinity host base url %q, got %q", expected, actual)
	}
}

func TestResolveHTTPSAccessBaseURLOverridesConfiguredURLHost(t *testing.T) {
	actual := ResolveHTTPSAccessBaseURL("https://rest.example.org/api/v1/path/contents?irods_path=", "https://download.example.org")
	expected := "https://download.example.org/api/v1/path/contents?irods_path="
	if actual != expected {
		t.Fatalf("expected configured base url host override %q, got %q", expected, actual)
	}
}

func TestResolveHTTPSAccessBaseURLUsesConfiguredPathWhenAffinityHostIncludesPath(t *testing.T) {
	actual := ResolveHTTPSAccessBaseURL("/api/v1/path/contents?irods_path=", "https://download.example.org/download?path=")
	expected := "https://download.example.org/api/v1/path/contents?irods_path="
	if actual != expected {
		t.Fatalf("expected configured path override on affinity host %q, got %q", expected, actual)
	}
}

func TestBuildAccessMethodsWithFilesystemBuildsS3AccessMethodFromBucket(t *testing.T) {
	cfg := &DrsConfig{
		S3AccessMethodSupported: true,
		S3AccessMethodBaseURL:   "s3://",
	}

	object := &InternalDrsObject{
		Id:           "object-123",
		AbsolutePath: "/tempZone/home/test1/drscoll/object.txt",
		IrodsZone:    "tempZone",
		ResourceName: "demoResc",
	}

	filesystem := &accessMethodsTestFilesystem{
		account: &irodstypes.IRODSAccount{
			ClientUser: "test1",
		},
		metadataByPath: map[string][]*irodstypes.IRODSMeta{
			"/tempZone/home/test1": {
				{Name: extension_s3admin.AVUBucketAttribute, Value: "test-bucket"},
			},
		},
	}

	methods := BuildAccessMethodsWithFilesystem(cfg, object, filesystem)
	if len(methods) != 1 {
		t.Fatalf("expected one s3 access method, got %+v", methods)
	}

	if methods[0].Type != "s3" || methods[0].URL != "s3://test-bucket/drscoll/object.txt" || !methods[0].Available {
		t.Fatalf("expected s3 access method uri for bucket, got %+v", methods[0])
	}
	if methods[0].AccessID != "test1" {
		t.Fatalf("expected s3 access id to be irods user id %q, got %+v", "test1", methods[0])
	}
}

func TestBuildAccessMethodsWithFilesystemUsesFirstBucketWhenMultipleFound(t *testing.T) {
	cfg := &DrsConfig{
		S3AccessMethodSupported: true,
		S3AccessMethodBaseURL:   "s3://",
	}

	object := &InternalDrsObject{
		Id:           "object-123",
		AbsolutePath: "/tempZone/home/test1/drscoll/object.txt",
		IrodsZone:    "tempZone",
		ResourceName: "demoResc",
	}

	filesystem := &accessMethodsTestFilesystem{
		account: &irodstypes.IRODSAccount{
			ClientUser: "test1",
		},
		metadataByPath: map[string][]*irodstypes.IRODSMeta{
			"/tempZone/home/test1": {
				{Name: extension_s3admin.AVUBucketAttribute, Value: "bucket-b"},
				{Name: extension_s3admin.AVUBucketAttribute, Value: "bucket-a"},
			},
		},
	}

	methods := BuildAccessMethodsWithFilesystem(cfg, object, filesystem)
	if len(methods) != 1 {
		t.Fatalf("expected one s3 access method, got %+v", methods)
	}

	if methods[0].URL != "s3://bucket-a/drscoll/object.txt" {
		t.Fatalf("expected first sorted bucket to be used, got %+v", methods[0])
	}
	if methods[0].AccessID != "test1" {
		t.Fatalf("expected s3 access id to be irods user id %q, got %+v", "test1", methods[0])
	}
}

func TestBuildAccessMethodsWithFilesystemBuildsS3MethodPerReplicaResource(t *testing.T) {
	cfg := &DrsConfig{
		S3AccessMethodSupported: true,
		S3AccessMethodBaseURL:   "s3://",
	}

	object := &InternalDrsObject{
		Id:           "object-123",
		AbsolutePath: "/tempZone/home/test1/drscoll/object.txt",
		IrodsZone:    "tempZone",
		ResourceName: "demoResc",
		Replicas: []InternalReplica{
			{ResourceName: "demoResc"},
			{ResourceName: "archiveResc"},
		},
	}

	filesystem := &accessMethodsTestFilesystem{
		account: &irodstypes.IRODSAccount{
			ClientUser: "test1",
		},
		metadataByPath: map[string][]*irodstypes.IRODSMeta{
			"/tempZone/home/test1": {
				{Name: extension_s3admin.AVUBucketAttribute, Value: "test-bucket"},
			},
		},
	}

	methods := BuildAccessMethodsWithFilesystem(cfg, object, filesystem)
	if len(methods) != 2 {
		t.Fatalf("expected one s3 access method per replica resource, got %+v", methods)
	}
	if methods[0].Type != "s3" || methods[0].URL != "s3://test-bucket/drscoll/object.txt" || methods[0].Region != "demoResc" {
		t.Fatalf("expected first s3 access method for demoResc, got %+v", methods[0])
	}
	if methods[1].Type != "s3" || methods[1].URL != "s3://test-bucket/drscoll/object.txt" || methods[1].Region != "archiveResc" {
		t.Fatalf("expected second s3 access method for archiveResc, got %+v", methods[1])
	}
	if methods[0].AccessID != "test1" || methods[1].AccessID != "test1" {
		t.Fatalf("expected s3 access ids to be irods user id %q, got %+v", "test1", methods)
	}
}

type accessMethodsTestFilesystem struct {
	account        *irodstypes.IRODSAccount
	metadataByPath map[string][]*irodstypes.IRODSMeta
}

func (f *accessMethodsTestFilesystem) StatFile(irodsPath string) (*irodsfs.Entry, error) {
	return nil, nil
}

func (f *accessMethodsTestFilesystem) List(irodsPath string) ([]*irodsfs.Entry, error) {
	return nil, nil
}

func (f *accessMethodsTestFilesystem) SearchByMeta(metaname string, metavalue string) ([]*irodsfs.Entry, error) {
	return nil, nil
}

func (f *accessMethodsTestFilesystem) ListMetadata(irodsPath string) ([]*irodstypes.IRODSMeta, error) {
	if f == nil || f.metadataByPath == nil {
		return nil, nil
	}
	return f.metadataByPath[irodsPath], nil
}

func (f *accessMethodsTestFilesystem) AddMetadata(irodsPath string, attName string, attValue string, attUnits string) error {
	return nil
}

func (f *accessMethodsTestFilesystem) DeleteMetadataByAVU(irodsPath string, attName string, attValue string, attUnits string) error {
	return nil
}

func (f *accessMethodsTestFilesystem) GetAccount() *irodstypes.IRODSAccount {
	return f.account
}
