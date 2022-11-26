package web

import (
	"context"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"

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

		notes, err := repo.GetNotes(ctx)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		wNotes := make([]note, len(notes))
		for i, n := range notes {
			wNotes[i] = note{NoteRev: n}
			head, err := n.GetBlobHead(ctx, repo, 100)
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
			wNotes[i].Header = string(head)
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
