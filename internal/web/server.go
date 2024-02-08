package web

import (
	"context"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"sort"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/pokstad/nestable/orm"
)

var (
	//go:embed static/*
	staticFS embed.FS
)

type note struct {
	orm.NoteRev
	Header string
}

func Serve(ctx context.Context, repo orm.Repo, r io.Reader, w io.Writer) error {
	srv := &http.Server{Addr: ":3000"}
	defer srv.Close()

	subStaticFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		return err
	}

	http.Handle("/", http.FileServer(http.FS(subStaticFS)))
	http.HandleFunc("/notes", func(rw http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()

		allNotes, err := repo.GetNotes(ctx)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		allHeaders := make([]string, len(allNotes))

		for i, n := range allNotes {
			head, err := n.GetBlobHead(ctx, repo, 100)
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
			allHeaders[i] = string(head)
		}
		wNotes := make([]note, len(allNotes))
		for i, n := range allNotes {
			wNotes[i] = note{NoteRev: n}
			wNotes[i].Header = allHeaders[i]
		}

		fuzzySearch := req.URL.Query().Get("fuzzy")
		if fuzzySearch != "" {
			ranks := fuzzy.RankFindNormalizedFold(fuzzySearch, allHeaders)
			sort.Sort(ranks)
			log.Print(ranks)
		}

		enc := json.NewEncoder(rw)
		if err := enc.Encode(wNotes); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	log.Print("Listening on :3000...")
	errQ := make(chan error)
	go func() { errQ <- srv.ListenAndServe() }()

	select {
	case <-ctx.Done():
		err = srv.Shutdown(ctx)
		break
	case err = <-errQ:
		break
	}

	return err
}
