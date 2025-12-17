package blisk

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"testing"
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
	original := BlobMeta{
		Hash: ComputeHash(blob),
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
	hash := ComputeHash(blob)

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

	original := make([]Hash, 100)
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

	slices.SortFunc(original, func(a, b Hash) int {
		return cmp.Compare(string(a[:]), string(b[:]))
	})
	slices.SortFunc(hashes, func(a, b Hash) int {
		return cmp.Compare(string(a[:]), string(b[:]))
	})

	if !slices.Equal(original, hashes) {
		t.Fatalf("expected hashes %v, got %v", original, hashes)
	}
}

func TestBlobPath(t *testing.T) {
	tests := []struct {
		hex  string
		path string
	}{
		{hex: "b133a0c0e9bee3be20163d2ad31d6248db292aa6dcb1ee087a2aa50e0fc75ae2", path: "blobs/b13/b133a0c0e9bee3be20163d2ad31d6248db292aa6dcb1ee087a2aa50e0fc75ae2"},
		{hex: "e5990593c22e6647a7bbde8e85f395e56727040cd8ccecb6d23ae41852a287b1", path: "blobs/e59/e5990593c22e6647a7bbde8e85f395e56727040cd8ccecb6d23ae41852a287b1"},
		{hex: "9c19636ddea25f0a13357d01c0aef27a4eb4c7f8cbaf2be35f272273572ce2e5", path: "blobs/9c1/9c19636ddea25f0a13357d01c0aef27a4eb4c7f8cbaf2be35f272273572ce2e5"},
		{hex: "70c6b82e6e442e607ba5ad8f38b4d69e17fd1a34b7ba765c47f2afe386917396", path: "blobs/70c/70c6b82e6e442e607ba5ad8f38b4d69e17fd1a34b7ba765c47f2afe386917396"},
		{hex: "69a48df50dd885d31f9c083a079c979434d758aefd13ee6d961a6d3c5a10f9d5", path: "blobs/69a/69a48df50dd885d31f9c083a079c979434d758aefd13ee6d961a6d3c5a10f9d5"},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("case=%d", i), func(t *testing.T) {
			hash, err := ParseHash(test.hex)
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			path := blobPath(hash)
			if path != test.path {
				t.Fatalf("expected path %v, got %v", test.path, path)
			}
		})
	}
}

func BenchmarkSHA256(b *testing.B) {
	sizes := []int{10_000, 100_000, 1_000_000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			data := make([]byte, size)
			rand.Read(data)

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				sha256.Sum256(data)
			}
		})
	}
}
