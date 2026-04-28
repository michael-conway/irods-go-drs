package drs_support

import "testing"

func TestBuildAccessMethodsBuildsConfiguredStubs(t *testing.T) {
	cfg := &DrsConfig{
		IrodsAccessMethodSupported: false,
		FileAccessMethodSupported:  false,
		HttpsAccessMethodSupported: true,
		HttpsAccessImplementation:  "irods-go-rest",
		HttpsAccessMethodBaseURL:   "https://download.example.org/api/v1/path/contents?irods_path=",
		OidcUrl:                    "https://issuer.example.org",
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
	if len(methods) != 1 {
		t.Fatalf("expected 1 access method, got %d", len(methods))
	}

	if methods[0].Type != "https" || methods[0].AccessID != "irods-go-rest-https" || methods[0].URL != "" || methods[0].Available {
		t.Fatalf("unexpected https access method: %+v", methods[0])
	}
	if methods[0].Cloud != "irods:tempZone" {
		t.Fatalf("expected irods cloud name, got %+v", methods[0])
	}
	if methods[0].Region != "demoResc" {
		t.Fatalf("expected resource-backed region, got %+v", methods[0])
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
		HttpsAccessImplementation:  "irods-https-api",
		HttpsAccessMethodBaseURL:   "https://download.example.org/api/v1/path/contents?irods_path=",
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
