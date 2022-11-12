-- Create virtual table for full text search of notes
CREATE VIRTUAL TABLE note_fts
	USING FTS5(
		note_rev_rowid,
		blob_sha256,
		blob_body
	);

-- one time mass inserting of FTS records for all current note revisions
INSERT INTO note_fts (note_rev_rowid, blob_sha256, blob_body)
	SELECT
		MAX(nr.rowid) note_rev_rowid,
		blob.sha256 blob_sha256,
		blob.body blob_body
	FROM note_rev AS nr
	INNER JOIN blob ON nr.blob_sha256 = blob.sha256
	GROUP BY nr.note_id;

-- removes stale FTS entries before new row is inserted
CREATE TRIGGER remove_old_note_fts BEFORE INSERT ON note_rev BEGIN
	DELETE FROM note_fts
	WHERE note_rev_rowid = (
		SELECT rowid
		FROM note_rev
		WHERE note_id = new.note_id
		GROUP BY note_id
		HAVING MAX(rowid)
	);
END;

-- inserts FTS entry for new note revision
CREATE TRIGGER insert_note_fts AFTER INSERT ON note_rev BEGIN
	INSERT INTO note_fts(note_rev_rowid, blob_sha256,blob_body)
		SELECT new.rowid, sha256, body
		FROM blob
		WHERE sha256 = new.blob_sha256;
END;
