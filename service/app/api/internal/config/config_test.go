package config

import (
	"path/filepath"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestAppAPIConfig(t *testing.T) {
	var config Config
	configPath := filepath.Join("..", "..", "etc", "app-api.yaml")
	if err := conf.Load(configPath, &config); err != nil {
		t.Fatalf("load app API config: %v", err)
	}

	if config.Environment != "local" {
		t.Errorf("Environment = %q, want %q", config.Environment, "local")
	}
	if config.Log.Encoding != "json" {
		t.Errorf("Log.Encoding = %q, want %q", config.Log.Encoding, "json")
	}
	if config.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q", config.Log.Level, "info")
	}
	if config.Middlewares.Log {
		t.Error("Middlewares.Log = true, want false so request bodies are not logged")
	}
	if !config.Middlewares.Trace || !config.Middlewares.Recover || !config.Middlewares.Timeout {
		t.Error("Trace, Recover, and Timeout middlewares must remain enabled")
	}
}
