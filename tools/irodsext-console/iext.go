package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/cyverse/go-irodsclient/config"
	"github.com/cyverse/go-irodsclient/types"

	"github.com/urfave/cli/v3"
)

var envManager *config.ICommandsEnvironmentManager

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func init() {
	cli.RootCommandHelpTemplate += "\niRODS Extended Command Tool\n"
	cli.CommandHelpTemplate += "\nYMMV\n"
	cli.SubcommandHelpTemplate += "\nor something\n"

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

}

func main() {

	cmd := &cli.Command{
		Name:  "init",
		Usage: "init user:password@host:port/zone",
		Action: func(context.Context, *cli.Command) error {
			fmt.Println("initializing irods session info")
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		logger.Error("error:%v", err.Error())
	}
}
