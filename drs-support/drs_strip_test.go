package drs_support

import "testing"

func TestStripDrsSemanticsOnSingleDataObject(t *testing.T) {
	rootPath := "/tempZone/home/test1/root"
	filesystem := newCompoundTestFilesystem(rootPath)
	filesystem.addDataObject(rootPath + "/file.txt")

	if err := filesystem.AddMetadata(rootPath+"/file.txt", DrsIdAvuAttrib, "object-1", DrsAvuUnit); err != nil {
		t.Fatalf("seed drs id metadata: %v", err)
	}
	if err := filesystem.AddMetadata(rootPath+"/file.txt", DrsAvuDescriptionAttrib, "desc", DrsAvuUnit); err != nil {
		t.Fatalf("seed drs description metadata: %v", err)
	}
	if err := filesystem.AddMetadata(rootPath+"/file.txt", "user:note", "keep", "custom"); err != nil {
		t.Fatalf("seed non-drs metadata: %v", err)
	}

	result, err := StripDrsSemantics(filesystem, rootPath+"/file.txt")
	if err != nil {
		t.Fatalf("strip drs semantics: %v", err)
	}

	if result.PathsVisited != 1 || result.PathsWithDrsMetadata != 1 || result.AvusRemoved != 2 {
		t.Fatalf("unexpected strip result %+v", result)
	}

	metas, err := filesystem.ListMetadata(rootPath + "/file.txt")
	if err != nil {
		t.Fatalf("list metadata after strip: %v", err)
	}
	if hasMetadataNameWithValue(metas, DrsIdAvuAttrib) || hasMetadataNameWithValue(metas, DrsAvuDescriptionAttrib) {
		t.Fatalf("expected drs metadata removed, got %+v", metas)
	}
	if !hasMetadata(metas, "user:note", "keep", "custom") {
		t.Fatalf("expected non-drs metadata preserved, got %+v", metas)
	}
}

func TestStripDrsSemanticsOnCompoundCollection(t *testing.T) {
	rootPath := "/tempZone/home/test1/compound"
	filesystem := newCompoundTestFilesystem(rootPath)
	filesystem.addCollection(rootPath + "/sub")
	filesystem.addDataObject(rootPath + "/sub/a.txt")
	filesystem.addDataObject(rootPath + "/sub/.drsignore")

	if err := filesystem.AddMetadata(rootPath, DrsIdAvuAttrib, "root-id", DrsAvuUnit); err != nil {
		t.Fatalf("seed root drs id: %v", err)
	}
	if err := filesystem.AddMetadata(rootPath, DrsAvuCompoundManifestAttrib, "true", DrsAvuUnit); err != nil {
		t.Fatalf("seed root compound marker: %v", err)
	}
	if err := filesystem.AddMetadata(rootPath+"/sub", DrsAvuAliasAttrib, "sub", DrsAvuUnit); err != nil {
		t.Fatalf("seed sub alias: %v", err)
	}
	if err := filesystem.AddMetadata(rootPath+"/sub/a.txt", DrsIdAvuAttrib, "child-id", DrsAvuUnit); err != nil {
		t.Fatalf("seed child drs id: %v", err)
	}
	if err := filesystem.AddMetadata(rootPath+"/sub/.drsignore", "user:note", "keep", "custom"); err != nil {
		t.Fatalf("seed drsignore non-drs metadata: %v", err)
	}

	result, err := StripDrsSemantics(filesystem, rootPath)
	if err != nil {
		t.Fatalf("strip compound drs semantics: %v", err)
	}

	if result.PathsVisited < 3 {
		t.Fatalf("expected recursive strip to visit multiple paths, got %+v", result)
	}
	if result.AvusRemoved < 4 {
		t.Fatalf("expected drs avus to be removed recursively, got %+v", result)
	}

	rootMetas, _ := filesystem.ListMetadata(rootPath)
	subMetas, _ := filesystem.ListMetadata(rootPath + "/sub")
	childMetas, _ := filesystem.ListMetadata(rootPath + "/sub/a.txt")
	if hasMetadataNameWithValue(rootMetas, DrsIdAvuAttrib) || hasMetadataNameWithValue(rootMetas, DrsAvuCompoundManifestAttrib) {
		t.Fatalf("expected root drs metadata removed, got %+v", rootMetas)
	}
	if hasMetadataNameWithValue(subMetas, DrsAvuAliasAttrib) {
		t.Fatalf("expected subcollection drs metadata removed, got %+v", subMetas)
	}
	if hasMetadataNameWithValue(childMetas, DrsIdAvuAttrib) {
		t.Fatalf("expected child data object drs metadata removed, got %+v", childMetas)
	}

	ignoreMetas, _ := filesystem.ListMetadata(rootPath + "/sub/.drsignore")
	if !hasMetadata(ignoreMetas, "user:note", "keep", "custom") {
		t.Fatalf("expected .drsignore object to remain with non-drs metadata, got %+v", ignoreMetas)
	}
}
