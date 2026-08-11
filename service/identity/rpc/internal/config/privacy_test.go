package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	identityv1 "hospital/contracts/gen/identity/v1"
)

func TestIdentityConfigsSuppressAdminPhoneSearchContent(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "..", ".."))
	configs := []string{
		filepath.Join(repositoryRoot, "service", "identity", "rpc", "etc", "identity-rpc.yaml"),
		filepath.Join(repositoryRoot, "deploy", "production", "config", "identity-rpc.yaml"),
	}
	want := identityv1.IdentityService_SearchAdminAccountByPhone_FullMethodName
	for _, configPath := range configs {
		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("read %s: %v", configPath, err)
		}
		if !strings.Contains(string(content), want) {
			t.Errorf("%s does not suppress request content for %s", configPath, want)
		}
	}
}
