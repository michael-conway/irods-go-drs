package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"

	irodsclientconfig "github.com/cyverse/go-irodsclient/config"
	"github.com/cyverse/go-irodsclient/fs"
	"github.com/cyverse/go-irodsclient/irods/types"
	drs_support "github.com/michael-conway/irods-go-drs/drs-support"
	"github.com/urfave/cli/v3"
)

type FileSystem interface {
	drs_support.IRODSFilesystem
	GetHomeDirPath() string
	Release()
}

type realFileSystem struct {
	*fs.FileSystem
}

func (f *realFileSystem) Release() {
	f.FileSystem.Release()
}

var createFileSystem = func(account *types.IRODSAccount, applicationName string) (FileSystem, error) {
	filesystem, err := fs.NewFileSystemWithDefault(account, applicationName)
	if err != nil {
		return nil, err
	}
	return &realFileSystem{filesystem}, nil
}

var envManager *irodsclientconfig.ICommandsEnvironmentManager

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

const APP_NAME = "drscmd"

type drsInfoResult struct {
	DRSID       string   `json:"drsId"`
	Path        string   `json:"path"`
	Zone        string   `json:"zone,omitempty"`
	Size        int64    `json:"size,omitempty"`
	Version     string   `json:"version,omitempty"`
	MimeType    string   `json:"mimeType,omitempty"`
	IsManifest  bool     `json:"isManifest"`
	Description string   `json:"description,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
}

type drsMakeResult struct {
	DRSID string `json:"drsId"`
	Path  string `json:"path"`
}

type drsRemoveResult struct {
	Path string `json:"path"`
}

func init() {
	setupCLI()
	setupEnvironment()
}

func setupCLI() {
	cli.RootCommandHelpTemplate += "\nDRS Administration Command Tool\n"

	cli.HelpFlag = &cli.BoolFlag{Name: "help"}
	cli.VersionFlag = &cli.BoolFlag{Name: "print-version", Aliases: []string{"V"}}

	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		fmt.Fprintf(w, "drscmd help\n")
		fmt.Fprintf(w, "\thelp - this help screen\n")
		fmt.Fprintf(w, "\t-V - Version\n")
		fmt.Fprintf(w, "\n\n")
		fmt.Fprintf(w, "\t------- session management --------\n\n")
		fmt.Fprintf(w, "\tiinit - initialize a connection\n")
		fmt.Fprintf(w, "\n\n")
		fmt.Fprintf(w, "\t------- drs --------\n\n")
		fmt.Fprintf(w, "\tdrsinfo - drs info for a given path or drs id\n")
		fmt.Fprintf(w, "\tdrsmake - make a single-object drs object at the given data object\n")
		fmt.Fprintf(w, "\tdrsrm - remove drs object characteristics from a given data object\n")
	}

	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Fprintf(cmd.Root().Writer, "version=%s\n", cmd.Root().Version)
	}
}

func setupEnvironment() {
	manager, err := irodsclientconfig.NewICommandsEnvironmentManager()
	if err != nil {
		logger.Error("error loading environment manager", "error", err)
		return
	}

	envManager = manager
}

func getCommand() *cli.Command {
	var authType string
	var zone string
	var host string
	var port int
	var user string
	var password string

	var drsPath bool
	var drsID bool
	var mimeType string
	var description string
	var aliases []string

	return &cli.Command{
		Name:    APP_NAME,
		Usage:   "DRS administration command tool for iRODS",
		Version: "dev",
		Commands: []*cli.Command{
			{
				Name:  "iinit",
				Usage: "initialize a connection to iRODS and save it as an iRODS environment for later use",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "t", Value: "native", Usage: "auth type", Destination: &authType},
					&cli.StringFlag{Name: "z", Value: "", Usage: "zone", Destination: &zone},
					&cli.StringFlag{Name: "h", Value: "", Usage: "host", Destination: &host},
					&cli.IntFlag{Name: "o", Value: 1247, Usage: "port", Destination: &port},
					&cli.StringFlag{Name: "u", Value: "", Usage: "user name", Destination: &user},
					&cli.StringFlag{Name: "p", Value: "", Usage: "password", Destination: &password},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					irodsAccount := types.IRODSAccount{
						AuthenticationScheme: types.GetAuthScheme(authType),
						Host:                 host,
						Port:                 port,
						ClientUser:           user,
						ClientZone:           zone,
						Password:             password,
					}

					envManager.FromIRODSAccount(&irodsAccount)
					if err := envManager.SaveEnvironment(); err != nil {
						logger.Error("error saving iRODS environment", "error", err)
						fmt.Fprintf(cmd.ErrWriter, "error saving iRODS environment\n")
						return err
					}

					fmt.Fprintf(cmd.Writer, "saved iRODS environment to %s\n", envManager.EnvironmentFilePath)
					return nil
				},
			},
			{
				Name:      "drsinfo",
				Usage:     "show drs info for a given data object path or drs id",
				ArgsUsage: "<path-or-drs-id>",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "path", Usage: "treat the argument as an iRODS path", Destination: &drsPath},
					&cli.BoolFlag{Name: "id", Usage: "treat the argument as a DRS id", Destination: &drsID},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					target := cmd.Args().First()
					if strings.TrimSpace(target) == "" {
						return fmt.Errorf("a DRS id or iRODS path is required")
					}

					filesystem, err := connectFileSystem()
					if err != nil {
						return err
					}
					defer filesystem.Release()

					object, err := getDrsObject(filesystem, target, drsPath, drsID)
					if err != nil {
						return err
					}

					return writeJSON(cmd.Writer, drsInfoResult{
						DRSID:       object.Id,
						Path:        object.AbsolutePath,
						Zone:        object.IrodsZone,
						Size:        object.Size,
						Version:     object.Version,
						MimeType:    object.MimeType,
						IsManifest:  object.IsManifest,
						Description: object.Description,
						Aliases:     object.Aliases,
					})
				},
			},
			{
				Name:      "drsmake",
				Usage:     "decorate a single iRODS data object as a DRS object",
				ArgsUsage: "<irods-data-object-path>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "mime-type", Aliases: []string{"mime"}, Usage: "explicit MIME type to store on the DRS object"},
					&cli.StringFlag{Name: "description", Aliases: []string{"d"}, Usage: "human-readable description"},
					&cli.StringSliceFlag{Name: "alias", Aliases: []string{"a"}, Usage: "alternate identifier to store as a DRS alias"},
				},
				Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
					mimeType = cmd.String("mime-type")
					description = cmd.String("description")
					aliases = cmd.StringSlice("alias")
					return ctx, nil
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					target := cmd.Args().First()
					if strings.TrimSpace(target) == "" {
						return fmt.Errorf("an iRODS data object path is required")
					}

					filesystem, err := connectFileSystem()
					if err != nil {
						return err
					}
					defer filesystem.Release()

					targetPath, err := resolveIRODSPath(target, filesystem)
					if err != nil {
						return err
					}

					drsID, err := drs_support.CreateDrsObjectFromDataObject(filesystem, targetPath, mimeType, description, aliases)
					if err != nil {
						return err
					}

					return writeJSON(cmd.Writer, drsMakeResult{
						DRSID: drsID,
						Path:  targetPath,
					})
				},
			},
			{
				Name:      "drsrm",
				Usage:     "remove single-object DRS metadata from an iRODS data object",
				ArgsUsage: "<irods-data-object-path>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					target := cmd.Args().First()
					if strings.TrimSpace(target) == "" {
						return fmt.Errorf("an iRODS data object path is required")
					}

					filesystem, err := connectFileSystem()
					if err != nil {
						return err
					}
					defer filesystem.Release()

					targetPath, err := resolveIRODSPath(target, filesystem)
					if err != nil {
						return err
					}

					if err := drs_support.RemoveSingleDrsObjectFromDataObject(filesystem, targetPath); err != nil {
						return err
					}

					return writeJSON(cmd.Writer, drsRemoveResult{Path: targetPath})
				},
			},
		},
	}
}

func connectFileSystem() (FileSystem, error) {
	if envManager == nil {
		return nil, fmt.Errorf("iRODS environment manager is not configured")
	}

	account, err := envManager.ToIRODSAccount()
	if err != nil {
		return nil, err
	}

	return createFileSystem(account, APP_NAME)
}

func getDrsObject(filesystem FileSystem, target string, forcePath bool, forceID bool) (*drs_support.InternalDrsObject, error) {
	if forcePath && forceID {
		return nil, fmt.Errorf("--path and --id cannot be used together")
	}

	if forcePath {
		targetPath, err := resolveIRODSPath(target, filesystem)
		if err != nil {
			return nil, err
		}
		return drs_support.GetDrsObjectByIRODSPath(filesystem, targetPath)
	}

	if forceID {
		return drs_support.GetDrsObjectByID(filesystem, strings.TrimSpace(target))
	}

	if looksLikeIRODSPath(target) {
		targetPath, err := resolveIRODSPath(target, filesystem)
		if err != nil {
			return nil, err
		}
		return drs_support.GetDrsObjectByIRODSPath(filesystem, targetPath)
	}

	return drs_support.GetDrsObjectByID(filesystem, strings.TrimSpace(target))
}

func looksLikeIRODSPath(target string) bool {
	target = strings.TrimSpace(target)
	return strings.HasPrefix(target, "/") || strings.HasPrefix(target, ".")
}

func resolveIRODSPath(target string, filesystem FileSystem) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fmt.Errorf("an iRODS path is required")
	}

	if strings.HasPrefix(target, "/") {
		return path.Clean(target), nil
	}

	if envManager == nil || envManager.Environment == nil {
		return "", fmt.Errorf("iRODS environment is not configured")
	}

	cwd, err := establishCwd(envManager.Environment.CurrentWorkingDir, filesystem)
	if err != nil {
		return "", err
	}

	return path.Clean(path.Join(cwd, target)), nil
}

func establishCwd(cwd string, filesystem FileSystem) (string, error) {
	if strings.TrimSpace(cwd) == "" {
		cwd = filesystem.GetHomeDirPath()
		envManager.Environment.CurrentWorkingDir = cwd

		if err := envManager.SaveEnvironment(); err != nil {
			logger.Error("error saving default cwd to environment", "error", err)
			return "", err
		}
	}

	return cwd, nil
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func main() {
	cmd := getCommand()

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		logger.Error("error", "error", err)
	}
}
