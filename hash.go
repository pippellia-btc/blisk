package blobstore

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
)

type Hash [32]byte

// Hex converts the hash into its hexadecimal representation.
func (h Hash) Hex() string {
	return hex.EncodeToString(h[:])
}

// String converts the hash into its hexadecimal representation.
func (h Hash) String() string {
	return h.Hex()
}

// Value implements the [driver.Valuers] interface so it can serialize itself as a hexadecimal string.
func (h Hash) Value() (driver.Value, error) {
	return h.Hex(), nil
}

// Scan implements the [sql.Scanner] interface so it can deserialize itself.
func (h *Hash) Scan(src any) error {
	switch s := src.(type) {
	case string:
		if len(s) != 64 {
			return fmt.Errorf("invalid hash length: %d", len(s))
		}
		b, err := hex.DecodeString(s)
		if err != nil {
			return err
		}
		copy(h[:], b)
		return nil

	case []byte:
		if len(s) != 32 {
			return fmt.Errorf("invalid hash length: %d", len(s))
		}
		copy(h[:], s)
		return nil

	case nil:
		return fmt.Errorf("NULL cannot be scanned into Hash")

	default:
		return fmt.Errorf("cannot scan %T into Hash", src)
	}
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
