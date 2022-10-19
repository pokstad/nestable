package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/pokstad/nestable/orm"
)

type editCmd struct {
	repo   orm.Repo
	noteID *int64
}

func newEditCmd(repo orm.Repo) subCmd {
	return &editCmd{repo: repo}
}

func (_ *editCmd) Help() string {
	return `Edit a note with your editor of choice. Specify a note or pick one from an interactive list.`
}

func (_ *editCmd) Names() []string {
	return []string{"edit", "e"}
}

func (ec *editCmd) FlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("edit", flag.ExitOnError)
	ec.noteID = fs.Int64("id", 0, "note ID you want to edit")
	return fs
}

func (ec *editCmd) Run(ctx context.Context, r io.Reader, w io.Writer) error {
	var revPick orm.NoteRev

	var err error
	if *ec.noteID == 0 {
		revPick, err = selectNoteRev(ctx, ec.repo, "Select a note to edit")
		if err != nil {
			return fmt.Errorf("selecting note rev for edit: %w", err)
		}
	}

	if revPick == (orm.NoteRev{}) {
		revPick, err = ec.repo.GetCurrentNoteRev(ctx, *ec.noteID)
		if err != nil {
			return fmt.Errorf("getting current note rev for ID %d: %w", *ec.noteID, err)
		}
	}

	blobReader, err := revPick.GetReader(ctx, ec.repo)
	if err != nil {
		return fmt.Errorf("get reader for rev pick: %w", err)
	}

	newBlob, err := runEditor(ctx, ec.repo, blobReader, r, w, os.Stderr)
	if err != nil {
		return fmt.Errorf("run external editor: %w", err)
	}
	defer newBlob.Close()

	newRev, err := revPick.UpdateBlob(ctx, ec.repo, newBlob)
	if err != nil {
		return fmt.Errorf("updating rev blob: %w", err)
	}

	_, err = fmt.Fprintln(w, newRev.SHA256)
	return err
}
