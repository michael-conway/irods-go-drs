package drs_support

import (
	"context"
	"fmt"
	"time"
)

type ValidationSeverity string

const (
	ValidationSeverityInfo    ValidationSeverity = "info"
	ValidationSeverityWarning ValidationSeverity = "warning"
	ValidationSeverityError   ValidationSeverity = "error"
)

type ValidationFinding struct {
	Severity ValidationSeverity `json:"severity"`
	DrsID    string             `json:"drsId,omitempty"`
	Path     string             `json:"path,omitempty"`
	Message  string             `json:"message"`
}

type MetadataUpdate struct {
	DrsID    string `json:"drsId,omitempty"`
	Path     string `json:"path,omitempty"`
	Field    string `json:"field"`
	OldValue string `json:"oldValue,omitempty"`
	NewValue string `json:"newValue,omitempty"`
	Status   string `json:"status"`
}

type ValidationReport struct {
	RootDrsID       string              `json:"rootDrsId"`
	StartedAt       time.Time           `json:"startedAt"`
	CompletedAt     time.Time           `json:"completedAt"`
	VisitedDrsIDs   []string            `json:"visitedDrsIds,omitempty"`
	Findings        []ValidationFinding `json:"findings,omitempty"`
	MetadataUpdates []MetadataUpdate    `json:"metadataUpdates,omitempty"`
}

type ObservedObjectState struct {
	Checksum    *InternalChecksum
	Size        int64
	CreatedTime time.Time
	UpdatedTime time.Time
}

type DrsObjectResolver interface {
	GetObjectByID(ctx context.Context, drsID string) (*InternalDrsObject, error)
	ReadObjectContents(ctx context.Context, object *InternalDrsObject) ([]byte, error)
	ObserveObjectState(ctx context.Context, object *InternalDrsObject) (*ObservedObjectState, error)
	UpdateObjectMetadata(ctx context.Context, object *InternalDrsObject, observed *ObservedObjectState) error
}

type DrsValidator struct {
	resolver DrsObjectResolver
}

func NewDrsValidator(resolver DrsObjectResolver) (*DrsValidator, error) {
	if resolver == nil {
		return nil, fmt.Errorf("no DRS object resolver provided")
	}

	return &DrsValidator{resolver: resolver}, nil
}

func (v *DrsValidator) Validate(ctx context.Context, drsID string) *ValidationReport {
	report := &ValidationReport{
		RootDrsID: drsID,
		StartedAt: time.Now().UTC(),
	}

	if v == nil || v.resolver == nil {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: ValidationSeverityError,
			DrsID:    drsID,
			Message:  "validator is not configured",
		})
		report.CompletedAt = time.Now().UTC()
		return report
	}

	visited := map[string]bool{}
	v.validateRecursive(ctx, drsID, report, visited)
	report.CompletedAt = time.Now().UTC()
	return report
}

func (v *DrsValidator) validateRecursive(ctx context.Context, drsID string, report *ValidationReport, visited map[string]bool) {
	if drsID == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: ValidationSeverityError,
			Message:  "encountered empty DRS id during validation",
		})
		return
	}

	if visited[drsID] {
		return
	}

	visited[drsID] = true
	report.VisitedDrsIDs = append(report.VisitedDrsIDs, drsID)

	object, err := v.resolver.GetObjectByID(ctx, drsID)
	if err != nil {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: ValidationSeverityError,
			DrsID:    drsID,
			Message:  fmt.Sprintf("failed to resolve DRS object: %v", err),
		})
		return
	}

	if object == nil {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: ValidationSeverityError,
			DrsID:    drsID,
			Message:  "resolver returned nil object",
		})
		return
	}

	if object.IsManifest {
		v.validateManifestObject(ctx, object, report, visited)
		return
	}

	v.validateAtomicObject(ctx, object, report)
}

