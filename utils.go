package blisk

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
)

// Hex3 is an hexadecimal string consisting of 3 characters, e.g. "00f".
type Hex3 string

// Int converts the Hex3 to an integer between 0 and 4095.
func (h Hex3) Int() int {
	if len(h) != 3 {
		panic("hex3 must be exactly 3 characters")
	}
	n, err := strconv.ParseUint(string(h), 16, 16) // 16 bits is enough
	if err != nil {
		panic("failed to decode hex3: " + err.Error())
	}
	return int(n)
}

// ToHex3 converts an integer between 0 and 4095 to a Hex3.
func ToHex3(n int) (Hex3, error) {
	if n < 0 || n > 4095 {
		return "", fmt.Errorf("number out of range: %d", n)
	}
	return Hex3(fmt.Sprintf("%03x", n)), nil
}

var hexes = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f"}

// AllHex3 returns all 4096 possible combinations of exactly 3 hex characters.
func AllHex3() []Hex3 {
	comb := make([]Hex3, 0, 4096)
	for _, c1 := range hexes {
		for _, c2 := range hexes {
			for _, c3 := range hexes {
				comb = append(comb, Hex3(c1+c2+c3))
			}
		}
	}
	return comb
}

// RandHex3 returns a random Hex3. Not suitable for security purposes.
func RandHex3() Hex3 {
	n := rand.IntN(4096)
	return Hex3(fmt.Sprintf("%03x", n))
}

// Cleanup closes a (temporary) file and removes it from disk.
func cleanup(f *os.File) {
	f.Close()
	os.Remove(f.Name())
}
