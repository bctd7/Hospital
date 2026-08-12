package main

import (
	"encoding/base64"
	"testing"
)

func TestParseAdminPhonesNormalizesAndRejectsDuplicates(t *testing.T) {
	phones, err := parseAdminPhones("138-0013-8000, +86 13900139000")
	if err != nil {
		t.Fatal(err)
	}
	if len(phones) != 2 || phones[0] != "+8613800138000" || phones[1] != "+8613900139000" {
		t.Fatalf("unexpected normalized phones: %#v", phones)
	}
	if _, err := parseAdminPhones("13800138000,+8613800138000"); err == nil {
		t.Fatal("expected duplicate phone rejection")
	}
}

func TestParseAdminPhonesRequiresConfiguredPhone(t *testing.T) {
	if _, err := parseAdminPhones(" , "); err == nil {
		t.Fatal("expected empty administrator list rejection")
	}
	if _, err := parseAdminPhones("123"); err == nil {
		t.Fatal("expected invalid phone rejection")
	}
}

func TestDecodePhoneLookupKey(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString(make([]byte, minimumKeyLength))
	key, err := decodePhoneLookupKey(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != minimumKeyLength {
		t.Fatalf("unexpected key length: %d", len(key))
	}
	if _, err := decodePhoneLookupKey(base64.StdEncoding.EncodeToString(make([]byte, 16))); err == nil {
		t.Fatal("expected short key rejection")
	}
}

func TestNewAdminSeedUsesStableFingerprintAndMaskedPhone(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	first := newAdminSeed("+8613800138000", key)
	second := newAdminSeed("+8613800138000", key)
	if first.maskedPhone != "138****8000" {
		t.Fatalf("unexpected masked phone: %s", first.maskedPhone)
	}
	if string(first.fingerprint) != string(second.fingerprint) {
		t.Fatal("expected stable phone fingerprint")
	}
}
