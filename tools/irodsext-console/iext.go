package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/cyverse/go-irodsclient/config"
	"github.com/cyverse/go-irodsclient/fs"
	"github.com/cyverse/go-irodsclient/irods/types"
	"github.com/urfave/cli/v3"
)

type FileSystem interface {
	GetHomeDirPath() string
	Stat(irodsPath string) (*fs.Entry, error)
	Release()
}

type realFileSystem struct {
	*fs.FileSystem
}

func (f *realFileSystem) Stat(irodsPath string) (*fs.Entry, error) {
	return f.FileSystem.Stat(irodsPath)
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

var envManager *config.ICommandsEnvironmentManager

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

const APP_NAME = "iext"

func init() {
	setupCLI()
	setupEnvironment()
}

func setupCLI() {
	cli.RootCommandHelpTemplate += "\niRODS Extended Command Tool\n"

	cli.HelpFlag = &cli.BoolFlag{Name: "help"}
	cli.VersionFlag = &cli.BoolFlag{Name: "print-version", Aliases: []string{"V"}}

	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		fmt.Fprintf(w, "iext help\n")
		fmt.Fprintf(w, "\thelp - this help screen\n")
		fmt.Fprintf(w, "\t-V - Version\n")
		fmt.Fprintf(w, "\n\n")

		fmt.Fprintf(w, "\t------- session managment --------\n\n")

		fmt.Fprintf(w, "\tinit - initialize a connection\n")
		fmt.Fprintf(w, "\texit - initialize a connection\n")

		fmt.Fprintf(w, "\n\n")

		fmt.Fprintf(w, "\t------- drs --------\n\n")

		fmt.Fprintf(w, "\tdrsinfo - drs info for a given path or drs id \n")
		fmt.Fprintf(w, "\tdrsmake - make a drs object at the given collection or data object  \n")
		fmt.Fprintf(w, "\tdrsrm - remove drs object characteristic at a given collection or data object \n")

	}
	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Fprintf(cmd.Root().Writer, "version=%s\n", cmd.Root().Version)
	}
}

func setupEnvironment() {
	// set up context to pick up user creds

	myManager, err := config.NewICommandsEnvironmentManager()
	if err != nil {
		logger.Error("error loading environment manager", "error", err)
	}

	envManager = myManager
}

func getCommand() *cli.Command {
	var auth_type string
	var zone string
	var host string
	var port int
	var user string
	var password string

	return &cli.Command{
		Commands: []*cli.Command{
			{
				Name:  "iinit",
				Usage: "initialize a connection to iRODS and save as an iRODS Environment for later use",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "t",
						Value:       "native",
						Usage:       "auth type",
						Destination: &auth_type,
					},
					&cli.StringFlag{
						Name:        "z",
						Value:       "",
						Usage:       "zone",
						Destination: &zone,
					},
					&cli.StringFlag{
						Name:        "h",
						Value:       "",
						Usage:       "host",
						Destination: &host,
					},
					&cli.IntFlag{
						Name:        "o",
						Value:       1247,
						Usage:       "port",
						Destination: &port,
					},
					&cli.StringFlag{
						Name:        "u",
						Value:       "",
						Usage:       "user name",
						Destination: &user,
					},
					&cli.StringFlag{
						Name:        "p",
						Value:       "",
						Usage:       "password",
						Destination: &password,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					irodsAccount := types.IRODSAccount{
						AuthenticationScheme: types.GetAuthScheme(auth_type),
						Host:                 host,
						Port:                 port,
						ClientUser:           user,
						ClientZone:           zone,
						Password:             password,
					}

					envManager.FromIRODSAccount(&irodsAccount)
					err := envManager.SaveEnvironment()
					if err != nil {
						logger.Error("error saving iRODS environment", "error", err)
						fmt.Fprintf(cmd.ErrWriter, "error saving iRODS environment\n")
					}

					fmt.Fprintf(cmd.Writer, "saved iRODS environment to %s\n", envManager.EnvironmentFilePath)
					return nil

				},
			},
		},
	}
}

func establishCwd(cwd string, filesystem FileSystem, err error) (string, error) {
	if cwd == "" {
		cwd = filesystem.GetHomeDirPath()
		envManager.Environment.CurrentWorkingDir = cwd

		err = envManager.SaveEnvironment()
		if err != nil {
			logger.Error("error connecting to irods-server", "error", err)
			return "", err
		}
	}
	return cwd, nil
}

func main() {
	cmd := getCommand()

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		logger.Error("error", "error", err)
	}
}
