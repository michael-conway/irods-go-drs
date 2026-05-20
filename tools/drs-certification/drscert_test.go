package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildComplianceConfigDefaultsToBasicOnly(t *testing.T) {
	objects := testCorpusObjects()

	config, err := buildComplianceConfig(objects, "test1", "test-password", "", "run-1")
	if err != nil {
		t.Fatalf("build compliance config: %v", err)
	}

	expectedBasicToken := base64.StdEncoding.EncodeToString([]byte("test1:test-password"))
	if config.ServiceInfo.AuthType != "basic" || config.ServiceInfo.AuthToken != expectedBasicToken {
		t.Fatalf("expected Basic-auth service-info config, got %+v", config.ServiceInfo)
	}
	if len(config.DRSObjectInfo) != len(objects) {
		t.Fatalf("expected %d object info entries, got %+v", len(objects), config.DRSObjectInfo)
	}
	for _, entry := range config.DRSObjectInfo {
		if entry.AuthType != "basic" || entry.AuthToken != expectedBasicToken {
			t.Fatalf("expected only Basic-auth object info entries without bearer token, got %+v", config.DRSObjectInfo)
		}
	}
	if !config.DRSObjectInfo[2].IsCompound {
		t.Fatalf("expected compound object to be marked compound, got %+v", config.DRSObjectInfo[2])
	}
	if len(config.DRSObjectAccess) != 1 || config.DRSObjectAccess[0].DRSID != "object-1" {
		t.Fatalf("expected only primary object access entry, got %+v", config.DRSObjectAccess)
	}

	if len(config.NegativeTests.InvalidAuth) != 1 {
		t.Fatalf("expected only basic invalid auth entry, got %+v", config.NegativeTests.InvalidAuth)
	}

	basic := config.NegativeTests.InvalidAuth[0]
	if basic.AuthType != "basic" || basic.DRSID != "object-1" {
		t.Fatalf("expected first invalid auth entry to be basic for object-1, got %+v", basic)
	}
	expectedInvalidBasic := base64.StdEncoding.EncodeToString([]byte("test1:__drs_certification_invalid_password__"))
	if basic.AuthToken != expectedInvalidBasic {
		t.Fatalf("expected invalid basic token %q, got %q", expectedInvalidBasic, basic.AuthToken)
	}
	assertExpectedStatuses(t, basic.ExpectedStatuses)
}

func TestBuildComplianceConfigIncludesBearerWhenTokenProvided(t *testing.T) {
	objects := testCorpusObjects()

	config, err := buildComplianceConfig(objects, "test1", "test-password", "bearer-token-1", "run-1")
	if err != nil {
		t.Fatalf("build compliance config: %v", err)
	}

	if len(config.DRSObjectInfo) != len(objects)*2 {
		t.Fatalf("expected basic and bearer object info entries, got %+v", config.DRSObjectInfo)
	}
	if config.DRSObjectInfo[0].AuthType != "basic" || config.DRSObjectInfo[1].AuthType != "bearer" {
		t.Fatalf("expected paired basic and bearer object info entries, got %+v", config.DRSObjectInfo[:2])
	}
	if config.DRSObjectInfo[1].AuthToken != "bearer-token-1" {
		t.Fatalf("expected bearer token in object info, got %+v", config.DRSObjectInfo[1])
	}
	if !config.DRSObjectInfo[5].IsCompound {
		t.Fatalf("expected bearer compound object to be marked compound, got %+v", config.DRSObjectInfo[5])
	}

	if len(config.DRSObjectAccess) != 2 {
		t.Fatalf("expected basic and bearer access entries, got %+v", config.DRSObjectAccess)
	}
	if config.DRSObjectAccess[1].AuthType != "bearer" || config.DRSObjectAccess[1].AuthToken != "bearer-token-1" {
		t.Fatalf("expected bearer access entry, got %+v", config.DRSObjectAccess)
	}

	if len(config.NegativeTests.InvalidAuth) != 2 {
		t.Fatalf("expected basic and bearer invalid auth entries, got %+v", config.NegativeTests.InvalidAuth)
	}

	bearer := config.NegativeTests.InvalidAuth[1]
	if bearer.AuthType != "bearer" || bearer.DRSID != "object-1" {
		t.Fatalf("expected second invalid auth entry to be bearer for object-1, got %+v", bearer)
	}
	if bearer.AuthToken != "__drs_certification_invalid_bearer__" {
		t.Fatalf("unexpected bearer token: %+v", bearer)
	}
	assertExpectedStatuses(t, bearer.ExpectedStatuses)
}

