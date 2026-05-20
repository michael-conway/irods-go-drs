package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	irodsfs "github.com/cyverse/go-irodsclient/fs"
	irodslowfs "github.com/cyverse/go-irodsclient/irods/fs"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
	extmetadata "github.com/michael-conway/go-irodsclient-extensions/metadata"
	extmetadatairodsfs "github.com/michael-conway/go-irodsclient-extensions/metadata/irodsfs"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
)

const (
	defaultOutputDir     = ".certification/drs"
	defaultReportPath    = "CERTFICATION.md"
	defaultSuiteBin      = "drs-compliance-suite"
	corpusSchemaVersion  = "1"
	complianceDRSVersion = "1.5.0"
)

type certificationFS struct {
	*irodsfs.FileSystem
}

func (f *certificationFS) QueryMetadataEntries(query extmetadata.EntryQuery) (extmetadata.EntryQueryResult, error) {
	return extmetadatairodsfs.NewAdapter(f.FileSystem).QueryEntries(query)
}

func (f *certificationFS) EnsureDataObjectChecksum(irodsPath string) (*irodstypes.IRODSChecksum, error) {
	conn, err := f.FileSystem.GetMetadataConnection(true)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.FileSystem.ReturnMetadataConnection(conn)
	}()

	return irodslowfs.GetDataObjectChecksum(conn, irodsPath, "")
}

type prepareOptions struct {
	configFile      string
	serverBaseURL   string
	outputDir       string
	runID           string
	reportPath      string
	bearerTokenFile string
}

type runOptions struct {
	outputDir      string
	corpusFile     string
	configFile     string
	reportPath     string
	suiteBin       string
	serverBaseURL  string
	platformName   string
	platformDesc   string
	complianceVers string
}

type cleanupOptions struct {
	configFile string
	corpusFile string
}

type Corpus struct {
	SchemaVersion        string         `json:"schemaVersion"`
	RunID                string         `json:"runId"`
	CreatedAt            time.Time      `json:"createdAt"`
	ConfigFile           string         `json:"configFile,omitempty"`
	ServerBaseURL        string         `json:"serverBaseUrl"`
	RootPath             string         `json:"rootPath"`
	EffectiveUser        string         `json:"effectiveUser"`
	ComplianceConfigPath string         `json:"complianceConfigPath"`
	ReportPath           string         `json:"reportPath"`
	BearerAuthEnabled    bool           `json:"bearerAuthEnabled,omitempty"`
	Objects              []CorpusObject `json:"objects"`
}

type CorpusObject struct {
	Role       string `json:"role"`
	DRSID      string `json:"drsId"`
	Path       string `json:"path"`
	IsCompound bool   `json:"isCompound,omitempty"`
}

type ComplianceConfig struct {
	ServiceInfo     ComplianceAuthObject      `json:"service_info"`
	DRSObjectInfo   []ComplianceDRSObjectInfo `json:"drs_object_info"`
	DRSObjectAccess []ComplianceAuthObject    `json:"drs_object_access"`
	NegativeTests   ComplianceNegativeTests   `json:"negative_tests"`
}

type ComplianceAuthObject struct {
	DRSID     string `json:"drs_id,omitempty"`
	AuthType  string `json:"auth_type"`
	AuthToken any    `json:"auth_token"`
}

type ComplianceDRSObjectInfo struct {
	DRSID                string `json:"drs_id"`
	AuthType             string `json:"auth_type"`
	AuthToken            any    `json:"auth_token"`
	IsBundle             bool   `json:"is_bundle"`
	IsCompound           bool   `json:"is_compound"`
	CompoundManifestType string `json:"compound_manifest_type"`
}

type ComplianceNegativeTests struct {
	InvalidDRSIDs    []ComplianceInvalidDRSID  `json:"invalid_drs_ids,omitempty"`
	InvalidAuth      []ComplianceInvalidAuth   `json:"invalid_auth,omitempty"`
	InvalidAccessIDs []ComplianceInvalidAccess `json:"invalid_access_ids,omitempty"`
}

