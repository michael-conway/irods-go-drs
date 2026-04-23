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
	metadata      []*irodstypes.IRODSMeta
	addedMetadata []*irodstypes.IRODSMeta
	statErr       error
	listErr       error
	addErr        error
}

func (f *fakeIRODSFilesystem) StatFile(_ string) (*irodsfs.Entry, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	if f.entry == nil {
		return nil, errors.New("missing fake entry")
	}
	return f.entry, nil
}

func (f *fakeIRODSFilesystem) ListMetadata(_ string) ([]*irodstypes.IRODSMeta, error) {
	if f.listErr != nil {
		return nil, f.listErr
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
