package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/pokstad/nestable/orm"
)

type newCmd struct {
	repo orm.Repo
	msg  *string
}

func newNewCmd(repo orm.Repo) subCmd {
	return &newCmd{repo: repo}
}

func (_ *newCmd) Help() string {
	return `Create a new note with your editor of choice.`
}

func (_ *newCmd) Names() []string {
	return []string{"new", "n"}
}

func (nc *newCmd) FlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	nc.msg = fs.String("m", "", "Provide note as an arg without invoking external editor")
	return fs
}

func (nc *newCmd) Run(ctx context.Context, r io.Reader, w io.Writer) error {
	var blob io.Reader = bytes.NewBufferString(*nc.msg)

	if *nc.msg == "" {
		blob, err := runEditor(ctx, nc.repo, bytes.NewReader(nil), r, w, os.Stderr)
		if err != nil {
			return fmt.Errorf("run external editor: %w", err)
		}
		defer blob.Close()
	}

	nr, err := nc.repo.NewNote(ctx, blob)
	if err != nil {
		return fmt.Errorf("new note in repo: %w", err)
	}

	_, err = fmt.Fprintln(w, nr.SHA256)
	return err
}
