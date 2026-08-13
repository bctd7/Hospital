package svc

import (
	"testing"

	"hospital/service/identity/rpc/internal/authentication/sms"
	"hospital/service/identity/rpc/internal/config"
)

func TestSMSVerifierAllowsLocalOnlyInSafeEnvironments(t *testing.T) {
	for _, environment := range []string{"local", "test"} {
		var value config.Config
		value.Environment = environment
		value.PhoneLogin.Verifier = "local"
		value.PhoneLogin.LocalCode = "246810"
		verifier, err := buildSMSVerifier(value)
		if err != nil {
			t.Fatalf("environment %s: %v", environment, err)
		}
		if _, ok := verifier.(*sms.LocalVerifier); !ok {
			t.Fatalf("environment %s returned %T", environment, verifier)
		}
	}
}

func TestSMSVerifierRejectsLocalInProduction(t *testing.T) {
	var value config.Config
	value.Environment = "production"
	value.PhoneLogin.Verifier = "local"
	value.PhoneLogin.LocalCode = "246810"
	if _, err := buildSMSVerifier(value); err == nil {
		t.Fatal("expected production local verifier configuration to fail")
	}
}

func TestSMSVerifierKeepsDisabledDefault(t *testing.T) {
	var value config.Config
	verifier, err := buildSMSVerifier(value)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := verifier.(sms.UnconfiguredVerifier); !ok {
		t.Fatalf("expected disabled verifier, got %T", verifier)
	}
}
