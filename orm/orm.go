//go:build sqlite_fts5

package orm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
)

const (
	NestName = ".notebook.nest"
)

type Repo struct {
	db *sql.DB
}

var clock = time.Now

// SetClock allows the normal time function to be overriden.
// This time function is used to derive the note timestamps.
// Intended to be used during tests to provide deterministic time.
func SetClock(c func() time.Time) { clock = c }

func LoadRepo(dbPath string) (Repo, error) {
	if dbPath == "" {
		for _, p := range []string{
			path.Join([]string{os.Getenv("HOME"), NestName}...),
			path.Join([]string{".", NestName}...),
		} {
			if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
				continue
			}
			dbPath = p
		}
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return Repo{}, err
	}

	repo := Repo{db: db}
	if err := repo.MigrateUp(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return Repo{}, fmt.Errorf("migrating up existing repo: %w", err)
	}

	return Repo{db}, nil
}

//go:embed migrations/*.sql
var migrationFS embed.FS

func InitRepo(repoPath string) (Repo, error) {
	db, err := sql.Open("sqlite3", repoPath)
	if err != nil {
		return Repo{}, fmt.Errorf("opening DB for initialization: %w", err)
	}

	repo := Repo{db: db}
	if err := repo.MigrateUp(); err != nil {
		return Repo{}, fmt.Errorf("migrating repo up: %w", err)
	}

	return repo, nil
}

func (r Repo) MigrateUp() error {
	migrations, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("loading embedded migrations: %w", err)
	}

	driver, err := sqlite3.WithInstance(r.db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		migrations,
		"migrations",
		driver,
	)
	if err != nil {
		return fmt.Errorf("migrate instance: %w", err)
	}
	return m.Up()

}

type ConfigKey string

const (
	ConfigEditor  ConfigKey = "editor"
	ConfigVersion ConfigKey = "version"
)

func (r Repo) GetConfig(ctx context.Context, key ConfigKey) (string, error) {
	row := r.db.QueryRowContext(ctx, "SELECT value FROM config WHERE key = (?)", key)

	var value string
	if err := row.Scan(&value); err != nil {
		return "", fmt.Errorf("getting config for key %q: %w", key, err)
	}

	return value, nil
}

func (r Repo) GetConfigKeys(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT key FROM config")
	if err != nil {
		return nil, fmt.Errorf("getting config keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("config key result: %w", err)
		}
		keys = append(keys, k)
	}

	return keys, nil
}

func (r Repo) SetConfig(ctx context.Context, key ConfigKey, value string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE CONFIG SET value = (?) WHERE key = (?)", value, key)
	if err != nil {
		return fmt.Errorf("setting config for key %q: %w", key, err)
	}
	return nil
}

type Note struct {
	ID int64
}

type Blob struct {
	SHA256 string
}

type NoteRev struct {
	Note
	Blob
	Timestamp time.Time
}

// Notes implements a subset of the sort.Interface
type Notes []NoteRev

func (n Notes) Len() int      { return len(n) }
func (n Notes) Swap(i, j int) { n[i], n[j] = n[j], n[i] }

// ByID allows Notes to be sorted by ID
type ByID struct{ Notes }

func (bi ByID) Less(i, j int) bool { return bi.Notes[i].ID < bi.Notes[j].ID }

