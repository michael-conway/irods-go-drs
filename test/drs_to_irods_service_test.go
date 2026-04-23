package test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

func TestApplyDrsMetadata(t *testing.T) {
	metas := []*irodstypes.IRODSMeta{
		{Name: drs_support.DrsIdAvuAttrib, Value: "drs-123", Units: drs_support.DrsAvuUnit},
		{Name: drs_support.DrsAvuVersionAttrib, Value: "v1", Units: drs_support.DrsAvuUnit},
		{Name: drs_support.DrsAvuMimeTypeAttrib, Value: "application/json", Units: drs_support.DrsAvuUnit},
		{Name: drs_support.DrsAvuDescriptionAttrib, Value: "test description", Units: drs_support.DrsAvuUnit},
		{Name: drs_support.DrsAvuAliasAttrib, Value: "alias-1", Units: drs_support.DrsAvuUnit},
		{Name: drs_support.DrsAvuAliasAttrib, Value: "alias-2", Units: drs_support.DrsAvuUnit},
		{Name: drs_support.DrsAvuCompoundManifestAttrib, Value: "true", Units: drs_support.DrsAvuUnit},
	}

	object := &drs_support.InternalDrsObject{}
	if err := drs_support.ApplyDrsMetadata(object, metas); err != nil {
		t.Fatalf("apply metadata: %v", err)
	}

	if object.Id != "drs-123" {
		t.Fatalf("expected id drs-123, got %q", object.Id)
	}

	if !object.IsManifest {
		t.Fatal("expected manifest metadata")
	}

	if len(object.Aliases) != 2 {
		t.Fatalf("expected 2 aliases, got %d", len(object.Aliases))
	}
}

func TestGetDrsObjectByID(t *testing.T) {
	createTime := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	updateTime := createTime.Add(5 * time.Minute)

	filesystem := &fakeIRODSFilesystem{
		account: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
		searchEntries: []*irodsfs.Entry{
			{
				ID:         1,
				Type:       irodsfs.FileEntry,
				Name:       "file.txt",
				Path:       "/tempZone/home/rods/file.txt",
				Size:       42,
				CreateTime: createTime,
				ModifyTime: updateTime,
			},
		},
		metadataByPath: map[string][]*irodstypes.IRODSMeta{
			"/tempZone/home/rods/file.txt": {
				{Name: drs_support.DrsIdAvuAttrib, Value: "drs-123", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuMimeTypeAttrib, Value: "text/plain", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuDescriptionAttrib, Value: "test description", Units: drs_support.DrsAvuUnit},
			},
		},
	}

	object, err := drs_support.GetDrsObjectByID(filesystem, "drs-123")
	if err != nil {
		t.Fatalf("get drs object by id: %v", err)
	}

	if object.Id != "drs-123" {
		t.Fatalf("expected DRS id drs-123, got %q", object.Id)
	}

	if object.AbsolutePath != "/tempZone/home/rods/file.txt" {
		t.Fatalf("expected object path to be preserved, got %q", object.AbsolutePath)
	}

	if object.MimeType != "text/plain" {
		t.Fatalf("expected mime type text/plain, got %q", object.MimeType)
	}
}

