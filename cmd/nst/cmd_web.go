package main

import (
	"context"
	"flag"
	"io"

	"github.com/pokstad/nestable/internal/web"
	"github.com/pokstad/nestable/orm"
)

type webCmd struct {
	repo   orm.Repo
	search *string
}

func newWebCmd(repo orm.Repo) subCmd {
	return &webCmd{repo: repo}
}

func (_ *webCmd) Help() string {
	return `View notes in web UI in default browser.`
}

func (_ *webCmd) Names() []string {
	return []string{"web", "w"}
}

func (vc *webCmd) FlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("web", flag.ExitOnError)
	return fs
}

func (vc *webCmd) Run(ctx context.Context, r io.Reader, w io.Writer) error {
	return web.Serve(ctx, vc.repo, r, w)
}
