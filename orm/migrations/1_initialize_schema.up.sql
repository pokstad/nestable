PRAGMA foreign_keys = ON;

/* note represents a logical note in the application */
CREATE TABLE note (
	id INTEGER PRIMARY KEY
);

/* note_rev is a revision of a note */
CREATE TABLE note_rev (
	note_id INTEGER NOT NULL,
	blob_sha256 VARCHAR(64) NOT NULL,
	timestamp DATETIME NOT NULL,
	
	FOREIGN KEY (note_id) REFERENCES note (id),
	FOREIGN KEY (blob_sha256) REFERENCES blob (sha256),
	
	PRIMARY KEY (note_id, blob_sha256, timestamp)
);

/* blob is the payload of a note_rev. Blobs are split off from revs to
enable deduplication of blobs across notes and revisions. */
CREATE TABLE blob (
	sha256 VARCHAR(64) PRIMARY KEY,
	body blob
);

/* Config stores user preferences for how notebooks behave */
CREATE TABLE config (
	key TEXT PRIMARY KEY,
	value TEXT,
	description TEXT
);

INSERT INTO config (key, value, description) VALUES
	("version", "0", "version of nestable for this nest"),
	("editor", "vi", "external editor to edit notes");
