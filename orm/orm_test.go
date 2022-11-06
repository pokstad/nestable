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

	err = orm.InitRepo(temp.Name())
	require.NoError(t, err)

	repo, err := orm.LoadRepo(temp.Name())
	require.NoError(t, err)

	return repo, func() { os.Remove(temp.Name()) }
}

func mockClock() cleanup {
	t := time.Unix(0, 0)
	orm.SetClock(func() time.Time {
		t = t.Add(time.Second)
		return t
	})
	return func() { orm.SetClock(time.Now) }
}
