package session

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/google/uuid"
)

const refreshSecretSize = 32

func newRefreshToken(sessionID string) (raw, hash string, err error) {
	secret := make([]byte, refreshSecretSize)
	if _, err = rand.Read(secret); err != nil {
		return "", "", err
	}
	raw = sessionID + "." + base64.RawURLEncoding.EncodeToString(secret)
	return raw, hashRefreshToken(raw), nil
}

func parseRefreshToken(raw string) (sessionID, hash string, err error) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 2 {
		return "", "", ErrInvalidRefreshToken
	}
	parsedID, err := uuid.Parse(parts[0])
	if err != nil || parsedID.String() != parts[0] {
		return "", "", ErrInvalidRefreshToken
	}
	secret, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(secret) != refreshSecretSize {
		return "", "", ErrInvalidRefreshToken
	}
	return parts[0], hashRefreshToken(raw), nil
}

func hashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func tokenHashMatches(expected, actual string) bool {
	if len(expected) != len(actual) || len(expected) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}
