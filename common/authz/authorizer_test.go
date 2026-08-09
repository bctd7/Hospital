package authz

import (
	"errors"
	"testing"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
)

func TestDepartmentAuthorization(t *testing.T) {
	doctor := authn.Principal{
		AccountID: "doctor-1", AccountType: authn.AccountTypeStaff, Status: authn.AccountStatusActive,
		Roles: []string{authn.RoleDepartmentDoctor}, DepartmentID: "department-a",
		Permissions: []string{contractauthz.PermissionAppointmentRead, contractauthz.PermissionAppointmentCancel},
	}

	tests := []struct {
		name       string
		permission string
		mode       AccessMode
		department string
		wantErr    error
	}{
		{name: "same department write", permission: contractauthz.PermissionAppointmentCancel, mode: AccessWrite, department: "department-a"},
		{name: "cross department read", permission: contractauthz.PermissionAppointmentRead, mode: AccessRead, department: "department-b"},
		{name: "cross department write", permission: contractauthz.PermissionAppointmentCancel, mode: AccessWrite, department: "department-b", wantErr: ErrScopeDenied},
		{name: "missing permission", permission: contractauthz.PermissionPlanningAdjust, mode: AccessWrite, department: "department-a", wantErr: ErrPermissionDenied},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AuthorizeDepartment(doctor, tt.permission, tt.mode, Resource{Type: "appointment", ID: "resource-1", DepartmentID: tt.department})
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestSuperAdminCanAccessAnyDepartment(t *testing.T) {
	admin := authn.Principal{
		AccountID: "admin-1", Status: authn.AccountStatusActive, Roles: []string{authn.RoleSuperAdmin},
		Permissions: []string{contractauthz.PermissionAppointmentCancel},
	}
	if err := AuthorizeDepartment(admin, contractauthz.PermissionAppointmentCancel, AccessWrite, Resource{DepartmentID: "department-b"}); err != nil {
		t.Fatal(err)
	}
}
