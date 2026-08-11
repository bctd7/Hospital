package svc

import (
	"slices"
	"testing"

	identityv1 "hospital/contracts/gen/identity/v1"
)

func TestSensitiveIdentityRPCMethodsIncludeAdminPhoneSearch(t *testing.T) {
	method := identityv1.IdentityService_SearchAdminAccountByPhone_FullMethodName
	if !slices.Contains(identityRPCMethodsWithSensitiveContent(), method) {
		t.Fatalf("sensitive RPC method %q is missing from app-api client log suppression", method)
	}
}
