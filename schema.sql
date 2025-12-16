CREATE TABLE IF NOT EXISTS blobs (
    hash        TEXT    PRIMARY KEY,    -- sha256 stored as an hexadecimal
    mime        TEXT    NOT NULL,
    size        INTEGER NOT NULL,
    created_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS uploads (
    uploader    TEXT    NOT NULL,   -- normally the nostr pubkey that uploaded the blob
    hash        TEXT    NOT NULL,   -- sha256 stored as an hexadecimal
    timestamp   INTEGER NOT NULL,
    PRIMARY KEY (uploader, hash),
    FOREIGN KEY (hash) REFERENCES blobs(hash) ON DELETE CASCADE
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS uploads_by_hash ON uploads(hash);