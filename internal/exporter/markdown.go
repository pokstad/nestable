package exporter

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"sort"

	"github.com/pokstad/nestable/orm"
)

var mdOnePageTmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"headerFor": func(ctx context.Context, r orm.Repo, n orm.NoteRev) (string, error) {
		head, err := n.GetBlobHead(ctx, r, 80)
		return string(head), err
	},
	"bodyFor": func(ctx context.Context, r orm.Repo, n orm.NoteRev) (string, error) {
		reader, err := n.GetReader(ctx, r)
		if err != nil {
			return "", err
		}

		body, err := ioutil.ReadAll(reader)
		if err != nil {
			return "", err
		}

		return string(body), nil
	},
}).Parse(`# My Nestable Notes
{{ $root := . }}
**Table of Contents**
{{range .notes}}
- <a href="#{{ .ID }}">[{{ .ID }}] {{ headerFor $root.ctx $root.repo . }}</a>{{end}}
{{range .notes}}
### <a name="{{ .ID }}">[{{ .ID }}] {{ headerFor $root.ctx $root.repo . }}</a>

{{ bodyFor $root.ctx $root.repo . }}
{{end}}`,
))

func ExportMarkdown(ctx context.Context, repo orm.Repo, w io.Writer) error {
	revs, err := repo.GetNotes(ctx)
	if err != nil {
		return fmt.Errorf("getting all note: %w", err)
	}

	sort.Sort(orm.ByID{Notes: revs})

	return mdOnePageTmpl.Execute(w, map[string]any{
		"notes": revs,
		"ctx":   ctx,
		"repo":  repo,
	})
}