type ComplianceInvalidDRSID struct {
	DRSID          string `json:"drs_id"`
	AuthType       string `json:"auth_type"`
	AuthToken      any    `json:"auth_token"`
	ExpectedStatus int    `json:"expected_status"`
}

type ComplianceInvalidAuth struct {
	DRSID            string `json:"drs_id"`
	AuthType         string `json:"auth_type"`
	AuthToken        any    `json:"auth_token"`
	ExpectedStatuses []int  `json:"expected_statuses"`
}

type ComplianceInvalidAccess struct {
	DRSID          string `json:"drs_id"`
	AccessID       string `json:"access_id"`
	AuthType       string `json:"auth_type"`
	AuthToken      any    `json:"auth_token"`
	ExpectedStatus int    `json:"expected_status"`
}

type runRecord struct {
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt"`
	Command     []string  `json:"command"`
	ExitCode    int       `json:"exitCode"`
	ReportPath  string    `json:"reportPath"`
}

func main() {
	if err := runCLI(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCLI(ctx context.Context, args []string) error {
	if len(args) == 0 {
		writeUsage(os.Stderr)
		return fmt.Errorf("command is required")
	}

	switch args[0] {
	case "prepare":
		opts, err := parsePrepareOptions(args[1:])
		if err != nil {
			return err
		}
		corpus, err := prepareCorpus(opts)
		if err != nil {
			return err
		}
		return writeJSON(os.Stdout, corpus)
	case "run":
		opts, err := parseRunOptions(args[1:])
		if err != nil {
			return err
		}
		return runComplianceSuite(ctx, opts)
	case "cleanup":
		opts, err := parseCleanupOptions(args[1:])
		if err != nil {
			return err
		}
		return cleanupCorpus(opts)
	case "all":
		prepareOpts, runOpts, err := parseAllOptions(args[1:])
		if err != nil {
			return err
		}
		corpus, err := prepareCorpus(prepareOpts)
		if err != nil {
			return err
		}
		runOpts.outputDir = prepareOpts.outputDir
		runOpts.corpusFile = filepath.Join(prepareOpts.outputDir, "corpus.json")
		runOpts.configFile = corpus.ComplianceConfigPath
		runOpts.reportPath = corpus.ReportPath
		if strings.TrimSpace(runOpts.serverBaseURL) == "" {
			runOpts.serverBaseURL = corpus.ServerBaseURL
		}
		return runComplianceSuite(ctx, runOpts)
	case "-h", "--help", "help":
		writeUsage(os.Stdout)
		return nil
	default:
		writeUsage(os.Stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func parsePrepareOptions(args []string) (prepareOptions, error) {
	opts := prepareOptions{}
	fs := flag.NewFlagSet("prepare", flag.ContinueOnError)
	fs.StringVar(&opts.configFile, "drs-config", "", "DRS YAML config path; defaults to DRS_E2E_CONFIG_FILE or DRS_CONFIG_FILE")
	fs.StringVar(&opts.serverBaseURL, "server-base-url", "", "DRS API base URL; defaults to http://localhost:<DrsListenPort>/ga4gh/drs/v1")
	fs.StringVar(&opts.outputDir, "output-dir", defaultOutputDir, "directory for corpus and compliance-suite config artifacts")
	fs.StringVar(&opts.runID, "run-id", "", "stable run id for the corpus root; defaults to a timestamp")
	fs.StringVar(&opts.reportPath, "report-path", defaultReportPath, "Markdown report path recorded for the compliance run")
	fs.StringVar(&opts.bearerTokenFile, "bearer-token-file", "", "file containing a bearer auth token; enables bearer-auth object checks")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	return opts, nil
}

func parseRunOptions(args []string) (runOptions, error) {
	opts := runOptions{}
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.StringVar(&opts.outputDir, "output-dir", defaultOutputDir, "directory containing prepared corpus artifacts")
	fs.StringVar(&opts.corpusFile, "corpus", "", "corpus JSON path; defaults to <output-dir>/corpus.json")
	fs.StringVar(&opts.configFile, "config-file", "", "compliance-suite JSON config path; defaults to corpus value")
	fs.StringVar(&opts.reportPath, "report-path", "", "Markdown report path; defaults to corpus value")
	fs.StringVar(&opts.suiteBin, "suite-bin", defaultSuiteBin, "drs-compliance-suite executable")
	fs.StringVar(&opts.serverBaseURL, "server-base-url", "", "DRS API base URL; defaults to corpus value")
	fs.StringVar(&opts.platformName, "platform-name", "irods-go-drs", "platform name for compliance report")
	fs.StringVar(&opts.platformDesc, "platform-description", "iRODS-backed GA4GH DRS implementation", "platform description for compliance report")
	fs.StringVar(&opts.complianceVers, "version", complianceDRSVersion, "DRS specification version to test")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	return opts, nil
}

func parseCleanupOptions(args []string) (cleanupOptions, error) {
	opts := cleanupOptions{}
	fs := flag.NewFlagSet("cleanup", flag.ContinueOnError)
	fs.StringVar(&opts.configFile, "drs-config", "", "DRS YAML config path; defaults to corpus value, DRS_E2E_CONFIG_FILE, or DRS_CONFIG_FILE")
	fs.StringVar(&opts.corpusFile, "corpus", filepath.Join(defaultOutputDir, "corpus.json"), "corpus JSON path")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	return opts, nil
}

func parseAllOptions(args []string) (prepareOptions, runOptions, error) {
	prepareOpts := prepareOptions{}
	runOpts := runOptions{
		suiteBin:       defaultSuiteBin,
		platformName:   "irods-go-drs",
		platformDesc:   "iRODS-backed GA4GH DRS implementation",
		complianceVers: complianceDRSVersion,
	}
	fs := flag.NewFlagSet("all", flag.ContinueOnError)
	fs.StringVar(&prepareOpts.configFile, "drs-config", "", "DRS YAML config path; defaults to DRS_E2E_CONFIG_FILE or DRS_CONFIG_FILE")
	fs.StringVar(&prepareOpts.serverBaseURL, "server-base-url", "", "DRS API base URL; defaults to http://localhost:<DrsListenPort>/ga4gh/drs/v1")
	fs.StringVar(&prepareOpts.outputDir, "output-dir", defaultOutputDir, "directory for corpus and compliance-suite artifacts")
	fs.StringVar(&prepareOpts.runID, "run-id", "", "stable run id for the corpus root; defaults to a timestamp")
	fs.StringVar(&prepareOpts.reportPath, "report-path", defaultReportPath, "Markdown report path recorded for the compliance run")
	fs.StringVar(&prepareOpts.bearerTokenFile, "bearer-token-file", "", "file containing a bearer auth token; enables bearer-auth object checks")
	fs.StringVar(&runOpts.suiteBin, "suite-bin", defaultSuiteBin, "drs-compliance-suite executable")
	fs.StringVar(&runOpts.platformName, "platform-name", runOpts.platformName, "platform name for compliance report")
	fs.StringVar(&runOpts.platformDesc, "platform-description", runOpts.platformDesc, "platform description for compliance report")
	fs.StringVar(&runOpts.complianceVers, "version", complianceDRSVersion, "DRS specification version to test")
	if err := fs.Parse(args); err != nil {
		return prepareOpts, runOpts, err
	}
	runOpts.serverBaseURL = prepareOpts.serverBaseURL
	return prepareOpts, runOpts, nil
}

func prepareCorpus(opts prepareOptions) (*Corpus, error) {
	config, resolvedConfigFile, err := readConfigForTool(opts.configFile)
	if err != nil {
		return nil, err
	}

	if err := requirePrepareConfig(config); err != nil {
		return nil, err
	}

	bearerToken, err := readBearerTokenFile(opts.bearerTokenFile)
	if err != nil {
		return nil, err
	}

	if opts.runID == "" {
		opts.runID = fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}
	opts.runID = sanitizeRunID(opts.runID)
	if opts.runID == "" {
		return nil, fmt.Errorf("run id is empty after sanitization")
	}
	if strings.TrimSpace(opts.outputDir) == "" {
		opts.outputDir = defaultOutputDir
	}
	if opts.serverBaseURL == "" {
		opts.serverBaseURL = defaultServerBaseURL(config)
	}
	if strings.TrimSpace(opts.reportPath) == "" {
		opts.reportPath = defaultReportPath
	}

	if err := os.MkdirAll(opts.outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir %q: %w", opts.outputDir, err)
	}

	filesystem, err := connectCertificationFS(config, config.IrodsPrimaryTestUser)
	if err != nil {
		return nil, err
	}
	defer filesystem.Release()

	rootPath := path.Join("/", config.IrodsZone, "home", config.IrodsPrimaryTestUser, "drs-certification", opts.runID)
	if err := filesystem.MakeDir(rootPath, true); err != nil {
		return nil, fmt.Errorf("create certification root %q: %w", rootPath, err)
	}

	objects, err := createCertificationObjects(filesystem, rootPath)
	if err != nil {
		return nil, err
	}

	complianceConfigPath := filepath.Join(opts.outputDir, "drs-compliance-config.json")
	complianceConfig, err := buildComplianceConfig(objects, config.IrodsPrimaryTestUser, config.IrodsPrimaryTestPassword, bearerToken, opts.runID)
	if err != nil {
		return nil, err
	}
	if err := writeJSONFile(complianceConfigPath, complianceConfig); err != nil {
		return nil, err
	}

	corpus := &Corpus{
		SchemaVersion:        corpusSchemaVersion,
		RunID:                opts.runID,
		CreatedAt:            time.Now().UTC(),
		ConfigFile:           resolvedConfigFile,
		ServerBaseURL:        opts.serverBaseURL,
		RootPath:             rootPath,
		EffectiveUser:        config.IrodsPrimaryTestUser,
		ComplianceConfigPath: complianceConfigPath,
		ReportPath:           opts.reportPath,
		BearerAuthEnabled:    bearerToken != "",
		Objects:              objects,
	}
	if err := writeJSONFile(filepath.Join(opts.outputDir, "corpus.json"), corpus); err != nil {
		return nil, err
	}

	return corpus, nil
}

func createCertificationObjects(filesystem *certificationFS, rootPath string) ([]CorpusObject, error) {
	objects := make([]CorpusObject, 0, 5)

	primary, err := createDataObjectFixture(
		filesystem,
		path.Join(rootPath, "basic-object.txt"),
		"drs certification primary object\n",
		"certification primary object",
		[]string{"drs-certification-primary"},
	)
	if err != nil {
		return nil, err
	}
	primary.Role = "primary"
	objects = append(objects, primary)

	for index := 1; index <= 3; index++ {
		object, err := createDataObjectFixture(
			filesystem,
			path.Join(rootPath, fmt.Sprintf("bulk-%d.txt", index)),
			fmt.Sprintf("drs certification bulk object %d\n", index),
			fmt.Sprintf("certification bulk object %d", index),
			[]string{fmt.Sprintf("drs-certification-bulk-%d", index)},
		)
		if err != nil {
			return nil, err
		}
		object.Role = fmt.Sprintf("bulk-%d", index)
		objects = append(objects, object)
	}

	compound, err := createCompoundFixture(filesystem, rootPath)
	if err != nil {
		return nil, err
	}
	objects = append(objects, compound)

	return objects, nil
}

func createDataObjectFixture(filesystem *certificationFS, objectPath string, content string, description string, aliases []string) (CorpusObject, error) {
	if _, err := filesystem.UploadFileFromBuffer(bytes.NewBufferString(content), objectPath, "", false, true, nil); err != nil {
		return CorpusObject{}, fmt.Errorf("upload object %q: %w", objectPath, err)
	}

	drsID, err := drs_support.CreateDrsObjectFromDataObject(filesystem, objectPath, "", description, aliases)
	if err != nil {
		return CorpusObject{}, fmt.Errorf("create DRS object for %q: %w", objectPath, err)
	}

	return CorpusObject{
		DRSID: drsID,
		Path:  objectPath,
	}, nil
}

func createCompoundFixture(filesystem *certificationFS, rootPath string) (CorpusObject, error) {
	compoundPath := path.Join(rootPath, "compound-root")
	includedCollectionPath := path.Join(compoundPath, "included")
	ignoredCollectionPath := path.Join(compoundPath, "ignored")

	if err := filesystem.MakeDir(includedCollectionPath, true); err != nil {
		return CorpusObject{}, fmt.Errorf("create compound included collection %q: %w", includedCollectionPath, err)
	}
	if err := filesystem.MakeDir(ignoredCollectionPath, true); err != nil {
		return CorpusObject{}, fmt.Errorf("create compound ignored collection %q: %w", ignoredCollectionPath, err)
	}
	if _, err := filesystem.UploadFileFromBuffer(
		bytes.NewBufferString("drs certification compound child\n"),
		path.Join(includedCollectionPath, "child.txt"),
		"",
		false,
		true,
		nil,
	); err != nil {
		return CorpusObject{}, fmt.Errorf("upload compound child: %w", err)
	}
	if _, err := filesystem.UploadFileFromBuffer(
		bytes.NewBufferString("drs certification ignored child\n"),
		path.Join(ignoredCollectionPath, "ignored.txt"),
		"",
		false,
		true,
		nil,
	); err != nil {
		return CorpusObject{}, fmt.Errorf("upload compound ignored child: %w", err)
	}
	if _, err := filesystem.UploadFileFromBuffer(
		bytes.NewBufferString("# DRS certification compound fixture\nignored/\n"),
		path.Join(compoundPath, drs_support.DrsIgnoreFileName),
		"",
		false,
		true,
		nil,
	); err != nil {
		return CorpusObject{}, fmt.Errorf("upload compound ignore file: %w", err)
	}

	result, err := drs_support.CreateCompoundDrsObjectFromCollection(filesystem, compoundPath)
	if err != nil {
		return CorpusObject{}, fmt.Errorf("create compound DRS object at %q: %w", compoundPath, err)
	}
	if result == nil {
		return CorpusObject{}, fmt.Errorf("create compound DRS object at %q returned nil result", compoundPath)
	}
	if len(result.NodeErrors) > 0 {
		return CorpusObject{}, fmt.Errorf("create compound DRS object at %q reported node errors: %+v", compoundPath, result.NodeErrors)
	}

	return CorpusObject{
		Role:       "compound",
		DRSID:      result.DrsID,
		Path:       compoundPath,
		IsCompound: true,
	}, nil
}

func buildComplianceConfig(objects []CorpusObject, username string, password string, bearerToken string, runID string) (*ComplianceConfig, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	bearerToken = strings.TrimSpace(bearerToken)
	if username == "" {
		return nil, fmt.Errorf("username is required for compliance Basic auth config")
	}
	if password == "" {
		return nil, fmt.Errorf("password is required for compliance Basic auth config")
	}

	primary, ok := findCorpusObject(objects, "primary")
	if !ok {
		return nil, fmt.Errorf("primary corpus object is required")
	}

	validBasicToken := basicToken(username, password)
	invalidBasicToken := basicToken(username, "__drs_certification_invalid_password__")

	objectInfo := make([]ComplianceDRSObjectInfo, 0, len(objects))
	for _, object := range objects {
		if object.DRSID == "" {
			continue
		}
		objectInfo = append(objectInfo, complianceObjectInfo(object, "basic", validBasicToken))
		if bearerToken != "" {
			objectInfo = append(objectInfo, complianceObjectInfo(object, "bearer", bearerToken))
		}
	}

	objectAccess := []ComplianceAuthObject{{
		DRSID:     primary.DRSID,
		AuthType:  "basic",
		AuthToken: validBasicToken,
	}}
	if bearerToken != "" {
		objectAccess = append(objectAccess, ComplianceAuthObject{
			DRSID:     primary.DRSID,
			AuthType:  "bearer",
			AuthToken: bearerToken,
		})
	}

	invalidAuth := []ComplianceInvalidAuth{{
		DRSID:            primary.DRSID,
		AuthType:         "basic",
		AuthToken:        invalidBasicToken,
		ExpectedStatuses: []int{401, 403},
	}}
	if bearerToken != "" {
		invalidAuth = append(invalidAuth, ComplianceInvalidAuth{
			DRSID:            primary.DRSID,
			AuthType:         "bearer",
			AuthToken:        "__drs_certification_invalid_bearer__",
			ExpectedStatuses: []int{401, 403},
		})
	}

	return &ComplianceConfig{
		ServiceInfo: ComplianceAuthObject{
			AuthType:  "none",
			AuthToken: "",
		},
		DRSObjectInfo:   objectInfo,
		DRSObjectAccess: objectAccess,
		NegativeTests: ComplianceNegativeTests{
			InvalidDRSIDs: []ComplianceInvalidDRSID{{
				DRSID:          "__drs_certification_missing_object_" + sanitizeRunID(runID),
				AuthType:       "basic",
				AuthToken:      validBasicToken,
				ExpectedStatus: 404,
			}},
			InvalidAuth: invalidAuth,
			InvalidAccessIDs: []ComplianceInvalidAccess{{
				DRSID:          primary.DRSID,
				AccessID:       "__drs_certification_missing_access__",
				AuthType:       "basic",
				AuthToken:      validBasicToken,
				ExpectedStatus: 404,
			}},
		},
	}, nil
}

func complianceObjectInfo(object CorpusObject, authType string, authToken any) ComplianceDRSObjectInfo {
	return ComplianceDRSObjectInfo{
		DRSID:                object.DRSID,
		AuthType:             authType,
		AuthToken:            authToken,
		IsBundle:             false,
		IsCompound:           object.IsCompound,
		CompoundManifestType: "json",
	}
}

func findCorpusObject(objects []CorpusObject, role string) (CorpusObject, bool) {
	for _, object := range objects {
		if object.Role == role {
			return object, true
		}
	}
	return CorpusObject{}, false
}

func basicToken(username string, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(strings.TrimSpace(username) + ":" + password))
}

func readBearerTokenFile(tokenFile string) (string, error) {
	tokenFile = strings.TrimSpace(tokenFile)
	if tokenFile == "" {
		return "", nil
	}

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", fmt.Errorf("read bearer token file %q: %w", tokenFile, err)
	}
	token := strings.TrimSpace(string(data))
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[len("bearer "):])
	}
	if token == "" {
		return "", fmt.Errorf("bearer token file %q is empty", tokenFile)
	}
	return token, nil
}

func runComplianceSuite(ctx context.Context, opts runOptions) error {
	corpusPath := opts.corpusFile
	if corpusPath == "" {
		corpusPath = filepath.Join(opts.outputDir, "corpus.json")
	}
	corpus, err := readCorpus(corpusPath)
	if err != nil {
		return err
	}

	configFile := firstNonEmpty(opts.configFile, corpus.ComplianceConfigPath)
	reportPath := firstNonEmpty(opts.reportPath, corpus.ReportPath)
	serverBaseURL := firstNonEmpty(opts.serverBaseURL, corpus.ServerBaseURL)
	if configFile == "" {
		return fmt.Errorf("compliance config file is required")
	}
	if reportPath == "" {
		reportPath = defaultReportPath
	}
	if serverBaseURL == "" {
		return fmt.Errorf("server base URL is required")
	}
	if opts.suiteBin == "" {
		opts.suiteBin = defaultSuiteBin
	}
	if opts.platformName == "" {
		opts.platformName = "irods-go-drs"
	}
	if opts.platformDesc == "" {
		opts.platformDesc = "iRODS-backed GA4GH DRS implementation"
	}
	if opts.complianceVers == "" {
		opts.complianceVers = complianceDRSVersion
	}

	args := []string{
		"--server_base_url", serverBaseURL,
		"--platform_name", opts.platformName,
		"--platform_description", opts.platformDesc,
		"--version", opts.complianceVers,
		"--config_file", configFile,
		"--report_path", reportPath,
	}

	startedAt := time.Now().UTC()
	cmd := exec.CommandContext(ctx, opts.suiteBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	completedAt := time.Now().UTC()

	exitCode := 0
	if err != nil {
		exitCode = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}

	record := runRecord{
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		Command:     append([]string{opts.suiteBin}, args...),
		ExitCode:    exitCode,
		ReportPath:  reportPath,
	}
	if writeErr := writeJSONFile(filepath.Join(filepath.Dir(corpusPath), "run.json"), record); writeErr != nil && err == nil {
		return writeErr
	}
	return err
}

func cleanupCorpus(opts cleanupOptions) error {
	corpus, err := readCorpus(opts.corpusFile)
	if err != nil {
		return err
	}

	configFile := firstNonEmpty(opts.configFile, corpus.ConfigFile)
	config, _, err := readConfigForTool(configFile)
	if err != nil {
		return err
	}

	if err := requirePrepareConfig(config); err != nil {
		return err
	}

	filesystem, err := connectCertificationFS(config, corpus.EffectiveUser)
	if err != nil {
		return err
	}
	defer filesystem.Release()

	if strings.TrimSpace(corpus.RootPath) == "" || corpus.RootPath == "/" {
		return fmt.Errorf("refusing to clean empty or root corpus path %q", corpus.RootPath)
	}
	if err := filesystem.RemoveDir(corpus.RootPath, true, true); err != nil && filesystem.Exists(corpus.RootPath) {
		return fmt.Errorf("cleanup corpus root %q: %w", corpus.RootPath, err)
	}
	return nil
}

func readConfigForTool(configFile string) (*drs_support.DrsConfig, string, error) {
	resolvedConfigFile, err := resolveToolConfigPath(configFile)
	if err != nil {
		return nil, "", err
	}

	oldValue, hadOldValue := os.LookupEnv(drs_support.ConfigFileEnvVar)
	if resolvedConfigFile != "" {
		if err := os.Setenv(drs_support.ConfigFileEnvVar, resolvedConfigFile); err != nil {
			return nil, "", err
		}
	}
	defer func() {
		if hadOldValue {
			_ = os.Setenv(drs_support.ConfigFileEnvVar, oldValue)
		} else {
			_ = os.Unsetenv(drs_support.ConfigFileEnvVar)
		}
	}()

	cfg, err := drs_support.ReadDrsConfig("", "", nil)
	if err != nil {
		return nil, resolvedConfigFile, err
	}
	return cfg, resolvedConfigFile, nil
}

func resolveToolConfigPath(configFile string) (string, error) {
	for _, candidate := range []string{
		configFile,
		os.Getenv("DRS_E2E_CONFIG_FILE"),
		os.Getenv(drs_support.ConfigFileEnvVar),
	} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		resolved, err := filepath.Abs(candidate)
		if err != nil {
			return "", fmt.Errorf("resolve config path %q: %w", candidate, err)
		}
		return resolved, nil
	}

	return "", fmt.Errorf("DRS config is required; pass --drs-config or set DRS_E2E_CONFIG_FILE or %s", drs_support.ConfigFileEnvVar)
}

func requirePrepareConfig(cfg *drs_support.DrsConfig) error {
	if cfg == nil {
		return fmt.Errorf("missing DRS config")
	}
	required := map[string]string{
		"IrodsHost":                cfg.IrodsHost,
		"IrodsZone":                cfg.IrodsZone,
		"IrodsAdminUser":           cfg.IrodsAdminUser,
		"IrodsAdminPassword":       cfg.IrodsAdminPassword,
		"IrodsPrimaryTestUser":     cfg.IrodsPrimaryTestUser,
		"IrodsPrimaryTestPassword": cfg.IrodsPrimaryTestPassword,
		"IrodsAuthScheme":          cfg.IrodsAuthScheme,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("DRS certification requires %s in the DRS config", field)
		}
	}
	if cfg.IrodsPort <= 0 {
		return fmt.Errorf("DRS certification requires IrodsPort in the DRS config")
	}
	return nil
}

