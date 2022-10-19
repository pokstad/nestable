package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/pokstad/nestable/orm"
)

var editorOptions = map[string][]string{
	"mate": []string{"--wait"},
}

func runEditor(ctx context.Context, repo orm.Repo, blob, stdin io.Reader, stdout, stderr io.Writer) (io.ReadCloser, error) {
	editor, err := repo.GetConfig(ctx, "editor")
	if err != nil {
		return nil, fmt.Errorf("getting editor config: %w", err)
	}

	edOpts := editorOptions[editor]

	tf, err := ioutil.TempFile("", "nestable-*.md")
	if err != nil {
		return nil, fmt.Errorf("temp file open: %w", err)
	}

	_, err = io.Copy(tf, blob)
	if err != nil {
		return nil, fmt.Errorf("copying blob to temp file: %w", err)
	}

	if err := tf.Close(); err != nil {
		return nil, fmt.Errorf("closing temp file: %w", err)
	}

	edOpts = append(edOpts, tf.Name())

	cmd := exec.Command(editor, edOpts...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("running editor: %w", err)
	}

	// TODO: replace in memory read of blob with io.Reader when sqlite streaming blob I/O lands
	newBlob, err := os.Open(tf.Name())
	if err != nil {
		return nil, fmt.Errorf("opening temp file: %w", err)
	}

	return closeDeleter{newBlob}, nil
}

type closeDeleter struct {
	*os.File
}

func (cd closeDeleter) Close() error {
	defer os.Remove(cd.File.Name())
	return cd.File.Close()
}
