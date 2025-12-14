package blobstore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schema string

var hexes = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f"}

type Store struct {
	DB      *sql.DB
	dirPath string
}

// New returns a database for blossom blobs, indexed by sqlite3.
func New(dir string, opts ...Option) (*Store, error) {
	if err := initBlobDirs(dir); err != nil {
		return nil, err
	}

	db, err := initSqlite(dir)
	if err != nil {
		return nil, err
	}

	store := &Store{
		DB:      db,
		dirPath: dir,
	}

	for _, opt := range opts {
		if err := opt(store); err != nil {
			return nil, err
		}
	}

	// run full optimize after options, to inform the query planner about new indexes (if any).
	if _, err := db.Exec("PRAGMA optimize=0x10002;"); err != nil {
		return nil, fmt.Errorf("failed to PRAGMA optimize: %w", err)
	}
	return store, nil
}

// Close the underlying database connection, committing all temporary data to disk.
func (s *Store) Close() error {
	return s.DB.Close()
}

// initBlobDirs creates the main dir /blobs and all sharder sub directories e.g. /aa/bb
func initBlobDirs(path string) error {
	baseDir := filepath.Join(path, "blobs")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return fmt.Errorf("failed to create base dir: %w", err)
	}

	for _, c1 := range hexes {
		for _, c2 := range hexes {
			for _, c3 := range hexes {
				for _, c4 := range hexes {
					// create sub directories /blobs/aa/bb
					path = filepath.Join(baseDir, c1+c2, c3+c4)
					if err := os.MkdirAll(path, 0o755); err != nil {
						return fmt.Errorf("failed to create %s: %w", path, err)
					}
				}
			}
		}
	}
	return nil
}

func initSqlite(path string) (*sql.DB, error) {
	name := filepath.Join(path, "index.sqlite")
	db, err := sql.Open("sqlite3", name)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite3 at %s: %w", name, err)
	}

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to apply base schema: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA busy_timeout = 1000;"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}
	return db, nil
}

// Save the provided blob by its hash, setting its ownership by the provided pubkey.
func (s *Store) Save(ctx context.Context, blob []byte, pubkey string) error {
	hash := sha256.Sum256(blob)
	hex := hex.EncodeToString(hash[:])
	path := s.BlobPath(hex)

	err := os.WriteFile(path, blob, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write blob %s: %w", hex, err)
	}

	mime := http.DetectContentType(blob)
	size := len(blob)

	now := time.Now().Unix()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT OR IGNORE INTO blobs (hash, mime, size, created_at, uploaded_by) VALUES (?, ?, ?, ?, 0)`,
		hash, mime, size, now)
	if err != nil {
		return err
	}

	res, err := tx.Exec(`INSERT OR IGNORE INTO uploads (pubkey, hash, timestamp) VALUES (?, ?, ?)
	`, pubkey, hash, now)
	if err != nil {
		return err
	}

	rows, _ := res.RowsAffected()
	if rows > 0 {
		// if it was added to the uploads, increment the counter in blobs
		_, err = tx.Exec(`UPDATE blobs SET uploaded_by = uploaded_by + 1 WHERE hash = ?`, hash)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// BlobPath returns the path of the blob based on its hash and the store directory.
// The hash must be encoded as an hexadecimal.
func (s *Store) BlobPath(hash string) string {
	return filepath.Join(s.dirPath, blobPath(hash))
}

// BlobPath returns the path of the blob based on its hash.
// The hash must be encoded as an hexadecimal.
func blobPath(hash string) string {
	return filepath.Join("blobs", hash[0:2], hash[2:4], hash)
}
