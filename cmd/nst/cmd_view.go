package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/charmbracelet/glamour"
	"github.com/pokstad/nestable/orm"
)

type viewCmd struct {
	repo orm.Repo
}

func newViewCmd(repo orm.Repo) subCmd {
	return &viewCmd{repo: repo}
}

func (_ *viewCmd) Help() string {
	return `View a specific note, or pick one from an interactive list. By default, renders Markdown content and prints to stdout.`
}

func (_ *viewCmd) Names() []string {
	return []string{"view", "v"}
}

func (vc *viewCmd) FlagSet() *flag.FlagSet {
	return flag.NewFlagSet("view", flag.ExitOnError)
}

func (vc *viewCmd) Run(ctx context.Context, r io.Reader, w io.Writer) error {
	rev, err := selectNoteRev(ctx, vc.repo, "Select a note to view")
	if err != nil {
		return fmt.Errorf("selecting note to view: %w", err)
	}

	bReader, err := rev.GetReader(ctx, vc.repo)
	if err != nil {
		return fmt.Errorf("getting blob reader: %w", err)
	}

	raw, err := ioutil.ReadAll(bReader)
	if err != nil {
		return fmt.Errorf("reading blob: %w", err)
	}

	out, err := glamour.RenderBytes(raw, "ascii")
	if err != nil {
		return fmt.Errorf("rendering blob: %w", err)
	}

	fmt.Fprint(w, string(out))

	return nil
}