func TestGetDrsObjectByIDReturnsNotFound(t *testing.T) {
	filesystem := &fakeIRODSFilesystem{
		account:       &irodstypes.IRODSAccount{ClientZone: "tempZone"},
		searchEntries: []*irodsfs.Entry{},
	}

	_, err := drs_support.GetDrsObjectByID(filesystem, "missing-drs-id")
	if err == nil {
		t.Fatal("expected missing DRS object error")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestGetDrsObjectByIDRejectsAmbiguousMatches(t *testing.T) {
	filesystem := &fakeIRODSFilesystem{
		account: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
		searchEntries: []*irodsfs.Entry{
			{ID: 1, Type: irodsfs.FileEntry, Path: "/tempZone/home/rods/file1.txt", Name: "file1.txt"},
			{ID: 2, Type: irodsfs.FileEntry, Path: "/tempZone/home/rods/file2.txt", Name: "file2.txt"},
		},
		metadataByPath: map[string][]*irodstypes.IRODSMeta{
			"/tempZone/home/rods/file1.txt": {
				{Name: drs_support.DrsIdAvuAttrib, Value: "drs-123", Units: drs_support.DrsAvuUnit},
			},
			"/tempZone/home/rods/file2.txt": {
				{Name: drs_support.DrsIdAvuAttrib, Value: "drs-123", Units: drs_support.DrsAvuUnit},
			},
		},
	}

	_, err := drs_support.GetDrsObjectByID(filesystem, "drs-123")
	if err == nil {
		t.Fatal("expected ambiguous DRS id error")
	}

	if !strings.Contains(err.Error(), "matched multiple data objects") {
		t.Fatalf("expected ambiguous match error, got %v", err)
	}
}

func TestGetDrsObjectByIRODSPath(t *testing.T) {
	createTime := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	updateTime := createTime.Add(5 * time.Minute)
	path := "/tempZone/home/rods/file.txt"

	filesystem := &fakeIRODSFilesystem{
		account: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
		entryByPath: map[string]*irodsfs.Entry{
			path: {
				ID:         1,
				Type:       irodsfs.FileEntry,
				Name:       "file.txt",
				Path:       path,
				Size:       42,
				CreateTime: createTime,
				ModifyTime: updateTime,
			},
		},
		metadataByPath: map[string][]*irodstypes.IRODSMeta{
			path: {
				{Name: drs_support.DrsIdAvuAttrib, Value: "drs-123", Units: drs_support.DrsAvuUnit},
				{Name: drs_support.DrsAvuMimeTypeAttrib, Value: "text/plain", Units: drs_support.DrsAvuUnit},
			},
		},
	}

	object, err := drs_support.GetDrsObjectByIRODSPath(filesystem, path)
	if err != nil {
		t.Fatalf("get drs object by path: %v", err)
	}

	if object.Id != "drs-123" {
		t.Fatalf("expected DRS id drs-123, got %q", object.Id)
	}
}

func TestGetDrsObjectByIRODSPathRejectsNonDrsObject(t *testing.T) {
	path := "/tempZone/home/rods/file.txt"
	filesystem := &fakeIRODSFilesystem{
		account: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
		entryByPath: map[string]*irodsfs.Entry{
			path: {ID: 1, Type: irodsfs.FileEntry, Name: "file.txt", Path: path},
		},
		metadataByPath: map[string][]*irodstypes.IRODSMeta{
			path: {
				{Name: "user:note", Value: "keep-me", Units: "custom"},
			},
		},
	}

	_, err := drs_support.GetDrsObjectByIRODSPath(filesystem, path)
	if err == nil {
		t.Fatal("expected non-DRS object error")
	}

	if !strings.Contains(err.Error(), "is not a DRS object") {
		t.Fatalf("expected non-DRS object error, got %v", err)
	}
}

func TestListDrsObjectsUnderCollectionHonorsRecursiveFlag(t *testing.T) {
	root := "/tempZone/home/rods"
	sub := root + "/nested"
	rootDrsPath := root + "/file1.txt"
	subDrsPath := sub + "/file2.txt"

	filesystem := &fakeIRODSFilesystem{
		account: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
		listByPath: map[string][]*irodsfs.Entry{
			root: {
				{ID: 1, Type: irodsfs.FileEntry, Name: "file1.txt", Path: rootDrsPath},
				{ID: 2, Type: irodsfs.DirectoryEntry, Name: "nested", Path: sub},
				{ID: 3, Type: irodsfs.FileEntry, Name: "plain.txt", Path: root + "/plain.txt"},
			},
			sub: {
				{ID: 4, Type: irodsfs.FileEntry, Name: "file2.txt", Path: subDrsPath},
			},
		},
		metadataByPath: map[string][]*irodstypes.IRODSMeta{
			rootDrsPath: {
				{Name: drs_support.DrsIdAvuAttrib, Value: "drs-root", Units: drs_support.DrsAvuUnit},
			},
			subDrsPath: {
				{Name: drs_support.DrsIdAvuAttrib, Value: "drs-nested", Units: drs_support.DrsAvuUnit},
			},
			root + "/plain.txt": {
				{Name: "user:note", Value: "keep-me", Units: "custom"},
			},
		},
	}

	nonRecursive, err := drs_support.ListDrsObjectsUnderCollection(filesystem, root, false)
	if err != nil {
		t.Fatalf("list non-recursive DRS objects: %v", err)
	}

	if len(nonRecursive) != 1 || nonRecursive[0].Id != "drs-root" {
		t.Fatalf("expected only root DRS object, got %+v", nonRecursive)
	}

	recursive, err := drs_support.ListDrsObjectsUnderCollection(filesystem, root, true)
	if err != nil {
		t.Fatalf("list recursive DRS objects: %v", err)
	}

	if len(recursive) != 2 {
		t.Fatalf("expected 2 recursive DRS objects, got %d", len(recursive))
	}
}

func TestListDrsObjectsReturnsPagedResults(t *testing.T) {
	root := "/tempZone"
	filesystem := &fakeIRODSFilesystem{
		account: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
		listByPath: map[string][]*irodsfs.Entry{
			root: {
				{ID: 1, Type: irodsfs.FileEntry, Name: "b.txt", Path: root + "/b.txt"},
				{ID: 2, Type: irodsfs.FileEntry, Name: "a.txt", Path: root + "/a.txt"},
				{ID: 3, Type: irodsfs.FileEntry, Name: "c.txt", Path: root + "/c.txt"},
			},
		},
		metadataByPath: map[string][]*irodstypes.IRODSMeta{
			root + "/a.txt": {{Name: drs_support.DrsIdAvuAttrib, Value: "drs-a", Units: drs_support.DrsAvuUnit}},
			root + "/b.txt": {{Name: drs_support.DrsIdAvuAttrib, Value: "drs-b", Units: drs_support.DrsAvuUnit}},
			root + "/c.txt": {{Name: drs_support.DrsIdAvuAttrib, Value: "drs-c", Units: drs_support.DrsAvuUnit}},
		},
	}

	page, err := drs_support.ListDrsObjects(filesystem, 1, 1)
	if err != nil {
		t.Fatalf("list paged DRS objects: %v", err)
	}

	if page.Total != 3 {
		t.Fatalf("expected total 3, got %d", page.Total)
	}

	if !page.HasMore {
		t.Fatal("expected HasMore to be true")
	}

	if len(page.Objects) != 1 || page.Objects[0].Id != "drs-b" {
		t.Fatalf("expected second sorted object drs-b, got %+v", page.Objects)
	}
}

func TestCreateDrsObjectFromDataObject(t *testing.T) {
	createTime := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	updateTime := createTime.Add(5 * time.Minute)
	checksum, err := irodstypes.CreateIRODSChecksum("d41d8cd98f00b204e9800998ecf8427e")
	if err != nil {
		t.Fatalf("create checksum: %v", err)
	}

	filesystem := &fakeIRODSFilesystem{
		account: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
		entry: &irodsfs.Entry{
			ID:         1,
			Type:       irodsfs.FileEntry,
			Name:       "file.txt",
			Path:       "/tempZone/home/rods/file.txt",
			Size:       42,
			CreateTime: createTime,
			ModifyTime: updateTime,
			IRODSReplicas: []irodstypes.IRODSReplica{
				{
					Checksum:   checksum,
					CreateTime: createTime,
					ModifyTime: updateTime,
				},
			},
		},
		metadata: []*irodstypes.IRODSMeta{},
	}

	drsID, err := drs_support.CreateDrsObjectFromDataObject(
		filesystem,
		"/tempZone/home/rods/file.txt",
		"text/plain",
		"file description",
		[]string{"alias-1", "alias-2"},
	)
	if err != nil {
		t.Fatalf("create drs object: %v", err)
	}

	if drsID == "" {
		t.Fatal("expected generated drs id")
	}

	if len(filesystem.addedMetadata) != 6 {
		t.Fatalf("expected 6 AVUs to be written, got %d", len(filesystem.addedMetadata))
	}

	if got := filesystem.addedMetadata[0]; got.Name != drs_support.DrsIdAvuAttrib || got.Value != drsID {
		t.Fatalf("expected first metadata to be DRS id %q, got %+v", drsID, got)
	}

	if got := filesystem.addedMetadata[1]; got.Name != drs_support.DrsAvuVersionAttrib || got.Value != "d41d8cd98f00b204e9800998ecf8427e" {
		t.Fatalf("expected version metadata from checksum, got %+v", got)
	}
}

func TestCreateDrsObjectFromDataObjectDerivesMimeTypeFromPath(t *testing.T) {
	createTime := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	updateTime := createTime.Add(5 * time.Minute)

	filesystem := &fakeIRODSFilesystem{
		account: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
		entry: &irodsfs.Entry{
			ID:         1,
			Type:       irodsfs.FileEntry,
			Name:       "file.txt",
			Path:       "/tempZone/home/rods/file.txt",
			Size:       42,
			CreateTime: createTime,
			ModifyTime: updateTime,
		},
		metadata: []*irodstypes.IRODSMeta{},
	}

	_, err := drs_support.CreateDrsObjectFromDataObject(
		filesystem,
		"/tempZone/home/rods/file.txt",
		"",
		"file description",
		nil,
	)
	if err != nil {
		t.Fatalf("create drs object: %v", err)
	}

	foundMimeType := false
	for _, meta := range filesystem.addedMetadata {
		if meta.Name == drs_support.DrsAvuMimeTypeAttrib {
			foundMimeType = true
			if meta.Value != "text/plain" {
				t.Fatalf("expected derived mime type text/plain, got %q", meta.Value)
			}
		}
	}

	if !foundMimeType {
		t.Fatal("expected derived mime type metadata to be written")
	}
}

func TestCreateDrsObjectFromDataObjectRejectsExistingDrsObject(t *testing.T) {
	filesystem := &fakeIRODSFilesystem{
		account: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
		entry: &irodsfs.Entry{
			ID:   1,
			Type: irodsfs.FileEntry,
			Path: "/tempZone/home/rods/file.txt",
			Name: "file.txt",
		},
		metadata: []*irodstypes.IRODSMeta{
			{Name: drs_support.DrsIdAvuAttrib, Value: "existing-drs-id", Units: drs_support.DrsAvuUnit},
		},
	}

	_, err := drs_support.CreateDrsObjectFromDataObject(filesystem, "/tempZone/home/rods/file.txt", "text/plain", "", nil)
	if err == nil {
		t.Fatal("expected an error for existing DRS object")
	}

	if !strings.Contains(err.Error(), "already a DRS object") {
		t.Fatalf("expected already a DRS object error, got %v", err)
	}
}

func TestRemoveSingleDrsObjectFromDataObject(t *testing.T) {
	filesystem := &fakeIRODSFilesystem{
		account: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
		entry: &irodsfs.Entry{
			ID:   1,
			Type: irodsfs.FileEntry,
			Path: "/tempZone/home/rods/file.txt",
			Name: "file.txt",
		},
		metadata: []*irodstypes.IRODSMeta{
			{Name: drs_support.DrsIdAvuAttrib, Value: "drs-123", Units: drs_support.DrsAvuUnit},
			{Name: drs_support.DrsAvuVersionAttrib, Value: "v1", Units: drs_support.DrsAvuUnit},
			{Name: drs_support.DrsAvuMimeTypeAttrib, Value: "text/plain", Units: drs_support.DrsAvuUnit},
			{Name: drs_support.DrsAvuDescriptionAttrib, Value: "file description", Units: drs_support.DrsAvuUnit},
			{Name: drs_support.DrsAvuAliasAttrib, Value: "alias-1", Units: drs_support.DrsAvuUnit},
			{Name: "user:note", Value: "keep-me", Units: "custom"},
		},
	}

	err := drs_support.RemoveSingleDrsObjectFromDataObject(filesystem, "/tempZone/home/rods/file.txt")
	if err != nil {
		t.Fatalf("remove drs object: %v", err)
	}

	if len(filesystem.deletedMetadata) != 5 {
		t.Fatalf("expected 5 DRS AVUs to be removed, got %d", len(filesystem.deletedMetadata))
	}

	for _, meta := range filesystem.deletedMetadata {
		if meta.Name == "user:note" {
			t.Fatalf("expected non-DRS metadata to be preserved, got removal %+v", meta)
		}
	}
}

func TestRemoveSingleDrsObjectFromDataObjectIsIdempotentForNonDrsObject(t *testing.T) {
	filesystem := &fakeIRODSFilesystem{
		account: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
		entry: &irodsfs.Entry{
			ID:   1,
			Type: irodsfs.FileEntry,
			Path: "/tempZone/home/rods/file.txt",
			Name: "file.txt",
		},
		metadata: []*irodstypes.IRODSMeta{
			{Name: "user:note", Value: "keep-me", Units: "custom"},
		},
	}

	err := drs_support.RemoveSingleDrsObjectFromDataObject(filesystem, "/tempZone/home/rods/file.txt")
	if err != nil {
		t.Fatalf("remove drs object idempotently: %v", err)
	}

	if len(filesystem.deletedMetadata) != 0 {
		t.Fatalf("expected no metadata removals for non-DRS object, got %d", len(filesystem.deletedMetadata))
	}
}

func TestRemoveSingleDrsObjectFromDataObjectRejectsCompoundManifest(t *testing.T) {
	filesystem := &fakeIRODSFilesystem{
		account: &irodstypes.IRODSAccount{ClientZone: "tempZone"},
		entry: &irodsfs.Entry{
			ID:   1,
			Type: irodsfs.FileEntry,
			Path: "/tempZone/home/rods/manifest.json",
			Name: "manifest.json",
		},
		metadata: []*irodstypes.IRODSMeta{
			{Name: drs_support.DrsIdAvuAttrib, Value: "drs-manifest-123", Units: drs_support.DrsAvuUnit},
			{Name: drs_support.DrsAvuCompoundManifestAttrib, Value: "true", Units: drs_support.DrsAvuUnit},
			{Name: drs_support.DrsAvuDescriptionAttrib, Value: "compound manifest", Units: drs_support.DrsAvuUnit},
		},
	}

	err := drs_support.RemoveSingleDrsObjectFromDataObject(filesystem, "/tempZone/home/rods/manifest.json")
	if err == nil {
		t.Fatal("expected compound manifest removal to fail")
	}

	if !strings.Contains(err.Error(), "compound DRS manifest") {
		t.Fatalf("expected compound manifest error, got %v", err)
	}

	if len(filesystem.deletedMetadata) != 0 {
		t.Fatalf("expected no metadata removals for compound manifest, got %d", len(filesystem.deletedMetadata))
	}
}

func TestCreateCompoundDrsObjectFromDataObjectSkeleton(t *testing.T) {
	filesystem := &fakeIRODSFilesystem{}

	_, err := drs_support.CreateCompoundDrsObjectFromDataObject(filesystem, "/tempZone/home/rods/manifest.json", "compound", []string{"alias-1"})
	if err == nil {
		t.Fatal("expected skeleton method to return an error")
	}

	if !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected not implemented error, got %v", err)
	}
}

func TestParseDrsManifest(t *testing.T) {
	manifest, err := drs_support.ParseDrsManifest([]byte(`{
		"schema":"irods-drs-manifest/v1",
		"type":"compound",
		"contents":[
			{"id":"drs://example.org/object-1","name":"bam"},
			{"id":"drs://example.org/object-2","name":"bai"}
		]
	}`))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	if len(manifest.Contents) != 2 {
		t.Fatalf("expected 2 manifest entries, got %d", len(manifest.Contents))
	}

	if issues := manifest.Validate(); len(issues) != 0 {
		t.Fatalf("expected no manifest validation issues, got %v", issues)
	}
}

type fakeValidatorResolver struct {
	objects         map[string]*drs_support.InternalDrsObject
	contents        map[string][]byte
	observed        map[string]*drs_support.ObservedObjectState
	updatedMetadata []string
}

func (r *fakeValidatorResolver) GetObjectByID(_ context.Context, drsID string) (*drs_support.InternalDrsObject, error) {
	object, ok := r.objects[drsID]
	if !ok {
		return nil, context.DeadlineExceeded
	}
	return object, nil
}

func (r *fakeValidatorResolver) ReadObjectContents(_ context.Context, object *drs_support.InternalDrsObject) ([]byte, error) {
	return r.contents[object.Id], nil
}

func (r *fakeValidatorResolver) ObserveObjectState(_ context.Context, object *drs_support.InternalDrsObject) (*drs_support.ObservedObjectState, error) {
	return r.observed[object.Id], nil
}

func (r *fakeValidatorResolver) UpdateObjectMetadata(_ context.Context, object *drs_support.InternalDrsObject, observed *drs_support.ObservedObjectState) error {
	r.updatedMetadata = append(r.updatedMetadata, object.Id)
	return nil
}

type fakeIRODSFilesystem struct {
	account       *irodstypes.IRODSAccount
	entry         *irodsfs.Entry
	entryByPath   map[string]*irodsfs.Entry
	searchEntries []*irodsfs.Entry
	metadata      []*irodstypes.IRODSMeta
	metadataByPath map[string][]*irodstypes.IRODSMeta
	listByPath    map[string][]*irodsfs.Entry
	addedMetadata []*irodstypes.IRODSMeta
	deletedMetadata []*irodstypes.IRODSMeta
	statErr       error
	listDirErr    error
	searchErr     error
	listErr       error
	addErr        error
	deleteErr     error
}

func (f *fakeIRODSFilesystem) StatFile(irodsPath string) (*irodsfs.Entry, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	if f.entryByPath != nil {
		if entry, ok := f.entryByPath[irodsPath]; ok {
			return entry, nil
		}
	}
	if f.entry == nil {
		return nil, errors.New("missing fake entry")
	}
	return f.entry, nil
}

func (f *fakeIRODSFilesystem) List(irodsPath string) ([]*irodsfs.Entry, error) {
	if f.listDirErr != nil {
		return nil, f.listDirErr
	}

	if f.listByPath != nil {
		if entries, ok := f.listByPath[irodsPath]; ok {
			return entries, nil
		}
	}

	return []*irodsfs.Entry{}, nil
}

func (f *fakeIRODSFilesystem) SearchByMeta(_ string, _ string) ([]*irodsfs.Entry, error) {
	if f.searchErr != nil {
		return nil, f.searchErr
	}

	return f.searchEntries, nil
}

func (f *fakeIRODSFilesystem) ListMetadata(irodsPath string) ([]*irodstypes.IRODSMeta, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}

	if f.metadataByPath != nil {
		if metas, ok := f.metadataByPath[irodsPath]; ok {
			return metas, nil
		}
	}

	return f.metadata, nil
}

