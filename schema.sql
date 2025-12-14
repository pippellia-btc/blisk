CREATE TABLE IF NOT EXISTS blobs (
    hash        BLOB    PRIMARY KEY,
    mime        TEXT    NOT NULL,
    size        INTEGER NOT NULL,
    created_at  INTEGER NOT NULL,
    uploaded_by INTEGER NOT NULL   -- number of pubkeys referencing this blob
);

CREATE TABLE IF NOT EXISTS uploads (
    pubkey      BLOB    NOT NULL,
    hash        BLOB    NOT NULL,
    timestamp INTEGER NOT NULL,
    PRIMARY KEY (pubkey, hash),
    FOREIGN KEY (hash) REFERENCES blobs(hash) ON DELETE CASCADE
) WITHOUT ROWID;
