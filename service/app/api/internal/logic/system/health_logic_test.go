package system

import (
	"context"
	"testing"
)

func TestHealth(t *testing.T) {
	logic := NewHealthLogic(context.Background(), nil)

	response, err := logic.Health()
	if err != nil {
		t.Fatalf("Health() returned an error: %v", err)
	}
	if response == nil {
		t.Fatal("Health() returned a nil response")
	}
	if response.Status != "ok" {
		t.Errorf("Status = %q, want %q", response.Status, "ok")
	}
	if response.Service != "app-api" {
		t.Errorf("Service = %q, want %q", response.Service, "app-api")
	}
	if response.Version != "v1" {
		t.Errorf("Version = %q, want %q", response.Version, "v1")
	}
}
