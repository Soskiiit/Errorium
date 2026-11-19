package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestPassword_BcryptRoundtrip(t *testing.T) {
	password := "SecretPassw0rd!"

	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword failed: %v", err)
	}

	if !strings.HasPrefix(hash, "$2") {
		t.Errorf("expected bcrypt prefix $2, got %q", hash)
	}

	if !checkPassword(hash, password) {
		t.Errorf("expected checkPassword to return true for correct password")
	}

	if checkPassword(hash, "WrongPassword!") {
		t.Errorf("expected checkPassword to return false for incorrect password")
	}
}

func TestPassword_BcryptLengths(t *testing.T) {
	testCases := []struct {
		name string
		len  int
	}{
		{"72-byte password", 72},
		{"73-byte password", 73},
		{"128-byte password", 128},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pw := strings.Repeat("x", tc.len)
			hash, err := hashPassword(pw)
			if err != nil {
				t.Fatalf("hashPassword failed for %d-byte password: %v", tc.len, err)
			}
			t.Logf("generated hash for %s: %s", tc.name, hash)
			if !strings.HasPrefix(hash, "$2") {
				t.Errorf("expected bcrypt prefix $2, got %q", hash)
			}
			if !checkPassword(hash, pw) {
				t.Errorf("expected checkPassword to return true for %d-byte password", tc.len)
			}
			if checkPassword(hash, "wrong-password") {
				t.Errorf("expected checkPassword to return false for wrong password")
			}
		})
	}
}

func TestPassword_LegacySHA256(t *testing.T) {
	saltHex := "0123456789abcdef0123456789abcdef" // 32 hex chars
	password := "legacySecret123"

	h := sha256.Sum256([]byte(saltHex + password))
	storedHash := saltHex + ":" + hex.EncodeToString(h[:])

	if !checkPassword(storedHash, password) {
		t.Errorf("expected legacy checkPassword to return true for valid password")
	}

	if checkPassword(storedHash, "invalidPassword") {
		t.Errorf("expected legacy checkPassword to return false for invalid password")
	}
}

func TestPassword_MalformedHashes(t *testing.T) {
	testCases := []struct {
		name string
		hash string
	}{
		{"empty", ""},
		{"too short", "abc:123"},
		{"32 chars without colon", "0123456789abcdef0123456789abcdef"},
		{"colon at wrong pos", "0123456789abcdef:0123456789abcdef"},
		{"garbage format", "invalid-hash-string-without-any-delimiter"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if checkPassword(tc.hash, "somepassword") {
				t.Errorf("expected false for malformed hash %q", tc.hash)
			}
		})
	}
}

func TestPassword_NeedsRehash(t *testing.T) {
	saltHex := "0123456789abcdef0123456789abcdef"
	h := sha256.Sum256([]byte(saltHex + "pass"))
	legacyHash := saltHex + ":" + hex.EncodeToString(h[:])

	if !needsRehash(legacyHash) {
		t.Errorf("expected needsRehash=true for legacy SHA-256 hash")
	}

	if !needsRehash("") {
		t.Errorf("expected needsRehash=true for empty hash")
	}

	if !needsRehash("plaintext_or_md5_hash") {
		t.Errorf("expected needsRehash=true for non-bcrypt hash")
	}

	if needsRehash("$2a$12$e8N45N1zWw8l0s5g2y9Uee94c25b34b82d3e") {
		t.Errorf("expected needsRehash=false for $2a$ bcrypt hash")
	}

	if needsRehash("$2b$12$e8N45N1zWw8l0s5g2y9Uee94c25b34b82d3e") {
		t.Errorf("expected needsRehash=false for $2b$ bcrypt hash")
	}

	bcryptHash, err := hashPassword("mySecretPassword")
	if err != nil {
		t.Fatalf("hashPassword failed: %v", err)
	}
	if needsRehash(bcryptHash) {
		t.Errorf("expected needsRehash=false for freshly hashed bcrypt password")
	}
}