func (nr NoteRev) UpdateBlob(ctx context.Context, r Repo, src io.Reader) (NoteRev, error) {
	h := sha256.New()
	src = io.TeeReader(src, h)

	// TODO: change reading of blob into memory to stream based I/O once golang sqlite supports blob I/O
	blob, err := ioutil.ReadAll(src)
	if err != nil {
		return NoteRev{}, fmt.Errorf("reading blob: %w", err)
	}

	sum := hex.EncodeToString(h.Sum(nil))

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return NoteRev{}, fmt.Errorf("starting edit note tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO blob (body, sha256) VALUES (?, ?)", blob, sum)
	if err != nil {
		return NoteRev{}, fmt.Errorf("inserting new blob: %w", err)
	}

	timestamp := clock()
	_, err = tx.ExecContext(ctx, "INSERT INTO note_rev(note_id, blob_sha256, timestamp) VALUES(?,?,?)", nr.ID, sum, timestamp.UTC())
	if err != nil {
		return NoteRev{}, fmt.Errorf("inserting new note rev: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return NoteRev{}, fmt.Errorf("commiting new note tx: %w", err)
	}

	return NoteRev{
		Note: Note{
			nr.ID,
		},
		Blob: Blob{
			SHA256: sum,
		},
		Timestamp: timestamp,
	}, nil
}

// GetBlobHead returns an excerpt from the front of the blob limited by the specified length
// from the provided repo
// TODO: once streaming blob IO is available, change behavior so that head scans until the first
// newline or the limit, which ever is encountered first
func (b Blob) GetBlobHead(ctx context.Context, r Repo, length int) ([]byte, error) {
	row := r.db.QueryRowContext(ctx, "SELECT substr(body, 1, ?) FROM blob WHERE sha256 = (?)", length, b.SHA256)
	var head []byte
	if err := row.Scan(&head); err != nil {
		return nil, fmt.Errorf("fetching blob head: %w", err)
	}
	return bytes.Split(head, []byte("\n"))[0], nil
}

func (b Blob) GetReader(ctx context.Context, r Repo) (io.Reader, error) {
	row := r.db.QueryRowContext(ctx, "SELECT body FROM blob WHERE sha256 = (?)", b.SHA256)
	var body []byte
	if err := row.Scan(&body); err != nil {
		return nil, fmt.Errorf("fetching blob: %w", err)
	}
	return bytes.NewReader(body), nil
}

func (r Repo) NewNote(ctx context.Context, src io.Reader) (NoteRev, error) {
	h := sha256.New()
	src = io.TeeReader(src, h)

	// TODO: change reading of blob into memory to stream based I/O once golang sqlite supports blob I/O
	blob, err := ioutil.ReadAll(src)
	if err != nil {
		return NoteRev{}, fmt.Errorf("reading blob: %w", err)
	}

	sum := hex.EncodeToString(h.Sum(nil))

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return NoteRev{}, fmt.Errorf("starting new note tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO blob (body, sha256) VALUES (?, ?)", blob, sum)
	if err != nil {
		return NoteRev{}, fmt.Errorf("inserting new blob: %w", err)
	}

	result, err := tx.ExecContext(ctx, "INSERT INTO note DEFAULT VALUES")
	if err != nil {
		return NoteRev{}, fmt.Errorf("inserting new note: %w", err)
	}

	noteID, err := result.LastInsertId()
	if err != nil {
		return NoteRev{}, fmt.Errorf("new note ID: %w", err)
	}

	timestamp := clock()
	_, err = tx.ExecContext(ctx, "INSERT INTO note_rev(note_id, blob_sha256, timestamp) VALUES(?,?,?)", noteID, sum, timestamp)
	if err != nil {
		return NoteRev{}, fmt.Errorf("inserting new note rev: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return NoteRev{}, fmt.Errorf("commiting new note tx: %w", err)
	}

	return NoteRev{
		Note: Note{
			ID: noteID,
		},
		Blob: Blob{
			SHA256: sum,
		},
		Timestamp: timestamp,
	}, nil
}

func (r Repo) GetCurrentNoteRev(ctx context.Context, id int64) (NoteRev, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT note_id, blob_sha256, timestamp, MAX(rowid) 
		FROM note_rev 
		WHERE note_id = (?)`, id)

	var nr NoteRev
	var rowid int
	if err := row.Scan(&nr.ID, &nr.SHA256, &nr.Timestamp, &rowid); err != nil {
		return NoteRev{}, fmt.Errorf("querying notes: %w", err)
	}

	return nr, nil
}

// GetNotes returns all current note revisions
func (r Repo) GetNotes(ctx context.Context) ([]NoteRev, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT note_id, blob_sha256, timestamp, MAX(rowid) 
		FROM note_rev 
		GROUP BY note_id
		ORDER BY timestamp DESC`)
	if err != nil {
		return nil, fmt.Errorf("querying notes: %w", err)
	}
	defer rows.Close()

	var revs []NoteRev

	for rows.Next() {
		var r NoteRev
		var rowid int64
		if err := rows.Scan(&r.ID, &r.SHA256, &r.Timestamp, &rowid); err != nil {
			return nil, fmt.Errorf("scanning note summaries reults: %w", err)
		}
		r.Timestamp = r.Timestamp.Local()
		revs = append(revs, r)
	}

	return revs, nil
}

// FTSResult is the result of a full text search of the blob table
type FTSResult struct {
	noteRevRowID int64
	SHA256       string
	BM25         float32
	Snippet      string
}

func (ftsr FTSResult) GetNoteRev(ctx context.Context, repo Repo) (NoteRev, error) {
	row := repo.db.QueryRowContext(ctx,
		`SELECT note_id, blob_sha256, timestamp 
		FROM note_rev 
		WHERE rowid = (?)`,
		ftsr.noteRevRowID)
	var nr NoteRev
	if err := row.Scan(&nr.ID, &nr.SHA256, &nr.Timestamp); err != nil {
		return NoteRev{}, fmt.Errorf("scanning blob fts reults: %w", err)
	}

	nr.Timestamp = nr.Timestamp.Local()
	return nr, nil
}

func (r Repo) FullTextSearch(ctx context.Context, searchTerm string) ([]FTSResult, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT
			note_rev_rowid,
			blob_sha256,
			bm25(note_fts, 0, 1.0),
			snippet(note_fts, -1, "👉 ", " 👈", "...", 20)
		FROM note_fts
		WHERE blob_body MATCH (?)
		ORDER BY bm25(note_fts, 0, 1.0);`,
		searchTerm)
	if err != nil {
		return nil, fmt.Errorf("querying notes: %w", err)
	}
	defer rows.Close()

	var results []FTSResult

	for rows.Next() {
		var b FTSResult
		if err := rows.Scan(&b.noteRevRowID, &b.SHA256, &b.BM25, &b.Snippet); err != nil {
			return nil, fmt.Errorf("scanning blob fts reults: %w", err)
		}
		results = append(results, b)
	}

	return results, nil
}

// WCTerm is a term in the word cloud
type WCTerm struct {
	Term          string
	NoteCount     int64
	InstanceCount int64
}

// WordCloudTerms returns all search terms in the word cloud that are not stop words
func (r Repo) WordCloudTerms(ctx context.Context) ([]WCTerm, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT
			term,
			doc,
			cnt
		FROM note_fts_vocab_cols
		WHERE col = "blob_body"
		ORDER BY cnt DESC`)
	if err != nil {
		return nil, fmt.Errorf("querying word cloud terms: %w", err)
	}
	defer rows.Close()

	var results []WCTerm

	for rows.Next() {
		var wc WCTerm
		if err := rows.Scan(&wc.Term, &wc.NoteCount, &wc.InstanceCount); err != nil {
			return nil, fmt.Errorf("scanning word cloud term reults: %w", err)
		}
		results = append(results, wc)
	}

	return results, nil
}

// Instances returns all the places the term occurs
func (wct WCTerm) Instances(ctx context.Context, repo Repo) ([]NoteRev, error) {
	rows, err := repo.db.QueryContext(ctx,
		`SELECT DISTINCT note_rev.note_id, note_rev.blob_sha256, note_rev.timestamp
		FROM note_fts_vocab_instances
		INNER JOIN note_fts
			ON note_fts_vocab_instances.doc = note_fts.rowid
		INNER JOIN note_rev
			ON note_fts.note_rev_rowid = note_rev.rowid
		WHERE term = (?)
		AND col = 'blob_body'`,
		wct.Term)
	if err != nil {
		return nil, fmt.Errorf("querying term instances for %q: %w", wct.Term, err)
	}
	defer rows.Close()

	var results []NoteRev

	for rows.Next() {
		var nr NoteRev
		if err := rows.Scan(&nr.ID, &nr.SHA256, &nr.Timestamp); err != nil {
			return nil, fmt.Errorf("scanning blob fts reults: %w", err)
		}
		nr.Timestamp = nr.Timestamp.Local()
		results = append(results, nr)
	}

	return results, nil
}
