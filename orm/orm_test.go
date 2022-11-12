package orm_test

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

// cleanup should be invoked from a defer statement to cleanup resources
type cleanup func()

func TestRepoConfig(t *testing.T) {
	repo, cleanup := getTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	editorVal, err := repo.GetConfig(ctx, orm.ConfigEditor)
	require.NoError(t, err)
	require.Equal(t, "vi", editorVal)

	err = repo.SetConfig(ctx, orm.ConfigEditor, "emacs")
	require.NoError(t, err)

	editorVal, err = repo.GetConfig(ctx, orm.ConfigEditor)
	require.NoError(t, err)
	require.Equal(t, "emacs", editorVal)

}

func TestRepoNote(t *testing.T) {
	clockCleanup := mockClock()
	defer clockCleanup()

	repo, cleanup := getTestRepo(t)
	defer cleanup()

	noteBody1 := "hey kid, i'm a note"

	expectNote1Rev1 := orm.NoteRev{
		Note: orm.Note{
			ID: 1,
		},
		Blob: orm.Blob{
			SHA256: "7bc7fd3d3933c999bbc14bb34f8f0221fa0d9076f9e37be0899780b17b88fd13",
		},
		Timestamp: time.Unix(1, 0),
	}

	ctx := context.Background()

	note1, err := repo.NewNote(ctx, bytes.NewBufferString(noteBody1))
	require.NoError(t, err)
	require.Equal(t, expectNote1Rev1, note1)
	assertNoteReader(t, ctx, repo, note1, []byte(noteBody1))

	head, err := note1.GetBlobHead(ctx, repo, 3)
	require.NoError(t, err)
	require.Equal(t, noteBody1[0:3], string(head))

	noteBody2 := "put your notes inside my chest"

	expectNote1Rev2 := orm.NoteRev{
		Note: orm.Note{
			ID: 1,
		},
		Blob: orm.Blob{
			SHA256: "87e4f11a1b9fd1fb6aad528c375fb43ead8963aa6b337797b3f3aaff0391cf26",
		},
		Timestamp: time.Unix(2, 0),
	}

	newRev, err := note1.UpdateBlob(ctx, repo, bytes.NewBufferString(noteBody2))
	require.NoError(t, err)
	require.Equal(t, expectNote1Rev2, newRev)
	assertNoteReader(t, ctx, repo, newRev, []byte(noteBody2))
	note1 = newRev

	curRev, err := repo.GetCurrentNoteRev(ctx, note1.ID)
	require.NoError(t, err)
	require.Equal(t, note1.ID, curRev.ID)
	require.Equal(t, note1.SHA256, curRev.SHA256)
	assertNoteReader(t, ctx, repo, curRev, []byte(noteBody2))

	noteBody3 := "when notes are on a bagel you can have notes anytime"

	expectNote2Rev1 := orm.NoteRev{
		Note: orm.Note{
			ID: 2,
		},
		Blob: orm.Blob{
			SHA256: "a97862377e4e24ecf089c1684e7935908b14a0cf540c5692ecefc205d9970765",
		},
		Timestamp: time.Unix(3, 0),
	}

	note2, err := repo.NewNote(ctx, bytes.NewBufferString(noteBody3))
	require.NoError(t, err)
	require.Equal(t, expectNote2Rev1, note2)
	assertNoteReader(t, ctx, repo, note2, []byte(noteBody3))

	notes, err := repo.GetNotes(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []orm.NoteRev{expectNote1Rev2, expectNote2Rev1}, notes)
}

func TestBlobFTS(t *testing.T) {
	clockCleanup := mockClock()
	defer clockCleanup()

	repo, _ := getTestRepo(t)
	//defer cleanup()

	ctx := context.Background()

	notes := []string{
		"this is a full text note for a nestable notebook",
		"once a note, always a note",
		"nestable allows you to map your mind",
	}
	revs := insertTestNotes(t, ctx, repo, notes)

	results, err := repo.FullTextSearch(ctx, "note")
	require.NoError(t, err)

	// Searching for a token will yield all notes that contain it
	require.Len(t, results, 2)
	require.Equal(t, revs[0].SHA256, results[1].SHA256)
	require.Equal(t, revs[1].SHA256, results[0].SHA256)

	// Updating a note will remove the old version from FTS results
	revs[0], err = revs[0].UpdateBlob(ctx, repo, bytes.NewBufferString("nestable sounds a lot like..."))
	require.NoError(t, err)
	results, err = repo.FullTextSearch(ctx, "note")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, revs[1].SHA256, results[0].SHA256)

	// Fetch a note revision from a result
	nr, err := results[0].GetNoteRev(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, revs[1], nr)
}

func insertTestNotes(t *testing.T, ctx context.Context, repo orm.Repo, notes []string) []orm.NoteRev {
	var revs []orm.NoteRev
	for _, n := range notes {
		rev, err := repo.NewNote(ctx, bytes.NewBufferString(n))
		require.NoError(t, err)
		revs = append(revs, rev)
	}
	return revs
}

func assertNoteReader(t *testing.T, ctx context.Context, repo orm.Repo, note orm.NoteRev, expectBody []byte) {
	r, err := note.GetReader(ctx, repo)
	require.NoError(t, err)

	actualBody, err := ioutil.ReadAll(r)
	require.NoError(t, err)

	require.Equal(t, expectBody, actualBody)
}

func getTestRepo(t *testing.T) (orm.Repo, cleanup) {
	temp, err := ioutil.TempFile("", "test-nestable-*.nest")
	require.NoError(t, err)
	require.NoError(t, temp.Close())
	t.Logf("repo path: %s", temp.Name())

	repo, err := orm.InitRepo(temp.Name())
	require.NoError(t, err)

	return repo, func() { os.Remove(temp.Name()) }
}

// mockClock makes time more deterministic in tests.
// Each time mockClock is called, the UNIX time is
// incremented by one second.
func mockClock() cleanup {
	t := time.Unix(0, 0)
	orm.SetClock(func() time.Time {
		t = t.Add(time.Second)
		return t
	})
	return func() { orm.SetClock(time.Now) }
}
