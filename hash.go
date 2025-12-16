package blobstore

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
)

type Hash [32]byte

// String converts the hash into its hexadecimal representation.
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// Value implements the [driver.Valuers] interface so it can serialize itself as a hexadecimal string.
func (h Hash) Value() (driver.Value, error) {
	return hex.EncodeToString(h[:]), nil
}

// ComputeHash of the provided data, by calling the cryptographically secure
// sha256 implementation of the standard library.
func ComputeHash(data []byte) Hash {
	return sha256.Sum256(data)
}

// ParseHash from the hexadecimal input string.
func ParseHash(input string) (Hash, error) {
	if len(input) != 64 {
		return Hash{}, errors.New("input lenght must be exactly 64 characters")
	}

	var hash Hash
	b, err := hex.DecodeString(input)
	if err != nil {
		return Hash{}, fmt.Errorf("failed to parsh hash: %w", err)
	}

	copy(hash[:], b)
	return hash, nil
}
