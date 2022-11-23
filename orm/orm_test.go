package orm_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/pokstad/nestable/internal/ormtest"
	"github.com/pokstad/nestable/orm"
	"github.com/stretchr/testify/require"
)

func TestRepoConfig(t *testing.T) {
	repo, cleanup := ormtest.TempTestRepo(t)
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
	clockCleanup := ormtest.MockClock()
	defer clockCleanup()

	repo, cleanup := ormtest.TempTestRepo(t)
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
	ormtest.AssertNoteReader(t, ctx, repo, note1, []byte(noteBody1))

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
	ormtest.AssertNoteReader(t, ctx, repo, newRev, []byte(noteBody2))
	note1 = newRev

	curRev, err := repo.GetCurrentNoteRev(ctx, note1.ID)
	require.NoError(t, err)
	require.Equal(t, note1.ID, curRev.ID)
	require.Equal(t, note1.SHA256, curRev.SHA256)
	ormtest.AssertNoteReader(t, ctx, repo, curRev, []byte(noteBody2))

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
	ormtest.AssertNoteReader(t, ctx, repo, note2, []byte(noteBody3))

	notes, err := repo.GetNotes(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []orm.NoteRev{expectNote1Rev2, expectNote2Rev1}, notes)
}

func TestBlobFTS(t *testing.T) {
	clockCleanup := ormtest.MockClock()
	defer clockCleanup()

	repo, cleanup := ormtest.TempTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	notes := []string{
		"this is a full text note for a nestable notebook",
		"once a note, always a note",
		"nestable allows you to map your mind",
	}
	revs := ormtest.InsertTestNotes(t, ctx, repo, notes)

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

func TestWordCloud(t *testing.T) {
	clockCleanup := ormtest.MockClock()
	defer clockCleanup()

	repo, cleanup := ormtest.TempTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	notes := []string{
		"a aa aaa",
		"a b c",
		"a b bc",
	}
	revs := ormtest.InsertTestNotes(t, ctx, repo, notes)

	// Fetch word cloud
	terms, err := repo.WordCloudTerms(ctx)
	require.NoError(t, err)
	expectTerms := []orm.WCTerm{
		orm.WCTerm{Term: "aa", NoteCount: 1, InstanceCount: 1},
		orm.WCTerm{Term: "aaa", NoteCount: 1, InstanceCount: 1},
		orm.WCTerm{Term: "bc", NoteCount: 1, InstanceCount: 1},
		orm.WCTerm{Term: "c", NoteCount: 1, InstanceCount: 1},
		orm.WCTerm{Term: "b", NoteCount: 2, InstanceCount: 2},
		orm.WCTerm{Term: "a", NoteCount: 3, InstanceCount: 3},
	}
	require.ElementsMatch(t, expectTerms, terms)

	instanceRevs, err := expectTerms[5].Instances(ctx, repo)
	require.NoError(t, err)
	t.Log(instanceRevs)
	require.Equal(t, revs, instanceRevs)
}
