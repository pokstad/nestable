package main

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/charmbracelet/glamour"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/pokstad/nestable/orm"
)

func selectNoteRev(ctx context.Context, repo orm.Repo, header string) (orm.NoteRev, error) {
	notes, err := repo.GetNotes(ctx)
	if err != nil {
		return orm.NoteRev{}, fmt.Errorf("listing notes: %w", err)
	}

	idx, err := fuzzyfinder.Find(notes,
		func(i int) string {
			head, err := notes[i].GetBlobHead(ctx, repo, 80)
			if err != nil {
				panic(err)
			}
			return fmt.Sprintf(
				"%s [%d] %s",
				notes[i].Timestamp.Local().Format(timestampLayout),
				notes[i].ID,
				string(head),
			)
		},
		fuzzyfinder.WithHeader(header),
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}

			selected := notes[i]

			bReader, err := selected.GetReader(ctx, repo)
			if err != nil {
				panic(err)
			}

			raw, err := ioutil.ReadAll(bReader)
			if err != nil {
				panic(err)
			}

			md, err := glamour.RenderBytes(raw, "ascii")
			if err != nil {
				panic(err)
			}

			return string(md)
		}),
	)
	if err != nil {
		return orm.NoteRev{}, fmt.Errorf("fuzzy find ids: %w", err)
	}

	return notes[idx], nil
}

func selectFTSResults(ctx context.Context, repo orm.Repo, results []orm.FTSResult) (orm.NoteRev, error) {
	idx, err := fuzzyfinder.Find(results,
		func(i int) string {
			return results[i].Snippet
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			nr, err := results[i].GetNoteRev(ctx, repo)
			if err != nil {
				panic(err)
			}
			reader, err := nr.GetReader(ctx, repo)
			if err != nil {
				panic(err)
			}

			raw, err := ioutil.ReadAll(reader)
			if err != nil {
				panic(err)
			}

			md, err := glamour.RenderBytes(raw, "ascii")
			if err != nil {
				panic(err)
			}

			return string(md)
		}),
	)

	if err != nil {
		return orm.NoteRev{}, fmt.Errorf("fuzzy finding search results: %w", err)
	}

	nr, err := results[idx].GetNoteRev(ctx, repo)
	if err != nil {
		return orm.NoteRev{}, fmt.Errorf("fetch note rev for search result: %w", err)
	}

	return nr, nil
}
