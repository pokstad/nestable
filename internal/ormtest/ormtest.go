package ormtest

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/pokstad/nestable/orm"
	"github.com/stretchr/testify/require"
)

// Cleanup should be invoked from a defer statement to cleanup resources
type Cleanup func()

// InsertTestNotes allows you to quickly insert notes as a list of strings
func InsertTestNotes(t *testing.T, ctx context.Context, repo orm.Repo, notes []string) []orm.NoteRev {
	var revs []orm.NoteRev
	for _, n := range notes {
		rev, err := repo.NewNote(ctx, bytes.NewBufferString(n))
		require.NoError(t, err)
		revs = append(revs, rev)
	}
	return revs
}

// AssertNoteReader ensures that the body of the note matches the expected body
func AssertNoteReader(t *testing.T, ctx context.Context, repo orm.Repo, note orm.NoteRev, expectBody []byte) {
	r, err := note.GetReader(ctx, repo)
	require.NoError(t, err)

	actualBody, err := ioutil.ReadAll(r)
	require.NoError(t, err)

	require.Equal(t, expectBody, actualBody)
}

// TempTestRepo creates a temporary file to store a new repo for testing
// Calling the Cleanup function will delete the temp file
func TempTestRepo(t *testing.T) (orm.Repo, Cleanup) {
	temp, err := ioutil.TempFile("", "test-nestable-*.nest")
	require.NoError(t, err)
	require.NoError(t, temp.Close())

	repo, err := orm.InitRepo(temp.Name())
	require.NoError(t, err)

	return repo, func() { os.Remove(temp.Name()) }
}

// MockClock makes time more deterministic in tests.
// Each time mockClock is called, the UNIX time is
// incremented by one second.
func MockClock() Cleanup {
	t := time.Unix(0, 0)
	orm.SetClock(func() time.Time {
		t = t.Add(time.Second)
		return t
	})
	return func() { orm.SetClock(time.Now) }
}
