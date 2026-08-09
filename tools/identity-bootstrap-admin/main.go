package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

func main() {
	accountID := flag.String("account-id", "", "existing Identity account UUID to promote")
	flag.Parse()
	if _, err := uuid.Parse(*accountID); err != nil {
		fatal("--account-id must be a valid UUID")
	}
	dsn := os.Getenv("IDENTITY_MYSQL_DSN")
	if dsn == "" {
		fatal("IDENTITY_MYSQL_DSN is required")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fatal(err.Error())
	}
	defer db.Close()
	if err := bootstrap(context.Background(), db, *accountID); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("bootstrapped first super administrator: %s\n", *accountID)
}

func bootstrap(ctx context.Context, db *sql.DB, accountID string) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existingAdmins int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM identity_account_roles ar
JOIN identity_roles r ON r.id = ar.role_id
WHERE r.code = 'super_admin'`).Scan(&existingAdmins); err != nil {
		return fmt.Errorf("count existing super administrators: %w", err)
	}
	if existingAdmins != 0 {
		return fmt.Errorf("a super administrator already exists; bootstrap is disabled")
	}

	var accountType, status string
	var version int64
	if err := tx.QueryRowContext(ctx, `
SELECT account_type, status, authorization_version
FROM identity_accounts
WHERE id = ?
FOR UPDATE`, accountID).Scan(&accountType, &status, &version); err != nil {
		return fmt.Errorf("find bootstrap account: %w", err)
	}
	if status != "active" {
		return fmt.Errorf("bootstrap account must be active")
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE identity_accounts
SET account_type = 'staff', authorization_version = authorization_version + 1,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?`, accountID); err != nil {
		return fmt.Errorf("promote bootstrap account: %w", err)
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO identity_account_roles (account_id, role_id)
SELECT ?, id FROM identity_roles WHERE code = 'super_admin' AND status = 'active'`, accountID)
	if err != nil {
		return fmt.Errorf("assign bootstrap role: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return fmt.Errorf("active super_admin role is missing")
	}

	operationID := uuid.NewString()
	before, _ := json.Marshal(map[string]any{
		"account_id": accountID, "account_type": accountType, "status": status,
		"authorization_version": version,
	})
	after, _ := json.Marshal(map[string]any{
		"account_id": accountID, "account_type": "staff", "status": status,
		"authorization_version": version + 1, "roles": []string{"super_admin"},
	})
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_authorization_audit
    (id, operation_id, operator_account_id, target_account_id, action, before_data, after_data, request_id)
VALUES (?, ?, ?, ?, 'identity.super_admin.bootstrapped', ?, ?, 'bootstrap')`,
		uuid.NewString(), operationID, accountID, accountID, before, after); err != nil {
		return fmt.Errorf("record bootstrap audit: %w", err)
	}
	payload, _ := json.Marshal(map[string]any{
		"operation_id": operationID, "target_account_id": accountID,
		"action": "identity.super_admin.bootstrapped", "authorization_version": version + 1,
	})
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_outbox_events
    (event_id, aggregate_id, event_type, schema_version, payload, occurred_at)
VALUES (?, ?, 'identity.authorization.changed.v1', 1, ?, ?)`,
		uuid.NewString(), accountID, payload, time.Now().UTC()); err != nil {
		return fmt.Errorf("record bootstrap event: %w", err)
	}
	return tx.Commit()
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
