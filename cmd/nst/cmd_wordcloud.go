package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/charmbracelet/glamour"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/pokstad/nestable/orm"
)

type wordCloudCmd struct {
	repo orm.Repo
}

func newWorkCloudCmd(repo orm.Repo) subCmd {
	return &wordCloudCmd{repo: repo}
}

func (_ *wordCloudCmd) Help() string {
	return `Select a term from the word cloud to see notes it appears in.`
}

func (_ *wordCloudCmd) Names() []string {
	return []string{"word-cloud", "wc"}
}

func (wcc *wordCloudCmd) FlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("wordCloud", flag.ExitOnError)
	return fs
}

func (wcc *wordCloudCmd) Run(ctx context.Context, r io.Reader, w io.Writer) error {
	terms, err := wcc.repo.WordCloudTerms(ctx)
	if err != nil {
		return fmt.Errorf("fetching word cloud: %w", err)
	}

	idx, err := fuzzyfinder.Find(terms, func(i int) string {
		return fmt.Sprintf("%20s - appears %5d in %5d notes",
			terms[i].Term, terms[i].InstanceCount, terms[i].NoteCount)
	})
	if err != nil {
		return fmt.Errorf("selecting word cloud term: %w", err)
	}

	selectedTerm := terms[idx]
	termInstances, err := selectedTerm.Instances(ctx, wcc.repo)
	if err != nil {
		return fmt.Errorf("fetching instances for %q: %w",
			selectedTerm.Term, err)
	}

	idx, err = fuzzyfinder.Find(termInstances, func(i int) string {
		instance := termInstances[i]
		head, err := instance.GetBlobHead(ctx, wcc.repo, 80)
		if err != nil {
			panic(err)
		}

		return fmt.Sprintf("[%s] - %s", instance.Timestamp, head)
	})
	if err != nil {
		return fmt.Errorf("selecting word cloud term: %w", err)
	}

	rev := termInstances[idx]

	bReader, err := rev.GetReader(ctx, wcc.repo)
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
