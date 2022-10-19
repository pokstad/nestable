package main

import (
	"context"
	"flag"
	"io"
	"os"
	"path"

	"github.com/pokstad/nestable/orm"
)

type initCmd struct {
	path *string
}

func newInitCmd(_ orm.Repo) subCmd {
	return &initCmd{}
}

func (_ *initCmd) Help() string {
	return `Initialize a new nestable database.`
}

func (_ *initCmd) Names() []string {
	return []string{"init", "i"}
}

func (ic *initCmd) FlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	ic.path = fs.String("path", path.Join(os.Getenv("HOME"), orm.NestName), "path to initialize nest file")
	return fs
}

func (ic *initCmd) Run(ctx context.Context, r io.Reader, w io.Writer) error {
	return orm.InitRepo(*ic.path)
}
