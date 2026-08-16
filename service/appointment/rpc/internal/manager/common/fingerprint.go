package common

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// RequestFingerprint returns a compact equality key for an idempotent request.
// Operation and request identifiers are deliberately excluded by their input
// types because they identify transport attempts rather than business content.
func RequestFingerprint(payload any) string {
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