func (f *fakeIRODSFilesystem) AddMetadata(_ string, attName string, attValue string, attUnits string) error {
	if f.addErr != nil {
		return f.addErr
	}

	f.addedMetadata = append(f.addedMetadata, &irodstypes.IRODSMeta{
		Name:  attName,
		Value: attValue,
		Units: attUnits,
	})
	return nil
}

func (f *fakeIRODSFilesystem) DeleteMetadataByAVU(_ string, attName string, attValue string, attUnits string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}

	f.deletedMetadata = append(f.deletedMetadata, &irodstypes.IRODSMeta{
		Name:  attName,
		Value: attValue,
		Units: attUnits,
	})
	return nil
}

func (f *fakeIRODSFilesystem) GetAccount() *irodstypes.IRODSAccount {
	return f.account
}

func TestDrsValidatorReportsBrokenManifestWithoutThrowing(t *testing.T) {
	validator, err := drs_support.NewDrsValidator(&fakeValidatorResolver{
		objects: map[string]*drs_support.InternalDrsObject{
			"manifest-1": {
				Id:           "manifest-1",
				AbsolutePath: "/tempZone/home/rods/manifest-1.json",
				IsManifest:   true,
			},
		},
		contents: map[string][]byte{
			"manifest-1": []byte(`{"schema":"irods-drs-manifest/v1","contents":[{"name":"missing-id"}]}`),
		},
	})
	if err != nil {
		t.Fatalf("create validator: %v", err)
	}

	report := validator.Validate(context.Background(), "manifest-1")
	if report == nil {
		t.Fatal("expected validation report")
	}

	if len(report.Findings) == 0 {
		t.Fatal("expected manifest validation findings")
	}
}

func TestDrsValidatorUpdatesAtomicMetadata(t *testing.T) {
	validator, err := drs_support.NewDrsValidator(&fakeValidatorResolver{
		objects: map[string]*drs_support.InternalDrsObject{
			"object-1": {
				Id:           "object-1",
				AbsolutePath: "/tempZone/home/rods/object-1.txt",
				Size:         10,
			},
		},
		observed: map[string]*drs_support.ObservedObjectState{
			"object-1": {
				Size:        20,
				CreatedTime: time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC),
				UpdatedTime: time.Date(2026, 4, 23, 11, 0, 0, 0, time.UTC),
				Checksum: &drs_support.InternalChecksum{
					Type:  "MD5",
					Value: "d41d8cd98f00b204e9800998ecf8427e",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("create validator: %v", err)
	}

	report := validator.Validate(context.Background(), "object-1")
	if len(report.MetadataUpdates) == 0 {
		t.Fatal("expected metadata updates in report")
	}
}