func connectCertificationFS(cfg *drs_support.DrsConfig, effectiveUser string) (*certificationFS, error) {
	effectiveUser = strings.TrimSpace(effectiveUser)
	if effectiveUser == "" {
		effectiveUser = strings.TrimSpace(cfg.IrodsPrimaryTestUser)
	}

	account, err := irodstypes.CreateIRODSProxyAccount(
		cfg.IrodsHost,
		cfg.IrodsPort,
		effectiveUser,
		cfg.IrodsZone,
		cfg.IrodsAdminUser,
		cfg.IrodsZone,
		cfg.AdminAuthScheme(),
		cfg.IrodsAdminPassword,
		cfg.IrodsDefaultResource,
	)
	if err != nil {
		return nil, fmt.Errorf("create iRODS proxy account: %w", err)
	}
	cfg.ApplyIRODSConnectionConfig(account)

	filesystem, err := irodsfs.NewFileSystemWithDefault(account, "irods-go-drs-certification")
	if err != nil {
		return nil, fmt.Errorf("connect to iRODS: %w", err)
	}
	return &certificationFS{FileSystem: filesystem}, nil
}

func defaultServerBaseURL(cfg *drs_support.DrsConfig) string {
	port := 8080
	if cfg != nil && cfg.DrsListenPort > 0 {
		port = cfg.DrsListenPort
	}
	return fmt.Sprintf("http://localhost:%d/ga4gh/drs/v1", port)
}

