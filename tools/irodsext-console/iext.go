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
	"github.com/fatih/color"
	"github.com/urfave/cli/v3"
)

var envManager *config.ICommandsEnvironmentManager

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

const APP_NAME = "iext"
const IEXT_CWD = "IEXT_CWD"

func init() {

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
		fmt.Fprintf(w, "\texit - terminaterr a connection\n")

		fmt.Fprintf(w, "\n\n")

		fmt.Fprintf(w, "\t------- file and directory --------\n\n")

		fmt.Fprintf(w, "\tls - directory list \n")
		fmt.Fprintf(w, "\tcd - change directory \n")
		fmt.Fprintf(w, "\tpwd - print working directory \n")
		fmt.Fprintf(w, "\tmkdir - make a directory \n")
		fmt.Fprintf(w, "\trm - remove a directory \n")
		fmt.Fprintf(w, "\tpwd - print working directory \n")

		fmt.Fprintf(w, "\n\n")

		fmt.Fprintf(w, "\t------- drs --------\n\n")

		fmt.Fprintf(w, "\tdrsinfo - drs info for a given path or drs id \n")
		fmt.Fprintf(w, "\tdrsmake - make a drs object at the given collection or data object  \n")
		fmt.Fprintf(w, "\tdrsrm - remove drs object characteristic at a given collection or data object \n")

	}
	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Fprintf(cmd.Root().Writer, "version=%s\n", cmd.Root().Version)
	}

	// set up context to pick up user creds

	myManager, error := config.NewICommandsEnvironmentManager()
	if error != nil {
		logger.Error("error loading environment manager %v", error.Error())
	}

	envManager = myManager
	error = envManager.Load()

	if error != nil {
		logger.Error("error loading environment manager %v", error.Error())
	}

}

// Resolve the current working directory, which is stashed in the environment. If not available,
// set to the users working directory.
func resolveCwd() (string, error) {
	cwd := os.Getenv(IEXT_CWD)

	if cwd == "" {
		irodsAccount, err := obtainIrodsAccount()

		if err != nil {
			logger.Error("unable to set environment variable %s", IEXT_CWD)
			return "", err
		}

		cwd = fmt.Sprintf("/%s/home/%s", irodsAccount.ClientZone, irodsAccount.ClientUser)
		err = os.Setenv(IEXT_CWD, cwd)
		if err != nil {
			logger.Error("unable to set environment variable %s", IEXT_CWD)
			return "", err
		}
	}

	return cwd, nil

}