func (v *DrsValidator) validateAtomicObject(ctx context.Context, object *InternalDrsObject, report *ValidationReport) {
	observed, err := v.resolver.ObserveObjectState(ctx, object)
	if err != nil {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: ValidationSeverityError,
			DrsID:    object.Id,
			Path:     object.AbsolutePath,
			Message:  fmt.Sprintf("failed to observe object state: %v", err),
		})
		return
	}

	if observed == nil {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: ValidationSeverityWarning,
			DrsID:    object.Id,
			Path:     object.AbsolutePath,
			Message:  "no observed object state was returned",
		})
		return
	}

	needsUpdate := false
	needsUpdate = compareMetadataField(report, object, "size", fmt.Sprintf("%d", object.Size), fmt.Sprintf("%d", observed.Size)) || needsUpdate
	needsUpdate = compareMetadataField(report, object, "created_time", object.CreatedTime.Format(time.RFC3339), observed.CreatedTime.Format(time.RFC3339)) || needsUpdate
	needsUpdate = compareMetadataField(report, object, "updated_time", object.UpdatedTime.Format(time.RFC3339), observed.UpdatedTime.Format(time.RFC3339)) || needsUpdate
	needsUpdate = compareChecksumField(report, object, observed.Checksum) || needsUpdate

	if !needsUpdate {
		return
	}

	if err := v.resolver.UpdateObjectMetadata(ctx, object, observed); err != nil {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: ValidationSeverityError,
			DrsID:    object.Id,
			Path:     object.AbsolutePath,
			Message:  fmt.Sprintf("failed to update metadata: %v", err),
		})
		return
	}

	applyObservedState(object, observed)
	for i := range report.MetadataUpdates {
		if report.MetadataUpdates[i].DrsID == object.Id && report.MetadataUpdates[i].Status == "" {
			report.MetadataUpdates[i].Status = "updated"
		}
	}
}

func (v *DrsValidator) validateManifestObject(ctx context.Context, object *InternalDrsObject, report *ValidationReport, visited map[string]bool) {
	body, err := v.resolver.ReadObjectContents(ctx, object)
	if err != nil {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: ValidationSeverityError,
			DrsID:    object.Id,
			Path:     object.AbsolutePath,
			Message:  fmt.Sprintf("failed to read manifest contents: %v", err),
		})
		return
	}

	manifest, err := ParseDrsManifest(body)
	if err != nil {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: ValidationSeverityError,
			DrsID:    object.Id,
			Path:     object.AbsolutePath,
			Message:  fmt.Sprintf("manifest parse failed: %v", err),
		})
		return
	}

	object.Contents = manifest.Contents

	for _, issue := range manifest.Validate() {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: ValidationSeverityError,
			DrsID:    object.Id,
			Path:     object.AbsolutePath,
			Message:  issue,
		})
	}

	for _, child := range manifest.Contents {
		if child.ID == "" {
			continue
		}

		v.validateRecursive(ctx, child.ID, report, visited)
	}
}

func compareMetadataField(report *ValidationReport, object *InternalDrsObject, field string, current string, observed string) bool {
	if current == observed {
		return false
	}

	report.MetadataUpdates = append(report.MetadataUpdates, MetadataUpdate{
		DrsID:    object.Id,
		Path:     object.AbsolutePath,
		Field:    field,
		OldValue: current,
		NewValue: observed,
	})
	return true
}

func compareChecksumField(report *ValidationReport, object *InternalDrsObject, observed *InternalChecksum) bool {
	currentType := ""
	currentValue := ""
	if object.Checksum != nil {
		currentType = object.Checksum.Type
		currentValue = object.Checksum.Value
	}

	observedType := ""
	observedValue := ""
	if observed != nil {
		observedType = observed.Type
		observedValue = observed.Value
	}

	updated := false
	if currentType != observedType {
		report.MetadataUpdates = append(report.MetadataUpdates, MetadataUpdate{
			DrsID:    object.Id,
			Path:     object.AbsolutePath,
			Field:    "checksum_type",
			OldValue: currentType,
			NewValue: observedType,
		})
		updated = true
	}

	if currentValue != observedValue {
		report.MetadataUpdates = append(report.MetadataUpdates, MetadataUpdate{
			DrsID:    object.Id,
			Path:     object.AbsolutePath,
			Field:    "checksum_value",
			OldValue: currentValue,
			NewValue: observedValue,
		})
		updated = true
	}

	return updated
}

func applyObservedState(object *InternalDrsObject, observed *ObservedObjectState) {
	if object == nil || observed == nil {
		return
	}

	object.Size = observed.Size
	object.CreatedTime = observed.CreatedTime
	object.UpdatedTime = observed.UpdatedTime
	object.Checksum = observed.Checksum
	if object.Version == "" && observed.Checksum != nil {
		object.Version = observed.Checksum.Value
	}
}
