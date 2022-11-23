package main

import (
	"context"
	"flag"
	"io"

	"github.com/pokstad/nestable/internal/exporter"
	"github.com/pokstad/nestable/orm"
)

type exportCmd struct {
	repo orm.Repo
}

func newExportCmd(repo orm.Repo) subCmd {
	return &exportCmd{repo: repo}
}

func (_ *exportCmd) Help() string {
	return `Exports notes to another format.`
}

func (_ *exportCmd) Names() []string {
	return []string{"export", "ex"}
}

func (ec *exportCmd) FlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	return fs
}

func (ec *exportCmd) Run(ctx context.Context, r io.Reader, w io.Writer) error {
	return exporter.ExportMarkdown(ctx, ec.repo, w)
}
