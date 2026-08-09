package repository

import (
	"context"
	"os"
	"testing"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	"hospital/service/identity/rpc/internal/authorization"
)

const (
	integrationAdminID     = "10000000-0000-0000-0000-000000000001"
	integrationDoctorID    = "10000000-0000-0000-0000-000000000002"
	integrationDepartmentA = "10000000-0000-0000-0000-000000000010"
	integrationDepartmentB = "10000000-0000-0000-0000-000000000011"
	integrationOperationID = "10000000-0000-0000-0000-000000000020"
)

func TestMySQLAuthorizationChange(t *testing.T) {
	dataSource := os.Getenv("IDENTITY_TEST_MYSQL_DSN")
	if dataSource == "" {
		t.Skip("IDENTITY_TEST_MYSQL_DSN is not set")
	}

	store, err := NewMySQLStore(dataSource)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	seedIdentityTestData(t, store, ctx)
	defer cleanupIdentityTestData(t, store, ctx)

	manager := authorization.NewManager(store)
	result, err := manager.ChangeStaffDepartment(
		ctx, integrationAdminID, integrationDoctorID, integrationDepartmentB,
		integrationOperationID, "integration-request",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.DepartmentID != integrationDepartmentB || result.AuthorizationVersion != 2 {
		t.Fatalf("unexpected authorization context: %#v", result)
	}

	replayed, err := manager.ChangeStaffDepartment(
		ctx, integrationAdminID, integrationDoctorID, integrationDepartmentB,
		integrationOperationID, "integration-request",
	)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.AuthorizationVersion != 2 {
		t.Fatalf("idempotent replay changed authorization version to %d", replayed.AuthorizationVersion)
	}

	assertCount(t, store, ctx, "identity_authorization_audit", "operation_id", integrationOperationID, 1)
	assertCount(t, store, ctx, "identity_outbox_events", "aggregate_id", integrationDoctorID, 1)
}

func seedIdentityTestData(t *testing.T, store *MySQLStore, ctx context.Context) {
	t.Helper()
	statements := []struct {
		query string
		args  []any
	}{
		{"INSERT INTO identity_departments (id, code, name) VALUES (?, 'integration-a', 'Integration A')", []any{integrationDepartmentA}},
		{"INSERT INTO identity_departments (id, code, name) VALUES (?, 'integration-b', 'Integration B')", []any{integrationDepartmentB}},
		{"INSERT INTO identity_accounts (id, account_type, status) VALUES (?, 'staff', 'active')", []any{integrationAdminID}},
		{"INSERT INTO identity_accounts (id, account_type, status) VALUES (?, 'staff', 'active')", []any{integrationDoctorID}},
		{"INSERT INTO identity_staff_profiles (account_id, department_id) VALUES (?, ?)", []any{integrationAdminID, integrationDepartmentA}},
		{"INSERT INTO identity_staff_profiles (account_id, department_id) VALUES (?, ?)", []any{integrationDoctorID, integrationDepartmentA}},
		{"INSERT INTO identity_account_roles (account_id, role_id) SELECT ?, id FROM identity_roles WHERE code = ?", []any{integrationAdminID, authn.RoleSuperAdmin}},
		{"INSERT INTO identity_account_roles (account_id, role_id) SELECT ?, id FROM identity_roles WHERE code = ?", []any{integrationDoctorID, authn.RoleDepartmentDoctor}},
	}
	for _, statement := range statements {
		if _, err := store.db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("seed identity integration data: %v", err)
		}
	}

	admin, err := store.GetAuthorizationContext(ctx, integrationAdminID)
	if err != nil {
		t.Fatal(err)
	}
	if !admin.HasPermission(contractauthz.PermissionIdentityAuthorizationManage) {
		t.Fatal("seeded administrator is missing authorization management permission")
	}
}

func cleanupIdentityTestData(t *testing.T, store *MySQLStore, ctx context.Context) {
	t.Helper()
	statements := []struct {
		query string
		args  []any
	}{
		{"DELETE FROM identity_outbox_events WHERE aggregate_id IN (?, ?)", []any{integrationAdminID, integrationDoctorID}},
		{"DELETE FROM identity_authorization_audit WHERE target_account_id IN (?, ?)", []any{integrationAdminID, integrationDoctorID}},
		{"DELETE FROM identity_account_roles WHERE account_id IN (?, ?)", []any{integrationAdminID, integrationDoctorID}},
		{"DELETE FROM identity_staff_profiles WHERE account_id IN (?, ?)", []any{integrationAdminID, integrationDoctorID}},
		{"DELETE FROM identity_accounts WHERE id IN (?, ?)", []any{integrationAdminID, integrationDoctorID}},
		{"DELETE FROM identity_departments WHERE id IN (?, ?)", []any{integrationDepartmentA, integrationDepartmentB}},
	}
	for _, statement := range statements {
		if _, err := store.db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Errorf("cleanup identity integration data: %v", err)
		}
	}
}

func assertCount(t *testing.T, store *MySQLStore, ctx context.Context, table, column, value string, want int) {
	t.Helper()
	query := "SELECT COUNT(*) FROM " + table + " WHERE " + column + " = ?" // table and column are test constants
	var got int
	if err := store.db.QueryRowContext(ctx, query, value).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}
