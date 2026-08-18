package main

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrationSourceURLUsesAbsoluteFileURL(t *testing.T) {
	directory := t.TempDir()

	value, err := migrationSourceURL(directory)
	if err != nil {
		t.Fatalf("migrationSourceURL returned an error: %v", err)
	}

	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatalf("parse migration URL: %v", err)
	}
	if parsed.Scheme != "file" {
		t.Fatalf("scheme = %q, want file", parsed.Scheme)
	}

	want, err := filepath.Abs(directory)
	if err != nil {
		t.Fatalf("resolve expected path: %v", err)
	}
	got := filepath.FromSlash(parsed.Host + parsed.Path)
	if got != filepath.Clean(want) {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestMigrationSourceURLRejectsFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "migration.sql")
	if err := os.WriteFile(file, []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatalf("write temporary file: %v", err)
	}

	if _, err := migrationSourceURL(file); err == nil {
		t.Fatal("migrationSourceURL should reject a file path")
	}
}
