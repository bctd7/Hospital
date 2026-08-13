package mysqlstore

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"os"
	"testing"

	"hospital/common/authn"
	contractevents "hospital/contracts/events"
	"hospital/service/identity/rpc/internal/account"
	accountmanager "hospital/service/identity/rpc/internal/account/manager"
)

const (
	identityAdminTestAdminID     = "20000000-0000-0000-0000-000000000001"
	identityAdminTestPatientID   = "20000000-0000-0000-0000-000000000002"
	identityAdminTestHospitalID  = "20000000-0000-0000-0000-000000000010"
	identityAdminTestCampusID    = "20000000-0000-0000-0000-000000000011"
	identityAdminTestDepartmentA = "20000000-0000-0000-0000-000000000012"
	identityAdminTestDepartmentB = "20000000-0000-0000-0000-000000000013"
	identityAdminTestPromoteOp   = "20000000-0000-0000-0000-000000000020"
	identityAdminTestUpdateOp    = "20000000-0000-0000-0000-000000000021"
	identityAdminTestTransferOp  = "20000000-0000-0000-0000-000000000022"
	identityAdminTestDisableOp   = "20000000-0000-0000-0000-000000000023"
	identityAdminTestEnableOp    = "20000000-0000-0000-0000-000000000024"
	identityAdminTestRevokeOp    = "20000000-0000-0000-0000-000000000025"
	identityAdminTestPhone       = "+8613812345678"
	identityAdminTestMaskedPhone = "138****5678"
)

