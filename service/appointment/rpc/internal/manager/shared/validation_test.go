package shared

import (
	"testing"

	"hospital/common/authn"
)

func TestRequirePatientViewAllowsPatientAndAuthorizedStaff(t *testing.T) {
	principals := []authn.Principal{
		{AccountID: "10000000-0000-4000-8000-000000000001", AccountType: authn.AccountTypePatient, Status: authn.AccountStatusActive},
		{AccountID: "10000000-0000-4000-8000-000000000002", AccountType: authn.AccountTypeStaff, Status: authn.AccountStatusActive, Roles: []string{authn.RoleDepartmentDoctor}},
		{AccountID: "10000000-0000-4000-8000-000000000003", AccountType: authn.AccountTypeStaff, Status: authn.AccountStatusActive, Roles: []string{authn.RoleSuperAdmin}},
	}

	for _, principal := range principals {
		if err := requirePatientView(principal); err != nil {
			t.Fatalf("expected principal %s to have patient-view access: %v", principal.AccountID, err)
		}
	}
}

func TestRequirePatientViewRejectsUnprivilegedStaff(t *testing.T) {
	principal := authn.Principal{
		AccountID:   "10000000-0000-4000-8000-000000000004",
		AccountType: authn.AccountTypeStaff,
		Status:      authn.AccountStatusActive,
	}
	if err := requirePatientView(principal); err == nil {
		t.Fatal("expected unprivileged staff to be rejected")
	}
}
