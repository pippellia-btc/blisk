package blisk

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schema string

var hexes = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f"}

var (
	ErrNotFound = errors.New("blob not found")
)

// Store is a local database of blossom blobs, indexed by sqlite.
type Store struct {
	writeMu sync.Mutex
	index   *sql.DB
	dirPath string
}

// New returns a local database for blossom blobs, indexed by sqlite.
func New(dir string) (*Store, error) {
	if err := initBlobDirs(dir); err != nil {
		return nil, err
	}

	db, err := initSqlite(dir)
	if err != nil {
		return nil, err
	}

	store := &Store{
		index:   db,
		dirPath: dir,
	}
	return store, nil
}

// Close the underlying database connection, committing all temporary files.
func (s *Store) Close() error {
	return s.index.Close()
}

// BlobPath returns the path of the blob based on its hash and the store directory.
func (s *Store) BlobPath(hash Hash) string {
	return filepath.Join(s.dirPath, blobPath(hash))
}

// BlobPath returns the path of the blob based on its hash.
func blobPath(hash Hash) string {
	hex := hash.String()
	return filepath.Join("blobs", hex[0:2], hex[2:4], hex)
}

// Optimize runs "PRAGMA optimize", which updates the statistics and heuristics
// of the query planner, improving read performance.
func (s *Store) Optimize(ctx context.Context) error {
	_, err := s.index.ExecContext(ctx, "PRAGMA optimize;")
	return err
}

// InitBlobDirs creates the main dir /blobs and all sharded sub directories e.g. /blobs/aa/bb
func initBlobDirs(path string) error {
	baseDir := filepath.Join(path, "blobs")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return fmt.Errorf("failed to create base dir: %w", err)
	}

	for _, c1 := range hexes {
		for _, c2 := range hexes {
			for _, c3 := range hexes {
				for _, c4 := range hexes {
					shard := filepath.Join(baseDir, c1+c2, c3+c4)
					if err := os.MkdirAll(shard, 0o755); err != nil {
						return fmt.Errorf("failed to create %s: %w", shard, err)
					}
				}
			}
		}
	}
	return nil
}

// InitSqlite initalize the sqlite index of the store.
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

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to activate foreign keys: %w", err)
	}

	if _, err = db.Exec("PRAGMA optimize=0x10002;"); err != nil {
		return nil, fmt.Errorf("failed to PRAGMA optimize: %w", err)
	}
	return db, nil
}

// Save the provided blob by its hash, returning the [BlobMeta] or an error.
// It is idempotent; multiple calls to Save with the same blob and uploader will result in a no-op.
func (s *Store) Save(ctx context.Context, blob []byte, uploader string) (BlobMeta, error) {
	meta := BlobMeta{
		Hash: ComputeHash(blob),
		MIME: http.DetectContentType(blob),
		Size: int64(len(blob)),
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	err := s.saveBlob(meta.Hash, blob)
	if err != nil {
		return BlobMeta{}, fmt.Errorf("failed to save blob %s: %w", meta.Hash, err)
	}

	meta.CreatedAt, err = s.saveMeta(ctx, uploader, meta)
	if err != nil {
		return BlobMeta{}, fmt.Errorf("failed to save metadata of blob %s: %w", meta.Hash, err)
	}
	return meta, nil
}

// SaveBlob by the provided hash.
// It is idempotent; multiple calls to saveBlob with the same hash will return early without writing to disk.
func (s *Store) saveBlob(hash Hash, blob []byte) error {
	path := s.BlobPath(hash)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = file.Write(blob)
	cErr := file.Close()
	if err == nil && cErr != nil {
		err = cErr
	}

	if err != nil {
		os.Remove(path)
		return err
	}
	return nil
}

// SaveMeta saves the blob metadata.
// It is idempotent; multiple calls to saveMeta with the same hash and uploader will result in a no-op.
func (s *Store) saveMeta(ctx context.Context, uploader string, meta BlobMeta) (createdAt int64, err error) {
	now := time.Now().Unix()

	tx, err := s.index.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT OR IGNORE INTO blobs (hash, mime, size, created_at) VALUES (?, ?, ?, ?)`,
		meta.Hash, meta.MIME, meta.Size, now)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`INSERT OR IGNORE INTO uploads (uploader, hash, timestamp) VALUES (?, ?, ?)`,
		uploader, meta.Hash, now)
	if err != nil {
		return 0, err
	}

	row := tx.QueryRow(`SELECT created_at from blobs WHERE hash = ?`, meta.Hash)
	if err = row.Scan(&createdAt); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return createdAt, nil
}

// Info returns the metadata of a blob, identified by the provided hash.
func (s *Store) Info(ctx context.Context, hash Hash) (BlobMeta, error) {
	meta := BlobMeta{Hash: hash}
	row := s.index.QueryRowContext(ctx, `SELECT mime, size, created_at FROM blobs WHERE hash = ?`, hash)

	err := row.Scan(&meta.MIME, &meta.Size, &meta.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return BlobMeta{}, fmt.Errorf("%w: hash %s", ErrNotFound, hash)
	}
	if err != nil {
		return BlobMeta{}, fmt.Errorf("failed to fetch metadata of blob %s: %w", hash, err)
	}
	return meta, nil
}

// Hashes return the list of all the hashes of blobs uploaded by the provided uploader.
func (s *Store) Hashes(ctx context.Context, uploader string) ([]Hash, error) {
	hashes, err := s.hashes(ctx, uploader)
	if err != nil {
		return nil, fmt.Errorf("failed to the blobs of of %s: %w", uploader, err)
	}
	return hashes, nil
}

func (s *Store) hashes(ctx context.Context, uploader string) ([]Hash, error) {
	rows, err := s.index.QueryContext(ctx, `SELECT hash FROM uploads WHERE uploader = ?`, uploader)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hashes := make([]Hash, 0, 20)
	for rows.Next() {
		var hash Hash
		if err = rows.Scan(&hash); err != nil {
			return nil, err
		}

		hashes = append(hashes, hash)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hashes, nil
}

// Load the [Blob] by the provided hash.
func (s *Store) Load(ctx context.Context, hash Hash) (*os.File, error) {
	path := s.BlobPath(hash)
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: hash %s", ErrNotFound, hash)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob %s: %w", hash, err)
	}
	return file, nil
}

// Delete a blob with the provided hash from the deleter uploads.
// If a blob is not referenced by any upload, then the blob is deleted.
func (s *Store) Delete(ctx context.Context, hash Hash, deleter string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	err := s.delete(ctx, hash, deleter)
	if err != nil {
		return fmt.Errorf("failed to delete blob %s from %s: %w", hash, deleter, err)
	}
	return nil
}

func (s *Store) delete(ctx context.Context, hash Hash, deleter string) error {
	tx, err := s.index.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`DELETE FROM uploads WHERE uploader = ? AND hash = ?`, deleter, hash)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return nil
	}

	var uploads int64
	err = tx.QueryRow(`SELECT COUNT(*) FROM uploads WHERE hash = ?`, hash).Scan(&uploads)
	if err != nil {
		return err
	}

	if uploads == 0 {
		_, err = tx.Exec(`DELETE FROM blobs WHERE hash = ?`, hash)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if uploads == 0 {
		path := s.BlobPath(hash)
		err = os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove file %w", err)
		}
	}
	return nil
}
