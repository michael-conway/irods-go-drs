package drs_support

import "testing"

func TestBuildAccessMethodsBuildsConfiguredStubs(t *testing.T) {
	cfg := &DrsConfig{
		AccessMethods:       []string{"http", "irods", "local", "s3"},
		HTTPAccessBaseURL:   "https://drs.example.org",
		IRODSAccessHost:     "irods.example.org",
		IRODSAccessPort:     1247,
		LocalAccessRootPath: "/mnt/irods",
	}

	object := &InternalDrsObject{
		Id:           "object-123",
		AbsolutePath: "/tempZone/home/test1/file.txt",
	}

	methods := BuildAccessMethods(cfg, object)
	if len(methods) != 4 {
		t.Fatalf("expected 4 access methods, got %d", len(methods))
	}

	if methods[0].Type != "http" || methods[0].AccessID != "http:object-123" || methods[0].URL != "" {
		t.Fatalf("unexpected http access method: %+v", methods[0])
	}

	if methods[1].Type != "irods" || methods[1].AccessID != "irods:object-123" || methods[1].URL != "" {
		t.Fatalf("unexpected irods access method: %+v", methods[1])
	}

	if methods[2].Type != "local" || methods[2].URL != "local:///mnt/irods/tempZone/home/test1/file.txt" {
		t.Fatalf("unexpected local access method: %+v", methods[2])
	}

	if methods[3].Type != "s3" || methods[3].AccessID != "s3:object-123" {
		t.Fatalf("unexpected s3 access method: %+v", methods[3])
	}
}

func TestBuildAccessMethodsSkipsUnconfiguredMethods(t *testing.T) {
	cfg := &DrsConfig{
		AccessMethods: []string{"http", "irods", "local"},
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
