package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/urfave/cli/v3"
)

func init() {
	cli.RootCommandHelpTemplate += "\nCUSTOMIZED: you bet ur muffins\n"
	cli.CommandHelpTemplate += "\nYMMV\n"
	cli.SubcommandHelpTemplate += "\nor something\n"

	cli.HelpFlag = &cli.BoolFlag{Name: "halp"}
	cli.VersionFlag = &cli.BoolFlag{Name: "print-version", Aliases: []string{"V"}}

	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		fmt.Fprintf(w, "best of luck to you\n")
	}
	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Fprintf(cmd.Root().Writer, "version=%s\n", cmd.Root().Version)
	}
	cli.OsExiter = func(cmd int) {
		fmt.Fprintf(cli.ErrWriter, "refusing to exit %d\n", cmd)
	}
	cli.ErrWriter = ioutil.Discard
	cli.FlagStringer = func(fl cli.Flag) string {
		return fmt.Sprintf("\t\t%s", fl.Names()[0])
	}
}

func main() {
	(&cli.Command{}).Run(context.Background(), os.Args)
}
