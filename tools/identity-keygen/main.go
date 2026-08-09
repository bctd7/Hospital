package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func main() {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	fmt.Printf("IDENTITY_ACCESS_PRIVATE_KEY_BASE64=%s\n", base64.StdEncoding.EncodeToString(privateKey))
	fmt.Printf("IDENTITY_ACCESS_PUBLIC_KEY_BASE64=%s\n", base64.StdEncoding.EncodeToString(publicKey))
}
