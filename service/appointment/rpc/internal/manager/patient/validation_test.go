package patient

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"hospital/common/authn"
)

func TestManagerAcceptsPatientIdentityWithoutStaffRole(t *testing.T) {
	err := requirePatient(authn.Principal{AccountID: uuid.NewString(), AccountType: authn.AccountTypePatient, Status: authn.AccountStatusActive})
	if err != nil {
		t.Fatalf("patient read rejected: %v", err)
	}
}

func TestManagerLetsStaffRolesUsePatientCapabilities(t *testing.T) {
	for _, role := range []string{authn.RoleDepartmentDoctor, authn.RoleSuperAdmin} {
		principal := authn.Principal{
			AccountID: uuid.NewString(), AccountType: authn.AccountTypeStaff,
			Status: authn.AccountStatusActive, Roles: []string{role},
		}
		if err := requirePatient(principal); err != nil {
			t.Fatalf("staff role %s rejected patient capability: %v", role, err)
		}
	}
}

func TestManagerRejectsNonUserAndInactivePatientCapabilities(t *testing.T) {
	tests := []authn.Principal{
		{AccountID: uuid.NewString(), AccountType: authn.AccountTypeService, Status: authn.AccountStatusActive},
		{AccountID: uuid.NewString(), AccountType: authn.AccountTypeStaff, Status: authn.AccountStatusActive},
		{AccountID: uuid.NewString(), AccountType: authn.AccountTypePatient, Status: authn.AccountStatusDisabled},
	}
	for _, principal := range tests {
		if err := requirePatient(principal); !errors.Is(err, ErrForbidden) {
			t.Fatalf("principal %+v error = %v, want ErrForbidden", principal, err)
		}
	}
}
