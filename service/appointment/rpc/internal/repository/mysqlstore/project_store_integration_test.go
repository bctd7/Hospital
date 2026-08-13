package mysqlstore

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	appointmentmanager "hospital/service/appointment/rpc/internal/manager"
)

const (
	catalogIntegrationOperatorID       = "30000000-0000-0000-0000-000000000001"
	catalogIntegrationDepartment       = "30000000-0000-0000-0000-000000000010"
	catalogIntegrationCreateOperation  = "30000000-0000-0000-0000-000000000020"
	catalogIntegrationUpdateOperation  = "30000000-0000-0000-0000-000000000021"
	catalogIntegrationDisableOperation = "30000000-0000-0000-0000-000000000022"
	catalogIntegrationEnableOperation  = "30000000-0000-0000-0000-000000000023"
)

func TestMySQLCatalogLifecycle(t *testing.T) {
	dataSource := os.Getenv("APPOINTMENT_TEST_MYSQL_DSN")
	if dataSource == "" {
		t.Skip("APPOINTMENT_TEST_MYSQL_DSN is not set")
	}

	store, err := New(dataSource)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	cleanupCatalogIntegrationData(t, store, ctx)
	defer cleanupCatalogIntegrationData(t, store, ctx)

	manager, err := appointmentmanager.NewStaffManager(store, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	operator := authn.Principal{
		AccountID:   catalogIntegrationOperatorID,
		AccountType: authn.AccountTypeStaff,
		Status:      authn.AccountStatusActive,
		Roles:       []string{authn.RoleSuperAdmin},
		Permissions: []string{
			contractauthz.PermissionAppointmentCreate,
			contractauthz.PermissionAppointmentRead,
			contractauthz.PermissionAppointmentUpdate,
		},
	}
	command := appointmentmanager.CreateProjectCommand{
		OwnerDepartmentID: catalogIntegrationDepartment,
		Name:              "Integration Examination Item",
		Description:       "Integration preparation description",
		OperationID:       catalogIntegrationCreateOperation,
		RequestID:         "catalog-integration-create",
	}

	created, err := manager.CreateProject(ctx, operator, command)
	if err != nil {
		t.Fatal(err)
	}
	if created.ItemID == "" || created.Status != appointmentmanager.StatusActive || created.Version != 1 {
		t.Fatalf("unexpected created item: %#v", created)
	}

	replayed, err := manager.CreateProject(ctx, operator, command)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ItemID != created.ItemID || !replayed.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("replayed item = %#v, want %#v", replayed, created)
	}

	command.Description = "Different preparation description"
	if _, err := manager.CreateProject(ctx, operator, command); !errors.Is(err, appointmentmanager.ErrConflict) {
		t.Fatalf("operation reuse error = %v, want ErrConflict", err)
	}
	got, err := manager.GetProject(ctx, operator, created.ItemID)
	if err != nil || got.ItemID != created.ItemID {
		t.Fatalf("get item=%#v err=%v", got, err)
	}
	page, err := manager.ListProjects(ctx, operator, appointmentmanager.ListProjectsQuery{
		OwnerDepartmentID: catalogIntegrationDepartment,
		Status:            appointmentmanager.StatusActive,
		Page:              1,
		PageSize:          10,
	})
	if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].ItemID != created.ItemID {
		t.Fatalf("list result=%#v err=%v", page, err)
	}

	name := "Updated Integration Examination Item"
	description := "Updated integration preparation description"
	updated, err := manager.UpdateProject(ctx, operator, appointmentmanager.UpdateProjectCommand{
		ItemID: created.ItemID, Name: &name, Description: &description,
		ExpectedVersion: 1, OperationID: catalogIntegrationUpdateOperation,
		RequestID: "catalog-integration-update",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != name || updated.Description != description || updated.Version != 2 {
		t.Fatalf("unexpected updated item: %#v", updated)
	}

	disabled, err := manager.DisableProject(ctx, operator, appointmentmanager.ChangeProjectStatusCommand{
		ItemID: created.ItemID, ExpectedVersion: 2,
		OperationID: catalogIntegrationDisableOperation,
	})
	if err != nil || disabled.Status != appointmentmanager.StatusDisabled || disabled.Version != 3 {
		t.Fatalf("disabled item=%#v err=%v", disabled, err)
	}
	enabledCommand := appointmentmanager.ChangeProjectStatusCommand{
		ItemID: created.ItemID, ExpectedVersion: 3,
		OperationID: catalogIntegrationEnableOperation,
	}
	enabled, err := manager.EnableProject(ctx, operator, enabledCommand)
	if err != nil || enabled.Status != appointmentmanager.StatusActive || enabled.Version != 4 {
		t.Fatalf("enabled item=%#v err=%v", enabled, err)
	}
	replayedEnable, err := manager.EnableProject(ctx, operator, enabledCommand)
	if err != nil || replayedEnable.Version != enabled.Version {
		t.Fatalf("replayed enable=%#v err=%v", replayedEnable, err)
	}

	var itemCount, operationCount, auditCount int64
	if err := store.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM appointment_examination_items WHERE id = ?`, created.ItemID).Scan(&itemCount); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM appointment_examination_item_operations WHERE item_id = ?`, created.ItemID).Scan(&operationCount); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM appointment_examination_item_audit WHERE item_id = ?`, created.ItemID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if itemCount != 1 || operationCount != 4 || auditCount != 4 {
		t.Fatalf("item/operation/audit counts=%d/%d/%d, want 1/4/4", itemCount, operationCount, auditCount)
	}
	var auditJSON string
	if err := store.db.QueryRowContext(ctx, `
SELECT CAST(after_data AS CHAR)
FROM appointment_examination_item_audit
WHERE operation_id = ?`, catalogIntegrationUpdateOperation).Scan(&auditJSON); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(auditJSON, description) {
		t.Fatal("audit data must not contain the full description")
	}
}

func cleanupCatalogIntegrationData(t *testing.T, store *Store, ctx context.Context) {
	t.Helper()
	if _, err := store.db.ExecContext(ctx, `
DELETE FROM appointment_examination_item_audit WHERE operator_account_id = ?`, catalogIntegrationOperatorID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
DELETE FROM appointment_examination_item_operations WHERE operator_account_id = ?`, catalogIntegrationOperatorID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
DELETE FROM appointment_examination_items WHERE owner_department_id = ?`, catalogIntegrationDepartment); err != nil {
		t.Fatal(err)
	}
}
