package svc

import (
	"testing"

	authprovider "hospital/service/identity/rpc/internal/authentication/provider"
	"hospital/service/identity/rpc/internal/config"
)

func TestPhoneVerificationProviderAllowsLocalOnlyInSafeEnvironments(t *testing.T) {
	for _, environment := range []string{"local", "test"} {
		var value config.Config
		value.Environment = environment
		value.PhoneLogin.Provider = "local"
		value.PhoneLogin.LocalCode = "246810"
		phoneProvider, err := phoneVerificationProvider(value)
		if err != nil {
			t.Fatalf("environment %s: %v", environment, err)
		}
		if _, ok := phoneProvider.(*authprovider.LocalPhoneVerificationProvider); !ok {
			t.Fatalf("environment %s returned %T", environment, phoneProvider)
		}
	}
}

func TestPhoneVerificationProviderRejectsLocalInProduction(t *testing.T) {
	var value config.Config
	value.Environment = "production"
	value.PhoneLogin.Provider = "local"
	value.PhoneLogin.LocalCode = "246810"
	if _, err := phoneVerificationProvider(value); err == nil {
		t.Fatal("expected production local provider configuration to fail")
	}
}

func TestPhoneVerificationProviderKeepsDisabledDefault(t *testing.T) {
	var value config.Config
	phoneProvider, err := phoneVerificationProvider(value)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := phoneProvider.(authprovider.UnconfiguredPhoneVerificationProvider); !ok {
		t.Fatalf("expected disabled provider, got %T", phoneProvider)
	}
}
