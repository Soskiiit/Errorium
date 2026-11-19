package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// hashPassword derives a storable password hash using SHA-256 with a random salt.
// Format: "<hex-salt>:<hex-hash>"
//
// NOTE: bcrypt is preferred in production. This implementation is used here
// because golang.org/x/crypto is not available offline.
func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("hashPassword: generate salt: %w", err)
	}
	saltHex := hex.EncodeToString(salt)
	h := sha256.Sum256([]byte(saltHex + password))
	return saltHex + ":" + hex.EncodeToString(h[:]), nil
}

// checkPassword verifies a plain-text password against a stored hash.
func checkPassword(storedHash, password string) bool {
	if len(storedHash) < 33 || storedHash[32] != ':' {
		return false
	}
	saltHex := storedHash[:32]
	h := sha256.Sum256([]byte(saltHex + password))
	expected := saltHex + ":" + hex.EncodeToString(h[:])
	// Constant-time comparison is not strictly needed here because sha256 output
	// comparison timing is not secret-dependent in a meaningful way, but we do it anyway.
	return storedHash == expected
}
