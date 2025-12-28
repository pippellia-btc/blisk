package blisk

import (
	"bytes"
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
	blob := bytes.NewReader(original)
	hash := sha256.Sum256(original)

	if _, err = store.Save(ctx, blob, ""); err != nil {
		t.Fatal(err)
	}

	file, err := store.Load(ctx, hash)
	if err != nil {
		t.Fatal(err)
	}

	data, err := io.ReadAll(file)
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
		Type: http.DetectContentType(blob),
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

	data := []byte("buddy")
	blob := bytes.NewReader(data)
	hash := sha256.Sum256(data)

	if _, err = store.Save(ctx, blob, "alice"); err != nil {
		t.Fatal(err)
	}

	if err = store.Delete(ctx, hash, "bob"); err != nil {
		t.Fatal(err)
	}

	if _, err = store.Info(ctx, hash); err != nil {
		// the blob should not have been deleted, since "bob" didn't upload it.
		// Therefore this call to Info should not return an error.
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

	originals := make([]blossom.Hash, 100)
	for i := range 100 {
		data := []byte(fmt.Sprintf("blob %d", i))
		blob := bytes.NewReader(data)

		meta, err := store.Save(ctx, blob, "alice")
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		originals[i] = meta.Hash
	}

	hashes, err := store.Hashes(ctx, "alice")
	if err != nil {
		t.Fatalf("expected error nil, got %v", err)
	}

	slices.SortFunc(originals, func(a, b blossom.Hash) int {
		return cmp.Compare(string(a[:]), string(b[:]))
	})
	slices.SortFunc(hashes, func(a, b blossom.Hash) int {
		return cmp.Compare(string(a[:]), string(b[:]))
	})

	if !slices.Equal(originals, hashes) {
		t.Fatalf("expected hashes %v, got %v", originals, hashes)
	}
}
