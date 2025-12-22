package blisk

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/pippellia-btc/blossom"
)

func TestToHex3(t *testing.T) {
	tests := []struct {
		n    int
		hex3 Hex3
	}{
		{n: 0, hex3: "000"},
		{n: 1, hex3: "001"},
		{n: 15, hex3: "00f"},
		{n: 4095, hex3: "fff"},
		{n: 171, hex3: "0ab"},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("case=%d", i), func(t *testing.T) {
			got, err := ToHex3(test.n)
			if err != nil {
				t.Fatal(err)
			}

			if got != test.hex3 {
				t.Fatalf("expected %v, got %v", test.hex3, got)
			}
		})
	}
}

func TestHex3Int(t *testing.T) {
	tests := []struct {
		n    int
		hex3 Hex3
	}{
		{n: 0, hex3: "000"},
		{n: 1, hex3: "001"},
		{n: 15, hex3: "00f"},
		{n: 4095, hex3: "fff"},
		{n: 171, hex3: "0ab"},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("case=%d", i), func(t *testing.T) {
			got := test.hex3.Int()

			if got != test.n {
				t.Fatalf("expected %v, got %v", test.hex3, got)
			}
		})
	}
}

func TestBlobPath(t *testing.T) {
	s := Store{}

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
			hash, err := blossom.ParseHash(test.hex)
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			path := s.BlobPath(hash)
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
