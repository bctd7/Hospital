package mysqlstore

import (
	"context"
	"os"
	"testing"

	"hospital/common/authn"
	"hospital/service/identity/rpc/internal/account"
	accountmanager "hospital/service/identity/rpc/internal/account/manager"
)

func TestMySQLPhoneRegistrationAndDoctorPromotion(t *testing.T) {
	dataSource := os.Getenv("IDENTITY_TEST_MYSQL_DSN")
	if dataSource == "" {
		t.Skip("IDENTITY_TEST_MYSQL_DSN is not set")
	}
	store, err := New(dataSource)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	const (
		adminID      = "10000000-0000-0000-0000-000000000101"
		departmentID = "10000000-0000-0000-0000-000000000102"
		operationID  = "10000000-0000-0000-0000-000000000103"
	)
	if _, err := store.db.ExecContext(ctx, `
INSERT INTO identity_organization_units (id, unit_type, code, name)
VALUES (?, 'department', 'login-test', 'Login Test')`, departmentID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, "INSERT INTO identity_accounts (id, account_type, status) VALUES (?, 'staff', 'active')", adminID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, "INSERT INTO identity_account_roles (account_id, role_id) SELECT ?, id FROM identity_roles WHERE code = 'super_admin'", adminID); err != nil {
		t.Fatal(err)
	}
	var patientID string
	defer func() {
		_, _ = store.db.ExecContext(ctx, "DELETE FROM identity_outbox_events WHERE aggregate_id = ?", patientID)
		_, _ = store.db.ExecContext(ctx, "DELETE FROM identity_authorization_audit WHERE target_account_id = ?", patientID)
		_, _ = store.db.ExecContext(ctx, "DELETE FROM identity_account_roles WHERE account_id IN (?, ?)", adminID, patientID)
		_, _ = store.db.ExecContext(ctx, "DELETE FROM identity_staff_profiles WHERE account_id = ?", patientID)
		_, _ = store.db.ExecContext(ctx, "DELETE FROM identity_account_phones WHERE account_id = ?", patientID)
		_, _ = store.db.ExecContext(ctx, "DELETE FROM identity_accounts WHERE id IN (?, ?)", adminID, patientID)
		_, _ = store.db.ExecContext(ctx, "DELETE FROM identity_organization_units WHERE id = ?", departmentID)
	}()

	fingerprint := make([]byte, 32)
	fingerprint[0] = 1
	patientID, err = store.FindOrCreateVerifiedPhoneAccount(ctx, fingerprint, "138****8000")
	if err != nil {
		t.Fatal(err)
	}
	repeatedID, err := store.FindOrCreateVerifiedPhoneAccount(ctx, fingerprint, "138****8000")
	if err != nil || repeatedID != patientID {
		t.Fatalf("phone registration was not idempotent: id=%s err=%v", repeatedID, err)
	}

	admin, err := store.GetPrincipal(ctx, adminID)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := accountmanager.NewManager(store, []byte("integration-phone-lookup-key-32x"))
	if err != nil {
		t.Fatal(err)
	}
	patient, _, err := manager.GetAccount(ctx, admin, patientID)
	if err != nil {
		t.Fatal(err)
	}
	managedAccount, _, err := manager.PromoteDoctor(
		ctx, admin, patientID, departmentID,
		account.DoctorProfileInput{DisplayName: "集成测试医生"},
		patient.ManagementVersion, true, operationID, "integration-login-request",
	)
	if err != nil {
		t.Fatal(err)
	}
	if managedAccount.AccountType != authn.AccountTypeStaff || managedAccount.IdentityType() != account.IdentityTypeDoctor || managedAccount.DepartmentID != departmentID {
		t.Fatalf("unexpected promoted identity: %#v", managedAccount)
	}
	if managedAccount.PhoneVerificationStatus != "verified" || managedAccount.PhoneVerificationSource != "admin" {
		t.Fatalf("phone was not admin verified: %#v", managedAccount)
	}
}

func assertCount(t *testing.T, store *Store, ctx context.Context, table, column, value string, want int) {
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
