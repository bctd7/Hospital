package mysqlstore

import (
	"context"
	"errors"
	"os"
	"testing"

	"hospital/common/authn"
	"hospital/service/identity/rpc/internal/organization"
)

const (
	organizationTestAdminID          = "20000000-0000-0000-0000-000000000001"
	organizationTestHospitalID       = "20000000-0000-0000-0000-000000000010"
	organizationTestCampusID         = "20000000-0000-0000-0000-000000000011"
	organizationTestCreateOperation  = "20000000-0000-0000-0000-000000000020"
	organizationTestUpdateOperation  = "20000000-0000-0000-0000-000000000021"
	organizationTestDisableOperation = "20000000-0000-0000-0000-000000000022"
	organizationTestEnableOperation  = "20000000-0000-0000-0000-000000000023"
)

func TestMySQLOrganizationStoreLifecycle(t *testing.T) {
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
	seedOrganizationTestData(t, store, ctx)
	defer cleanupOrganizationTestData(t, store, ctx)

	admin, err := store.GetAuthorizationContext(ctx, organizationTestAdminID)
	if err != nil {
		t.Fatal(err)
	}
	manager := organization.NewManager(store)

	created, err := manager.CreateUnit(ctx, admin, organization.CreateUnitCommand{
		Type:        organization.UnitTypeDepartment,
		ParentID:    organizationTestCampusID,
		Name:        "Integration Department",
		OperationID: organizationTestCreateOperation,
		RequestID:   "organization-integration-create",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.ParentID != organizationTestCampusID || created.Version != 1 {
		t.Fatalf("unexpected created organization unit: %#v", created)
	}
	organizationTestDepartmentIDValue := created.ID

	replayed, err := manager.CreateUnit(ctx, admin, organization.CreateUnitCommand{
		Type:        organization.UnitTypeDepartment,
		ParentID:    organizationTestCampusID,
		Name:        "Integration Department",
		OperationID: organizationTestCreateOperation,
		RequestID:   "organization-integration-create-retry",
	})
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != created.ID || replayed.Version != 1 {
		t.Fatalf("organization create replay returned %#v, want %#v", replayed, created)
	}

	campus, err := manager.GetManagedUnit(ctx, admin, organizationTestCampusID)
	if err != nil {
		t.Fatal(err)
	}
	if campus.ChildCount != 1 {
		t.Fatalf("campus child count = %d, want 1", campus.ChildCount)
	}
	if _, err := manager.DisableUnit(ctx, admin, organization.ChangeUnitStatusCommand{
		UnitID:          organizationTestCampusID,
		ExpectedVersion: campus.Version,
		OperationID:     "20000000-0000-0000-0000-000000000024",
	}); !errors.Is(err, organization.ErrActiveChildren) {
		t.Fatalf("disable campus with active child error = %v, want ErrActiveChildren", err)
	}

	newName := "Renamed Integration Department"
	updated, err := manager.UpdateUnit(ctx, admin, organization.UpdateUnitCommand{
		UnitID:          organizationTestDepartmentIDValue,
		Name:            &newName,
		ExpectedVersion: 1,
		OperationID:     organizationTestUpdateOperation,
		RequestID:       "organization-integration-update",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != newName || updated.Version != 2 {
		t.Fatalf("unexpected updated organization unit: %#v", updated)
	}

	disabled, err := manager.DisableUnit(ctx, admin, organization.ChangeUnitStatusCommand{
		UnitID:          organizationTestDepartmentIDValue,
		ExpectedVersion: 2,
		OperationID:     organizationTestDisableOperation,
		RequestID:       "organization-integration-disable",
	})
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Status != organization.StatusDisabled || disabled.Version != 3 {
		t.Fatalf("unexpected disabled organization unit: %#v", disabled)
	}

	enabled, err := manager.EnableUnit(ctx, admin, organization.ChangeUnitStatusCommand{
		UnitID:          organizationTestDepartmentIDValue,
		ExpectedVersion: 3,
		OperationID:     organizationTestEnableOperation,
		RequestID:       "organization-integration-enable",
	})
	if err != nil {
		t.Fatal(err)
	}
	if enabled.Status != organization.StatusActive || enabled.Version != 4 {
		t.Fatalf("unexpected enabled organization unit: %#v", enabled)
	}

	status := organization.StatusActive
	campusID := organizationTestCampusID
	units, err := store.ListUnits(ctx, organization.ListFilter{
		Type:     organization.UnitTypeDepartment,
		ParentID: &campusID,
		Status:   &status,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 1 || units[0].ID != organizationTestDepartmentIDValue {
		t.Fatalf("unexpected organization unit list: %#v", units)
	}

	directory, err := manager.GetDirectoryContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if directory.Hospital.ID != organizationTestHospitalID || len(directory.Campuses) != 1 ||
		directory.Campuses[0].ID != organizationTestCampusID {
		t.Fatalf("unexpected public organization context: %#v", directory)
	}
	directoryDepartments, err := manager.ListDirectoryDepartments(ctx, organizationTestCampusID)
	if err != nil {
		t.Fatal(err)
	}
	if len(directoryDepartments) != 1 || directoryDepartments[0].ID != organizationTestDepartmentIDValue ||
		directoryDepartments[0].CampusName != "Test Campus" {
		t.Fatalf("unexpected public departments: %#v", directoryDepartments)
	}
	managedUnits, err := manager.ListManagedUnits(ctx, admin, organization.ListFilter{
		Type:     organization.UnitTypeDepartment,
		ParentID: &campusID,
		Status:   &status,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(managedUnits) != 1 || managedUnits[0].ID != organizationTestDepartmentIDValue {
		t.Fatalf("unexpected managed organization units: %#v", managedUnits)
	}

	assertCount(t, store, ctx, "identity_organization_audit", "unit_id", organizationTestDepartmentIDValue, 4)
	assertCount(t, store, ctx, "identity_outbox_events", "aggregate_id", organizationTestDepartmentIDValue, 4)
}

func seedOrganizationTestData(t *testing.T, store *Store, ctx context.Context) {
	t.Helper()
	statements := []struct {
		query string
		args  []any
	}{
		{"INSERT INTO identity_accounts (id, account_type, status) VALUES (?, 'staff', 'active')", []any{organizationTestAdminID}},
		{"INSERT INTO identity_account_roles (account_id, role_id) SELECT ?, id FROM identity_roles WHERE code = ?", []any{organizationTestAdminID, authn.RoleSuperAdmin}},
		{"INSERT INTO identity_organization_units (id, parent_id, unit_type, code, name, status, version) VALUES (?, NULL, 'hospital', 'ORG-TEST-HOSPITAL', 'Test Hospital', 'active', 1)", []any{organizationTestHospitalID}},
		{"INSERT INTO identity_organization_units (id, parent_id, unit_type, code, name, status, version) VALUES (?, ?, 'campus', 'ORG-TEST-CAMPUS', 'Test Campus', 'active', 1)", []any{organizationTestCampusID, organizationTestHospitalID}},
	}
	for _, statement := range statements {
		if _, err := store.db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			cleanupOrganizationTestData(t, store, ctx)
			t.Fatalf("seed organization integration data: %v", err)
		}
	}
}

func cleanupOrganizationTestData(t *testing.T, store *Store, ctx context.Context) {
	t.Helper()
	statements := []struct {
		query string
		args  []any
	}{
		{"DELETE FROM identity_outbox_events WHERE aggregate_id IN (SELECT unit_id FROM identity_organization_audit WHERE operator_account_id = ?)", []any{organizationTestAdminID}},
		{"DELETE FROM identity_organization_audit WHERE operator_account_id = ?", []any{organizationTestAdminID}},
		{"DELETE FROM identity_organization_units WHERE parent_id = ?", []any{organizationTestCampusID}},
		{"DELETE FROM identity_organization_units WHERE id = ?", []any{organizationTestCampusID}},
		{"DELETE FROM identity_organization_units WHERE id = ?", []any{organizationTestHospitalID}},
		{"DELETE FROM identity_account_roles WHERE account_id = ?", []any{organizationTestAdminID}},
		{"DELETE FROM identity_accounts WHERE id = ?", []any{organizationTestAdminID}},
	}
	for _, statement := range statements {
		if _, err := store.db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Errorf("cleanup organization integration data: %v", err)
		}
	}
}