func main() {

	var auth_type string
	var zone string
	var host string
	var port int
	var user string
	var password string
	var longFormat bool

	cmd := &cli.Command{
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
					&cli.BoolFlag{
						Name:        "l",
						Value:       true,
						Usage:       "long format",
						Destination: &longFormat,
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
						logger.Error("error saving iRODS environment", err.Error())
						fmt.Fprintf(cmd.ErrWriter, "error saving iRODS environment\n")
					}

					fmt.Fprintf(cmd.Writer, "saved iRODS environment to %s\n", envManager.EnvironmentFilePath)
					return nil

				},
			},

			// imiscsvrinfo

			{
				Name:  "imiscsvrinfo",
				Usage: "Connect to the server and retrieve some basic server information.\nCan be used as a simple test for connecting to the server.",
				Action: func(ctx context.Context, cmd *cli.Command) error {

					filesystem, _, err := obtainFilesystem(cmd)

					if err != nil {
						logger.Error("error connecting to irods", err.Error())
						fmt.Fprintf(cmd.ErrWriter, "error connecting to irods\n")
					}

					defer filesystem.Release()
					version, err := filesystem.GetServerVersion()

					if err != nil {
						logger.Error("error connecting to irods", err.Error())
						fmt.Fprintf(cmd.ErrWriter, "error connecting to irods\n")
					}

					fmt.Fprintf(cmd.Writer, "irods version: %s\n", version.ReleaseVersion)
					fmt.Fprintf(cmd.Writer, "api version: %s\n", version.APIVersion)
					return nil

				},
			},

			// ipwd

			{
				Name:  "ipwd",
				Usage: "Connect to the server and retrieve some basic server information.\nCan be used as a simple test for connecting to the server.",
				Action: func(ctx context.Context, cmd *cli.Command) error {

					filesystem, _, err := obtainFilesystem(cmd)

					if err != nil {
						logger.Error("error connecting to irods", err.Error())
						fmt.Fprintf(cmd.ErrWriter, "error connecting to irods\n")
					}

					defer filesystem.Release()

					cwd, err := resolveCwd()

					if err != nil {
						logger.Error("error resolving cwd", err.Error())
						fmt.Fprintf(cmd.ErrWriter, "error resolving cwd\n")
					}

					fmt.Fprintf(cmd.Writer, "%s\n", cwd)
					return nil

				},
			},

			// ils

			/*Options are:
			-A  ACL (access control list) and inheritance format
			-d  List collections themselves, not their contents
			-l  long format
			-L  very long format
			-r  recursive - show subcollections
			-t  ticket - use a read (or write) ticket to access collection information
			-v  verbose
			-V  Very verbose
			-h  this help
			--bundle - list the subfiles in the bundle file (usually stored in the
			/myZone/bundle collection) created by iphybun command.*/

			{
				Name:  "ils",
				Usage: "Display data objects and collections stored in iRODS.",
				Action: func(ctx context.Context, cmd *cli.Command) error {

					cwd, err := resolveCwd()

					if err != nil {
						logger.Error("error resolving cwd", err.Error())
						fmt.Fprintf(cmd.ErrWriter, "error resolving cwd\n")
					}

					filesystem, _, err := obtainFilesystem(cmd)

					defer filesystem.Release()

					entries, err := filesystem.List(cwd)
					if err != nil {
						logger.Error("error listing filesystem", err.Error())
						fmt.Fprintf(cmd.ErrWriter, "error listing filesystem\n")
					}

					if len(entries) == 0 {
						fmt.Printf("\n")
					} else {
						for _, entry := range entries {
							if entry.Type == fs.FileEntry {
								fmt.Printf("%s\n", entry.Name)
							} else {
								color.Blue(entry.Name)
							}

						}
					}

					return nil

				},
			},
			// icd
			/*

				Change the current working collection.

				Usage: icd [-vVh] [COLLECTION]

				If invoked without any arguments, the current working collection is set
				back to your home collection as defined in your irods_environment.json
				file.

				If COLLECTION matches "..", the current working collection is set to the
				parent collection.

				If COLLECTION begins with a forward slash, the path is treated as an absolute
				path.

				Upon success, the current working collection is stored in
				irods_environment.json.PID where PID matches the shell's PID number. This
				allows multiple terminal sessions to exist within the same environment.

				If the inclusion of the PID isn't sufficient, this behavior can be overridden
				by setting the environment variable, IRODS_ENVIRONMENT_FILE, to the absolute
				path of a file that will serve as the new session file. This may require
				re-running iinit. Setting the environment variable causes icd to replace the
				".PID" extension with ".cwd".

				Options:
				 -v  Verbose.
				 -V  Very verbose.
				 -h  Show this message.

			*/
			{
				Name:  "icd",
				Usage: "Change the current working collection.",
				Action: func(ctx context.Context, cmd *cli.Command) error {

					cwd, err := resolveCwd()

					if err != nil {
						logger.Error("error resolving cwd", err.Error())
						fmt.Fprintf(cmd.ErrWriter, "error resolving cwd\n")
					}

					filesystem, _, err := obtainFilesystem(cmd)

					defer filesystem.Release()

					entries, err := filesystem.List(cwd)
					if err != nil {
						logger.Error("error listing filesystem", err.Error())
						fmt.Fprintf(cmd.ErrWriter, "error listing filesystem\n")
					}

					if len(entries) == 0 {
						fmt.Printf("\n")
					} else {
						for _, entry := range entries {
							if entry.Type == fs.FileEntry {
								fmt.Printf("%s\n", entry.Name)
							} else {
								color.Blue(entry.Name)
							}

						}
					}

					return nil

				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		logger.Error("error:%v", err.Error())
	}
}

// Code to obtain login information from the environment and connect to irods
// it is incumbent on the caller to close the underlying filesystem
func obtainFilesystem(cmd *cli.Command) (*fs.FileSystem, *types.IRODSAccount, error) {

	irodsAccount, err := obtainIrodsAccount()

	if err != nil {
		logger.Error("error getting irodsAccount out of environment", err.Error())
		fmt.Fprintf(cmd.ErrWriter, "error saving iRODS environment\n")
	}

	filesystem, err := fs.NewFileSystemWithDefault(irodsAccount, APP_NAME)
	return filesystem, irodsAccount, err
}

// Load the env Manager
func obtainIrodsAccount() (*types.IRODSAccount, error) {

	irodsAccount, err := envManager.ToIRODSAccount()

	if err != nil {
		logger.Error("error obtaining iRODS account from manager", err.Error())
		return nil, err
	}

	processId, err = obtainCurrentProcessId()
	
	logger.Info()

	return irodsAccount, nil

}

// get the current process id
func obtainCurrentProcessId() (int, error) {
	processId := os.Getpid()
	logger.Info(fmt.Sprintf("current process id: %d", process_id))
	return processId, nil
}
