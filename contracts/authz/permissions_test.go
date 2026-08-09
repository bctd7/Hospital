package authz

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestGoPermissionCatalogMatchesYAML(t *testing.T) {
	file, err := os.Open("permissions.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	yamlCodes := make(map[string]struct{})
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "- code: ") {
			yamlCodes[strings.TrimSpace(strings.TrimPrefix(line, "- code: "))] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	goCodes := make(map[string]struct{}, len(AllPermissions))
	for _, permission := range AllPermissions {
		if permission.Code == "" || permission.Service == "" {
			t.Fatalf("invalid permission entry: %#v", permission)
		}
		if _, exists := goCodes[permission.Code]; exists {
			t.Fatalf("duplicate Go permission %q", permission.Code)
		}
		goCodes[permission.Code] = struct{}{}
	}

	if len(goCodes) != len(yamlCodes) {
		t.Fatalf("Go permission count %d does not match YAML count %d", len(goCodes), len(yamlCodes))
	}
	for code := range goCodes {
		if _, exists := yamlCodes[code]; !exists {
			t.Errorf("permission %q is missing from permissions.yaml", code)
		}
	}
}