func TestBuildComplianceConfigRequiresPrimaryObject(t *testing.T) {
	if _, err := buildComplianceConfig(nil, "test1", "test-password", "", "run-1"); err == nil {
		t.Fatal("expected missing primary corpus object to fail")
	}
}

func TestReadBearerTokenFileTrimsTokenAndPrefix(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token.txt")
	if err := os.WriteFile(tokenFile, []byte("Bearer bearer-token-1\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	token, err := readBearerTokenFile(tokenFile)
	if err != nil {
		t.Fatalf("read token file: %v", err)
	}
	if token != "bearer-token-1" {
		t.Fatalf("expected stripped bearer token, got %q", token)
	}
}

func TestPrepareOptionsDoNotAcceptRunPhaseOptions(t *testing.T) {
	if _, err := parsePrepareOptions([]string{"--server-base-url", "http://localhost:8888/ga4gh/drs/v1"}); err == nil {
		t.Fatal("expected prepare to reject --server-base-url")
	}
	if _, err := parsePrepareOptions([]string{"--report-path", "CERTIFICATION.md"}); err == nil {
		t.Fatal("expected prepare to reject --report-path")
	}
}

func TestJSONArtifactsDoNotSerializeRunPhaseOptions(t *testing.T) {
	corpusData, err := json.Marshal(Corpus{
		SchemaVersion:        corpusSchemaVersion,
		RunID:                "run-1",
		RootPath:             "/tempZone/home/test1/drs-certification/run-1",
		EffectiveUser:        "test1",
		ComplianceConfigPath: ".certification/drs/drs-compliance-config.json",
	})
	if err != nil {
		t.Fatalf("marshal corpus: %v", err)
	}
	assertNoRunPhaseOptions(t, corpusData)

	recordData, err := json.Marshal(runRecord{
		StartedAt:   testTime(),
		CompletedAt: testTime(),
		ExitCode:    0,
	})
	if err != nil {
		t.Fatalf("marshal run record: %v", err)
	}
	assertNoRunPhaseOptions(t, recordData)
}

func TestSanitizeRunID(t *testing.T) {
	if actual := sanitizeRunID(" run/id 1 "); actual != "run-id-1" {
		t.Fatalf("expected sanitized run id %q, got %q", "run-id-1", actual)
	}
}

func testCorpusObjects() []CorpusObject {
	return []CorpusObject{
		{Role: "primary", DRSID: "object-1", Path: "/tempZone/home/test1/drs-certification/run/basic-object.txt"},
		{Role: "bulk-1", DRSID: "object-2", Path: "/tempZone/home/test1/drs-certification/run/bulk-1.txt"},
		{Role: "compound", DRSID: "compound-1", Path: "/tempZone/home/test1/drs-certification/run/compound-root", IsCompound: true},
	}
}

func testTime() time.Time {
	return time.Date(2026, time.May, 20, 12, 0, 0, 0, time.UTC)
}

func assertNoRunPhaseOptions(t *testing.T, data []byte) {
	t.Helper()
	text := string(data)
	for _, forbidden := range []string{"reportPath", "CERTIFICATION.md", "serverBaseUrl", "localhost:8888"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("expected no run-phase option %q in JSON artifact, got %s", forbidden, text)
		}
	}
}

func assertExpectedStatuses(t *testing.T, statuses []int) {
	t.Helper()
	if len(statuses) != 2 || statuses[0] != 401 || statuses[1] != 403 {
		t.Fatalf("expected statuses [401 403], got %+v", statuses)
	}
}
