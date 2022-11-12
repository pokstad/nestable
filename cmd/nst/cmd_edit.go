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
	search *string
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
	ec.search = fs.String("s", "", "full text search term to filter results")
	return fs
}

func (ec *editCmd) Run(ctx context.Context, r io.Reader, w io.Writer) error {
	var (
		rev orm.NoteRev
		err error
	)

	if *ec.noteID != 0 {
		rev, err = ec.repo.GetCurrentNoteRev(ctx, *ec.noteID)
		if err != nil {
			return fmt.Errorf("getting current note rev for ID %d: %w", *ec.noteID, err)
		}
	}

	if *ec.search != "" && rev == (orm.NoteRev{}) {
		results, err := ec.repo.FullTextSearch(ctx, *ec.search)
		if err != nil {
			return fmt.Errorf("full text search with term %q: %w", *ec.search, err)
		}

		rev, err = selectFTSResults(ctx, ec.repo, results)
		if err != nil {
			return fmt.Errorf("selecting search results: %w", err)
		}
	}

	if rev == (orm.NoteRev{}) {
		rev, err = selectNoteRev(ctx, ec.repo, "Select a note to edit")
		if err != nil {
			return fmt.Errorf("selecting note rev for edit: %w", err)
		}
	}

	if rev == (orm.NoteRev{}) {

	}

	blobReader, err := rev.GetReader(ctx, ec.repo)
	if err != nil {
		return fmt.Errorf("get reader for rev pick: %w", err)
	}

	newBlob, err := runEditor(ctx, ec.repo, blobReader, r, w, os.Stderr)
	if err != nil {
		return fmt.Errorf("run external editor: %w", err)
	}
	defer newBlob.Close()

	newRev, err := rev.UpdateBlob(ctx, ec.repo, newBlob)
	if err != nil {
		return fmt.Errorf("updating rev blob: %w", err)
	}

	_, err = fmt.Fprintln(w, newRev.SHA256)
	return err
}
