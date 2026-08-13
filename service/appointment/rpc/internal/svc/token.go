package svc

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
)

func decodePublicKey(encoded string) (ed25519.PublicKey, error) {
	if encoded == "" {
		return nil, errors.New("access public key is required")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	if len(decoded) != ed25519.PublicKeySize {
		return nil, errors.New("access public key must be an Ed25519 public key")
	}
	return ed25519.PublicKey(decoded), nil
}
