package blobstore

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	store, err := New("blossom")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
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
