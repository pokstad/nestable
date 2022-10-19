package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pokstad/nestable/orm"
)

const (
	timestampLayout = "2006-01-02 15:04:05"
)

var (
	// common flags for all subcommands
	nestPath = flag.String("nest", "", "path to nest file")
)

type subCmdFunc func(ctx context.Context, r io.Reader, w io.Writer, args []string) error

type subCmd interface {
	FlagSet() *flag.FlagSet
	Run(context.Context, io.Reader, io.Writer) error
	Help() string
	Names() []string
}

// cmdFactory generalizes the constructor pattern for subcommands
type cmdFactory func(repo orm.Repo) subCmd

var subCmdFactories = []cmdFactory{
	newInitCmd,
	newNewCmd,
	newEditCmd,
	newViewCmd,
	newBrowseCmd,
	newGetConfigCmd,
	newSetConfigCmd,
}

var subCmdsLookup = func() map[string]cmdFactory {
	l := map[string]cmdFactory{}
	for _, scf := range subCmdFactories {
		sc := scf(orm.Repo{})
		for _, n := range sc.Names() {
			l[n] = scf
		}
	}
	return l
}()

func showHelp() {
	fmt.Println("Provide a valid subcommand:\n")
	for _, scf := range subCmdFactories {
		sc := scf(orm.Repo{})
		fmt.Printf("%20s", strings.Join(sc.Names(), " or "))
		fmt.Printf("\t%s\n", sc.Help())
	}
}

func main() {
	flag.Parse()

	repo, err := orm.LoadRepo(*nestPath)
	if err != nil {
		log.Fatalf("unable to load nest: %w", err)
	}

	subArgStart := len(os.Args) - flag.NArg()

	subArgs := os.Args[subArgStart:]
	if len(subArgs) < 1 {
		showHelp()
		return
	}

	sub, subArgs := subArgs[0], subArgs[1:] // pop subcommand off

	scf, ok := subCmdsLookup[sub]
	if !ok {
		log.Fatalf("unknown command: %q", sub)
	}

	sc := scf(repo) // create subcommand with

	if err := sc.FlagSet().Parse(subArgs); err != nil {
		log.Fatal(err)
	}

	if err := sc.Run(context.Background(), os.Stdin, os.Stdout); err != nil {
		log.Fatal(err.Error())
	}

}
