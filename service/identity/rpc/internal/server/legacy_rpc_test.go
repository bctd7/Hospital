package server

import (
	"testing"

	identityv1 "hospital/contracts/gen/identity/v1"
)

func TestLegacyAuthorizationWriteRPCsAreNotRegistered(t *testing.T) {
	legacyMethods := map[string]struct{}{
		"PromoteToDepartmentDoctor": {},
		"AssignRole":                {},
		"ChangeStaffDepartment":     {},
		"ChangeAccountStatus":       {},
	}
	for _, method := range identityv1.IdentityService_ServiceDesc.Methods {
		if _, legacy := legacyMethods[method.MethodName]; legacy {
			t.Errorf("legacy authorization write RPC %s is still registered", method.MethodName)
		}
	}
}
