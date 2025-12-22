package blisk

import (
	"cmp"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/pippellia-btc/blossom"
)

var (
	ctx     = context.Background()
	testDir = "test"
)

func init() {
	// initialize the blisk directories and index
	store, err := New(testDir)
	if err != nil {
		panic(err)
	}
	defer store.Close()
}

func TestSaveLoadBlob(t *testing.T) {
	store, err := New(testDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	original := []byte("bobby")
	hash := sha256.Sum256(original)

	if err := store.saveBlob(hash, original); err != nil {
		t.Fatal(err)
	}

	blob, err := store.Load(ctx, hash)
	if err != nil {
		t.Fatal(err)
	}

	data, err := io.ReadAll(blob)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(original, data) {
		t.Fatalf("blob has been altered: original %v, got %v", string(original), string(data))
	}
}

func TestSaveInfoMeta(t *testing.T) {
	store, err := New(testDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	blob := []byte("blobby")
	original := blossom.BlobMeta{
		Hash: blossom.ComputeHash(blob),
		MIME: http.DetectContentType(blob),
		Size: int64(len(blob)),
	}

	original.CreatedAt, err = store.saveMeta(ctx, "whatever", original)
	if err != nil {
		t.Fatal(err)
	}

	stored, err := store.Info(ctx, original.Hash)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(original, stored) {
		t.Error("metadata has been altered")
		t.Errorf("original: \t%v", original)
		t.Errorf("stored:   \t%v", stored)
	}
}

func TestSaveDelete(t *testing.T) {
	store, err := New(testDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	blob := []byte("buddy")
	hash := blossom.ComputeHash(blob)

	if _, err = store.Save(ctx, blob, "alice"); err != nil {
		t.Fatal(err)
	}

	if err = store.Delete(ctx, hash, "bob"); err != nil {
		t.Fatal(err)
	}

	if _, err = store.Info(ctx, hash); err != nil {
		// the blob should not have been deleted, since "bob" didn't upload it.
		// Therefore this Info should not return an error.
		t.Fatalf("expected no error from Info, got %v", err)
	}

	if err = store.Delete(ctx, hash, "alice"); err != nil {
		t.Fatal(err)
	}

	if _, err = store.Info(ctx, hash); !errors.Is(err, ErrNotFound) {
		// the blob should have been deleted, so Info should return [ErrNotFound].
		t.Fatalf("expected error %v, got %v", ErrNotFound, err)
	}
}

func TestBlobsOf(t *testing.T) {
	store, err := New(testDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	original := make([]blossom.Hash, 100)
	for i := range 100 {
		blob := []byte(fmt.Sprintf("blob %d", i))

		meta, err := store.Save(ctx, blob, "alice")
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		original[i] = meta.Hash
	}

	hashes, err := store.Hashes(ctx, "alice")
	if err != nil {
		t.Fatalf("expected error nil, got %v", err)
	}

	slices.SortFunc(original, func(a, b blossom.Hash) int {
		return cmp.Compare(string(a[:]), string(b[:]))
	})
	slices.SortFunc(hashes, func(a, b blossom.Hash) int {
		return cmp.Compare(string(a[:]), string(b[:]))
	})

	if !slices.Equal(original, hashes) {
		t.Fatalf("expected hashes %v, got %v", original, hashes)
	}
}
