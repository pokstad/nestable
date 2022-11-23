package exporter_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"

	"github.com/pokstad/nestable/internal/exporter"
	"github.com/pokstad/nestable/internal/ormtest"
	"github.com/stretchr/testify/require"
)

func TestExportMarkdown(t *testing.T) {
	cleanupClock := ormtest.MockClock()
	defer cleanupClock()

	repo, cleanupRepo := ormtest.TempTestRepo(t)
	defer cleanupRepo()

	notes := []string{
		"header\n\nparagraph1\n\nparagraph2",
		"header2\n\nparagraph1\n\nparagraph2\n\nparagraph3",
	}

	ctx := context.Background()
	_ = ormtest.InsertTestNotes(t, ctx, repo, notes)

	expectMD, err := ioutil.ReadFile("testdata/markdown_export.md")
	require.NoError(t, err)

	buf := bytes.NewBuffer(nil)
	require.NoError(t, exporter.ExportMarkdown(ctx, repo, buf))

	require.Equal(t, string(expectMD), buf.String())
}
