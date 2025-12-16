package blobstore

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"slices"
	"testing"
)

var (
	ctx     = context.Background()
	testDir = "test"
)

func init() {
	// initialize the blobstore directories and index
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

func TestSaveLookupMeta(t *testing.T) {
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

	stored, err := store.Lookup(ctx, original.Hash)
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

	if _, err = store.Lookup(ctx, hash); err != nil {
		// the blob should not have been deleted, since "bob" didn't upload it.
		// Therefore this lookup should not return an error.
		t.Fatalf("expected no error from lookup")
	}

	if err = store.Delete(ctx, hash, "alice"); err != nil {
		t.Fatal(err)
	}

	if _, err = store.Lookup(ctx, hash); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected error %v, got %v", ErrNotFound, err)
	}
}

// ExtractMetadata reads the file at path and returns its MIME type and size.
func ExtractMetadata(file *os.File) (mime string, size int64, err error) {
	info, err := file.Stat()
	if err != nil {
		return "", 0, err
	}

	// Read first 512 bytes for MIME detection
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", 0, err
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", 0, err
	}

	mime = http.DetectContentType(buf[:n])
	size = info.Size()
	return mime, size, nil
}

func TestBlobPath(t *testing.T) {
	tests := []struct {
		hex  string
		path string
	}{
		{hex: "b133a0c0e9bee3be20163d2ad31d6248db292aa6dcb1ee087a2aa50e0fc75ae2", path: "blobs/b1/33/b133a0c0e9bee3be20163d2ad31d6248db292aa6dcb1ee087a2aa50e0fc75ae2"},
		{hex: "e5990593c22e6647a7bbde8e85f395e56727040cd8ccecb6d23ae41852a287b1", path: "blobs/e5/99/e5990593c22e6647a7bbde8e85f395e56727040cd8ccecb6d23ae41852a287b1"},
		{hex: "9c19636ddea25f0a13357d01c0aef27a4eb4c7f8cbaf2be35f272273572ce2e5", path: "blobs/9c/19/9c19636ddea25f0a13357d01c0aef27a4eb4c7f8cbaf2be35f272273572ce2e5"},
		{hex: "70c6b82e6e442e607ba5ad8f38b4d69e17fd1a34b7ba765c47f2afe386917396", path: "blobs/70/c6/70c6b82e6e442e607ba5ad8f38b4d69e17fd1a34b7ba765c47f2afe386917396"},
		{hex: "69a48df50dd885d31f9c083a079c979434d758aefd13ee6d961a6d3c5a10f9d5", path: "blobs/69/a4/69a48df50dd885d31f9c083a079c979434d758aefd13ee6d961a6d3c5a10f9d5"},
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