func readCorpus(corpusPath string) (*Corpus, error) {
	corpusPath = strings.TrimSpace(corpusPath)
	if corpusPath == "" {
		return nil, fmt.Errorf("corpus path is required")
	}

	data, err := os.ReadFile(corpusPath)
	if err != nil {
		return nil, fmt.Errorf("read corpus %q: %w", corpusPath, err)
	}
	corpus := &Corpus{}
	if err := json.Unmarshal(data, corpus); err != nil {
		return nil, fmt.Errorf("decode corpus %q: %w", corpusPath, err)
	}
	return corpus, nil
}

func writeJSONFile(filePath string, value any) error {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return fmt.Errorf("output file path is required")
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("create output directory for %q: %w", filePath, err)
	}

	tmpPath := filePath + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create %q: %w", tmpPath, err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		_ = file.Close()
		return fmt.Errorf("encode %q: %w", tmpPath, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %q: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("rename %q to %q: %w", tmpPath, filePath, err)
	}
	return nil
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func sanitizeRunID(runID string) string {
	runID = strings.TrimSpace(runID)
	var builder strings.Builder
	for _, r := range runID {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-', r == '_', r == '.':
			builder.WriteRune(r)
		default:
			builder.WriteRune('-')
		}
	}
	return strings.Trim(builder.String(), "-.")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func writeUsage(w io.Writer) {
	fmt.Fprintln(w, "drscert prepares an iRODS-backed corpus and compliance-suite config for irods-go-drs self-testing.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  drscert prepare --drs-config <config.yaml> --server-base-url <url> --output-dir .certification/drs [--bearer-token-file token.txt]")
	fmt.Fprintln(w, "  drscert run --output-dir .certification/drs --suite-bin drs-compliance-suite [--report-path CERTFICATION.md]")
	fmt.Fprintln(w, "  drscert cleanup --corpus .certification/drs/corpus.json")
	fmt.Fprintln(w, "  drscert all --drs-config <config.yaml> --server-base-url <url> --suite-bin drs-compliance-suite [--bearer-token-file token.txt]")
}