func TestMySQLAccountManagerLifecycle(t *testing.T) {
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
	phoneKey := []byte("identity-admin-integration-key-32-bytes")
	defer cleanupAccountManagerTestData(t, store, ctx)
	seedAccountManagerTestData(t, store, ctx, phoneKey)

	admin, err := store.GetPrincipal(ctx, identityAdminTestAdminID)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := accountmanager.NewManager(store, phoneKey)
	if err != nil {
		t.Fatal(err)
	}

	profile, err := manager.UpdateDisplayProfile(ctx, authn.Principal{AccountID: identityAdminTestPatientID}, "患者甲")
	if err != nil {
		t.Fatal(err)
	}
	if profile.Nickname != "患者甲" || profile.ManagementVersion != 2 {
		t.Fatalf("unexpected display profile: %#v", profile)
	}

	page, err := manager.ListAccounts(ctx, admin, account.AccountFilter{Nickname: "患者", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != identityAdminTestPatientID {
		t.Fatalf("unexpected account page: %#v", page)
	}
	byPhone, maskedPhone, verificationStatus, verificationSource, err := manager.SearchByPhone(ctx, admin, "138 1234 5678")
	if err != nil {
		t.Fatal(err)
	}
	if byPhone.ID != identityAdminTestPatientID || maskedPhone != identityAdminTestMaskedPhone || verificationStatus != "verified" || verificationSource != "sms" {
		t.Fatalf("unexpected phone search result: account=%#v phone=%s status=%s source=%s", byPhone, maskedPhone, verificationStatus, verificationSource)
	}

	managedAccount, _, err := manager.PromoteDoctor(ctx, admin, identityAdminTestPatientID, identityAdminTestDepartmentA,
		account.DoctorProfileInput{DisplayName: "张医生", StaffNo: "D-001", Description: "影像诊断"},
		2, true, identityAdminTestPromoteOp, "identity-admin-integration")
	if err != nil {
		t.Fatal(err)
	}
	assertManagedAccount(t, managedAccount, account.IdentityTypeDoctor, authn.AccountStatusActive, identityAdminTestDepartmentA, 3, 2)

	doctors, err := manager.ListDoctors(ctx, identityAdminTestDepartmentA, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if doctors.Total != 1 || len(doctors.Items) != 1 || doctors.Items[0].DisplayName != "张医生" {
		t.Fatalf("unexpected doctor directory after promotion: %#v", doctors)
	}

	displayName := "张主任"
	description := "影像诊断与复核"
	managedAccount, _, err = manager.UpdateDoctor(ctx, admin, identityAdminTestPatientID,
		account.OptionalDoctorProfileInput{DisplayName: &displayName, Description: &description},
		3, identityAdminTestUpdateOp, "identity-admin-integration")
	if err != nil {
		t.Fatal(err)
	}
	assertManagedAccount(t, managedAccount, account.IdentityTypeDoctor, authn.AccountStatusActive, identityAdminTestDepartmentA, 4, 2)
	if managedAccount.DisplayName != displayName || managedAccount.Description != description {
		t.Fatalf("doctor profile was not updated: %#v", managedAccount)
	}

	managedAccount, _, err = manager.ChangeDoctorDepartment(ctx, admin, identityAdminTestPatientID, identityAdminTestDepartmentB,
		4, identityAdminTestTransferOp, "identity-admin-integration")
	if err != nil {
		t.Fatal(err)
	}
	assertManagedAccount(t, managedAccount, account.IdentityTypeDoctor, authn.AccountStatusActive, identityAdminTestDepartmentB, 5, 3)

	managedAccount, _, err = manager.SetAccountEnabled(ctx, admin, identityAdminTestPatientID, false,
		5, identityAdminTestDisableOp, "identity-admin-integration")
	if err != nil {
		t.Fatal(err)
	}
	assertManagedAccount(t, managedAccount, account.IdentityTypeDoctor, authn.AccountStatusDisabled, identityAdminTestDepartmentB, 6, 4)
	doctors, err = manager.ListDoctors(ctx, identityAdminTestDepartmentB, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if doctors.Total != 0 {
		t.Fatalf("disabled doctor remained in public directory: %#v", doctors)
	}

	managedAccount, _, err = manager.SetAccountEnabled(ctx, admin, identityAdminTestPatientID, true,
		6, identityAdminTestEnableOp, "identity-admin-integration")
	if err != nil {
		t.Fatal(err)
	}
	assertManagedAccount(t, managedAccount, account.IdentityTypeDoctor, authn.AccountStatusActive, identityAdminTestDepartmentB, 7, 5)

	managedAccount, _, err = manager.RevokeDoctor(ctx, admin, identityAdminTestPatientID,
		7, identityAdminTestRevokeOp, "identity-admin-integration")
	if err != nil {
		t.Fatal(err)
	}
	assertManagedAccount(t, managedAccount, account.IdentityTypePatient, authn.AccountStatusActive, identityAdminTestDepartmentB, 8, 6)
	departmentAccounts, err := manager.ListAccounts(ctx, admin, account.AccountFilter{
		DepartmentID: identityAdminTestDepartmentB, Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if departmentAccounts.Total != 0 {
		t.Fatalf("revoked doctor remained in the current department filter: %#v", departmentAccounts)
	}

	replayed, _, err := manager.RevokeDoctor(ctx, admin, identityAdminTestPatientID,
		7, identityAdminTestRevokeOp, "identity-admin-integration")
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ManagementVersion != 8 {
		t.Fatalf("idempotent replay changed state: account=%#v", replayed)
	}

	assertCount(t, store, ctx, "identity_authorization_audit", "target_account_id", identityAdminTestPatientID, 6)
	assertCount(t, store, ctx, "identity_outbox_events", "aggregate_id", identityAdminTestPatientID, 5)
	assertIdentityAuthorizationOutboxEvents(t, store, ctx)
}

func assertManagedAccount(t *testing.T, managedAccount account.Account, identityType, status, departmentID string, managementVersion, authorizationVersion int64) {
	t.Helper()
	if managedAccount.IdentityType() != identityType || managedAccount.AccountStatus != status || managedAccount.DepartmentID != departmentID ||
		managedAccount.ManagementVersion != managementVersion || managedAccount.AuthorizationVersion != authorizationVersion {
		t.Fatalf("unexpected managed account: %#v", managedAccount)
	}
}

func assertIdentityAuthorizationOutboxEvents(t *testing.T, store *Store, ctx context.Context) {
	t.Helper()
	wantVersions := map[string]int64{
		identityAdminTestPromoteOp:  2,
		identityAdminTestTransferOp: 3,
		identityAdminTestDisableOp:  4,
		identityAdminTestEnableOp:   5,
		identityAdminTestRevokeOp:   6,
	}
	rows, err := store.db.QueryContext(ctx, `
SELECT event_type, schema_version, payload
FROM identity_outbox_events
WHERE aggregate_id = ?`, identityAdminTestPatientID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	got := make(map[string]int64, len(wantVersions))
	for rows.Next() {
		var eventType string
		var schemaVersion int
		var payloadData []byte
		if err := rows.Scan(&eventType, &schemaVersion, &payloadData); err != nil {
			t.Fatal(err)
		}
		if eventType != contractevents.EventTypeIdentityAuthorizationChangedV1 || schemaVersion != 1 {
			t.Fatalf("unexpected identity authorization event contract: type=%q schema_version=%d", eventType, schemaVersion)
		}
		var payload contractevents.IdentityAuthorizationChangedV1Payload
		if err := json.Unmarshal(payloadData, &payload); err != nil {
			t.Fatalf("decode identity authorization event payload: %v", err)
		}
		got[payload.OperationID] = payload.AuthorizationVersion
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(wantVersions) {
		t.Fatalf("unexpected identity authorization event count: got=%v want=%v", got, wantVersions)
	}
	for operationID, wantVersion := range wantVersions {
		if got[operationID] != wantVersion {
			t.Fatalf("unexpected authorization version for operation %s: got=%d want=%d", operationID, got[operationID], wantVersion)
		}
	}
}

func seedAccountManagerTestData(t *testing.T, store *Store, ctx context.Context, phoneKey []byte) {
	t.Helper()
	fingerprint := hmac.New(sha256.New, phoneKey)
	_, _ = fingerprint.Write([]byte(identityAdminTestPhone))
	statements := []struct {
		query string
		args  []any
	}{
		{"INSERT INTO identity_organization_units (id, parent_id, unit_type, code, name) VALUES (?, NULL, 'hospital', 'crud-hospital', 'CRUD Hospital')", []any{identityAdminTestHospitalID}},
		{"INSERT INTO identity_organization_units (id, parent_id, unit_type, code, name) VALUES (?, ?, 'campus', 'crud-campus', 'CRUD Campus')", []any{identityAdminTestCampusID, identityAdminTestHospitalID}},
		{"INSERT INTO identity_organization_units (id, parent_id, unit_type, code, name) VALUES (?, ?, 'department', 'crud-a', 'CRUD A')", []any{identityAdminTestDepartmentA, identityAdminTestCampusID}},
		{"INSERT INTO identity_organization_units (id, parent_id, unit_type, code, name) VALUES (?, ?, 'department', 'crud-b', 'CRUD B')", []any{identityAdminTestDepartmentB, identityAdminTestCampusID}},
		{"INSERT INTO identity_accounts (id, account_type, status) VALUES (?, 'staff', 'active')", []any{identityAdminTestAdminID}},
		{"INSERT INTO identity_staff_profiles (account_id, department_id, display_name) VALUES (?, ?, '管理员')", []any{identityAdminTestAdminID, identityAdminTestDepartmentA}},
		{"INSERT INTO identity_account_roles (account_id, role_id) SELECT ?, id FROM identity_roles WHERE code = 'super_admin'", []any{identityAdminTestAdminID}},
		{"INSERT INTO identity_accounts (id, account_type, status) VALUES (?, 'patient', 'active')", []any{identityAdminTestPatientID}},
		{"INSERT INTO identity_account_phones (account_id, phone_fingerprint, phone_masked, verification_status, verification_source, verified_at) VALUES (?, ?, ?, 'verified', 'sms', CURRENT_TIMESTAMP(3))", []any{identityAdminTestPatientID, fingerprint.Sum(nil), identityAdminTestMaskedPhone}},
	}
	for _, statement := range statements {
		if _, err := store.db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("seed identity admin integration data: %v", err)
		}
	}
}

func cleanupAccountManagerTestData(t *testing.T, store *Store, ctx context.Context) {
	t.Helper()
	statements := []struct {
		query string
		args  []any
	}{
		{"DELETE FROM identity_outbox_events WHERE aggregate_id = ?", []any{identityAdminTestPatientID}},
		{"DELETE FROM identity_authorization_audit WHERE target_account_id = ?", []any{identityAdminTestPatientID}},
		{"DELETE FROM identity_account_profiles WHERE account_id = ?", []any{identityAdminTestPatientID}},
		{"DELETE FROM identity_account_roles WHERE account_id IN (?, ?)", []any{identityAdminTestAdminID, identityAdminTestPatientID}},
		{"DELETE FROM identity_staff_profiles WHERE account_id IN (?, ?)", []any{identityAdminTestAdminID, identityAdminTestPatientID}},
		{"DELETE FROM identity_account_phones WHERE account_id = ?", []any{identityAdminTestPatientID}},
		{"DELETE FROM identity_accounts WHERE id IN (?, ?)", []any{identityAdminTestAdminID, identityAdminTestPatientID}},
		{"DELETE FROM identity_organization_units WHERE id IN (?, ?)", []any{identityAdminTestDepartmentA, identityAdminTestDepartmentB}},
		{"DELETE FROM identity_organization_units WHERE id = ?", []any{identityAdminTestCampusID}},
		{"DELETE FROM identity_organization_units WHERE id = ?", []any{identityAdminTestHospitalID}},
	}
	for _, statement := range statements {
		if _, err := store.db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Errorf("cleanup identity admin integration data: %v", err)
		}
	}
}
