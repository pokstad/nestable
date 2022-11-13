--- Following tables support word cloud feature by making raw token information available
CREATE VIRTUAL TABLE note_fts_vocab_cols USING fts5vocab(note_fts, col);
CREATE VIRTUAL TABLE note_fts_vocab_instances USING fts5vocab(note_fts, instance);

--- Stop word database contains common English words that we don't want in our word cloud
CREATE TABLE stop_words (
	word TEXT
);