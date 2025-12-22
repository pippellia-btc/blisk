package blisk

import (
	"fmt"
	"testing"
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
