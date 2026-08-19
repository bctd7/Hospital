package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestProductionProviderEnvironmentIsLoaded(t *testing.T) {
	t.Setenv("AMAP_WEB_SERVICE_KEY", "amap-test-key")
	t.Setenv("GUIDANCE_LLM_ENDPOINT", "https://example.com/chat/completions")
	t.Setenv("GUIDANCE_LLM_API_KEY", "llm-test-key")
	t.Setenv("GUIDANCE_LLM_MODEL", "test-model")

	path := filepath.Join(t.TempDir(), "guidance.yaml")
	content := []byte(`Name: guidance-rpc
ListenOn: 127.0.0.1:8082
MySQL:
  DataSource: test
AuthorizationRedis:
  Addr: 127.0.0.1:6379
Token:
  AccessPublicKeyBase64: test
AppointmentRPC:
  Endpoints:
    - 127.0.0.1:8081
AMap:
  WebServiceKey: stale-amap-value
LLM:
  Endpoint: https://stale.example.com
  APIKey: stale-llm-value
  Model: stale-model
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	var cfg Config
	if err := conf.Load(path, &cfg, conf.UseEnv()); err != nil {
		t.Fatalf("load config: %v", err)
	}
	if err := BindProviderEnvironment(&cfg); err != nil {
		t.Fatalf("bind provider environment: %v", err)
	}
	if cfg.AMap.WebServiceKey != "amap-test-key" {
		t.Fatalf("AMap key = %q", cfg.AMap.WebServiceKey)
	}
	if cfg.LLM.Endpoint != "https://example.com/chat/completions" ||
		cfg.LLM.APIKey != "llm-test-key" || cfg.LLM.Model != "test-model" {
		t.Fatalf("LLM config was not expanded: endpoint=%q key=%q model=%q", cfg.LLM.Endpoint, cfg.LLM.APIKey, cfg.LLM.Model)
	}
}

func TestProviderEnvironmentRejectsPartialLLMConfiguration(t *testing.T) {
	t.Setenv("GUIDANCE_LLM_ENDPOINT", "https://example.com/chat/completions")
	t.Setenv("GUIDANCE_LLM_API_KEY", "")
	t.Setenv("GUIDANCE_LLM_MODEL", "")

	var cfg Config
	if err := BindProviderEnvironment(&cfg); !errors.Is(err, ErrIncompleteProviderEnvironment) {
		t.Fatalf("error = %v, want %v", err, ErrIncompleteProviderEnvironment)
	}
}
