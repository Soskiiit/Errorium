package handler

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// dummyBcryptHash is a precomputed valid cost-12 bcrypt hash used to equalize login response
// times when an email does not exist, mitigating timing-based user enumeration.
var dummyBcryptHash = []byte("$2a$12$1F6Qx61V4RnZgMt8OpttIeMqIBGw0juHFI15ukklKQAZAnwdjTxUy")

// dummyPasswordCheck performs a cost-12 bcrypt comparison against a dummy hash to equalize
// response timing and prevent user enumeration when the account does not exist.
func dummyPasswordCheck(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyBcryptHash, truncatePassword(password))
}

// truncatePassword truncates passwords exceeding bcrypt's 72-byte limit.
func truncatePassword(password string) []byte {
	b := []byte(password)
	if len(b) > 72 {
		return b[:72]
	}
	return b
}

// hashPassword derives a storable password hash using bcrypt with cost 12.
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword(truncatePassword(password), 12)
	if err != nil {
		return "", fmt.Errorf("hashPassword: %w", err)
	}
	return string(bytes), nil
}

// checkPassword verifies a plain-text password against a stored hash.
// Supports both bcrypt ($2...) and legacy SHA-256 (<salt>:<hash>) formats.
func checkPassword(storedHash, password string) bool {
	if strings.HasPrefix(storedHash, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(storedHash), truncatePassword(password)) == nil
	}
	if len(storedHash) < 33 || storedHash[32] != ':' {
		return false
	}
	saltHex := storedHash[:32]
	h := sha256.Sum256([]byte(saltHex + password))
	expected := saltHex + ":" + hex.EncodeToString(h[:])
	return subtle.ConstantTimeCompare([]byte(storedHash), []byte(expected)) == 1
}

// needsRehash returns true if the stored hash is not in bcrypt format.
func needsRehash(storedHash string) bool {
	return !strings.HasPrefix(storedHash, "$2")
}
