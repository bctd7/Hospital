// 管理员初始化命令在空 Identity 库中幂等创建医院根节点，并授予配置手机号超级管理员角色。
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	contractevents "hospital/contracts/events"
)

const (
	adminPhonesEnv   = "IDENTITY_BOOTSTRAP_ADMIN_PHONES"
	phoneLookupEnv   = "IDENTITY_PHONE_LOOKUP_KEY_BASE64"
	identityDSNEnv   = "IDENTITY_MYSQL_DSN"
	hospitalCodeEnv  = "IDENTITY_BOOTSTRAP_HOSPITAL_CODE"
	hospitalNameEnv  = "IDENTITY_BOOTSTRAP_HOSPITAL_NAME"
	bootstrapAction  = "identity.super_admin.bootstrapped"
	minimumKeyLength = 32
)

var mainlandPhone = regexp.MustCompile(`^1[3-9][0-9]{9}$`)

type adminSeed struct {
	maskedPhone string
	fingerprint []byte
}

func main() {
	phones, err := parseAdminPhones(os.Getenv(adminPhonesEnv))
	if err != nil {
		fatal(err.Error())
	}
	phoneKey, err := decodePhoneLookupKey(os.Getenv(phoneLookupEnv))
	if err != nil {
		fatal(err.Error())
	}
	dsn := strings.TrimSpace(os.Getenv(identityDSNEnv))
	if dsn == "" {
		fatal(identityDSNEnv + " is required")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fatal(err.Error())
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		fatal(fmt.Sprintf("ping identity mysql: %v", err))
	}
	hospitalID, hospitalCreated, err := bootstrapHospital(
		ctx,
		db,
		strings.TrimSpace(os.Getenv(hospitalCodeEnv)),
		strings.TrimSpace(os.Getenv(hospitalNameEnv)),
	)
	if err != nil {
		fatal(fmt.Sprintf("bootstrap hospital root: %v", err))
	}
	hospitalState := "already configured"
	if hospitalCreated {
		hospitalState = "created"
	}
	fmt.Printf("hospital root %s: unit_id=%s\n", hospitalState, hospitalID)

	for _, phone := range phones {
		seed := newAdminSeed(phone, phoneKey)
		accountID, changed, err := bootstrapAdmin(ctx, db, seed)
		if err != nil {
			fatal(fmt.Sprintf("bootstrap administrator %s: %v", seed.maskedPhone, err))
		}
		state := "already configured"
		if changed {
			state = "created"
		}
		fmt.Printf("super administrator %s: account_id=%s phone=%s\n", state, accountID, seed.maskedPhone)
	}
}

func bootstrapHospital(
	ctx context.Context,
	db *sql.DB,
	code, name string,
) (unitID string, created bool, err error) {
	if code == "" {
		return "", false, fmt.Errorf("%s is required", hospitalCodeEnv)
	}
	if name == "" {
		return "", false, fmt.Errorf("%s is required", hospitalNameEnv)
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return "", false, fmt.Errorf("begin hospital bootstrap transaction: %w", err)
	}
	defer tx.Rollback()

	var existingCode, existingName, status string
	err = tx.QueryRowContext(ctx, `
SELECT id, code, name, status
FROM identity_organization_units
WHERE unit_type = 'hospital'
ORDER BY id
LIMIT 1
FOR UPDATE`).Scan(&unitID, &existingCode, &existingName, &status)
	if err == nil {
		if status != "active" {
			return "", false, errors.New("existing hospital root must be active")
		}
		var roots int
		if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM identity_organization_units WHERE unit_type = 'hospital'`).Scan(&roots); err != nil {
			return "", false, fmt.Errorf("count hospital roots: %w", err)
		}
		if roots != 1 {
			return "", false, fmt.Errorf("expected one hospital root, found %d", roots)
		}
		if err := tx.Commit(); err != nil {
			return "", false, fmt.Errorf("commit idempotent hospital bootstrap: %w", err)
		}
		return unitID, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", false, fmt.Errorf("find hospital root: %w", err)
	}

	unitID = uuid.NewString()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_organization_units
    (id, parent_id, unit_type, code, name, status, version)
VALUES (?, NULL, 'hospital', ?, ?, 'active', 1)`, unitID, code, name); err != nil {
		return "", false, fmt.Errorf("create hospital root: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", false, fmt.Errorf("commit hospital bootstrap: %w", err)
	}
	return unitID, true, nil
}

func parseAdminPhones(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	phones := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		normalized, err := normalizePhone(part)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[normalized]; exists {
			return nil, fmt.Errorf("%s contains a duplicate phone", adminPhonesEnv)
		}
		seen[normalized] = struct{}{}
		phones = append(phones, normalized)
	}
	if len(phones) == 0 {
		return nil, fmt.Errorf("%s must contain at least one phone", adminPhonesEnv)
	}
	return phones, nil
}

func normalizePhone(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer(" ", "", "-", "").Replace(value)
	value = strings.TrimPrefix(value, "+86")
	if !mainlandPhone.MatchString(value) {
		return "", errors.New("bootstrap administrator phone must be a mainland mobile number")
	}
	return "+86" + value, nil
}

func decodePhoneLookupKey(raw string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", phoneLookupEnv, err)
	}
	if len(key) < minimumKeyLength {
		return nil, fmt.Errorf("%s must decode to at least %d bytes", phoneLookupEnv, minimumKeyLength)
	}
	return key, nil
}

func newAdminSeed(normalizedPhone string, phoneKey []byte) adminSeed {
	digest := hmac.New(sha256.New, phoneKey)
	_, _ = digest.Write([]byte(normalizedPhone))
	digits := strings.TrimPrefix(normalizedPhone, "+86")
	return adminSeed{
		maskedPhone: digits[:3] + "****" + digits[7:],
		fingerprint: digest.Sum(nil),
	}
}

func bootstrapAdmin(ctx context.Context, db *sql.DB, seed adminSeed) (accountID string, changed bool, err error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return "", false, fmt.Errorf("begin bootstrap transaction: %w", err)
	}
	defer tx.Rollback()

	var accountType, status, roleCode string
	var authorizationVersion int64
	err = tx.QueryRowContext(ctx, `
SELECT a.id, a.account_type, a.status, a.authorization_version, COALESCE(r.code, '')
FROM identity_account_phones ph
JOIN identity_accounts a ON a.id = ph.account_id
LEFT JOIN identity_account_roles ar ON ar.account_id = a.id
LEFT JOIN identity_roles r ON r.id = ar.role_id
WHERE ph.phone_fingerprint = ?
FOR UPDATE`, seed.fingerprint).Scan(
		&accountID, &accountType, &status, &authorizationVersion, &roleCode,
	)
	newAccount := errors.Is(err, sql.ErrNoRows)
	if err != nil && !newAccount {
		return "", false, fmt.Errorf("find bootstrap account: %w", err)
	}

	if newAccount {
		accountID = uuid.NewString()
		accountType = "staff"
		status = "active"
		authorizationVersion = 1
		if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_accounts (id, account_type, status, authorization_version)
VALUES (?, 'staff', 'active', 1)`, accountID); err != nil {
			return "", false, fmt.Errorf("create bootstrap account: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_account_phones
    (account_id, phone_fingerprint, phone_masked, verification_status, verification_source, verified_at)
VALUES (?, ?, ?, 'verified', 'admin', CURRENT_TIMESTAMP(3))`,
			accountID, seed.fingerprint, seed.maskedPhone,
		); err != nil {
			return "", false, fmt.Errorf("create bootstrap phone binding: %w", err)
		}
	} else {
		if status != "active" {
			return "", false, errors.New("bootstrap account must be active")
		}
		if roleCode == "super_admin" {
			if accountType != "staff" {
				return "", false, errors.New("existing super administrator must be a staff account")
			}
			if err := tx.Commit(); err != nil {
				return "", false, fmt.Errorf("commit idempotent bootstrap: %w", err)
			}
			return accountID, false, nil
		}
		if roleCode != "" {
			return "", false, fmt.Errorf("bootstrap account already has role %q", roleCode)
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE identity_accounts
SET account_type = 'staff',
    authorization_version = authorization_version + 1,
    management_version = management_version + 1,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?`, accountID); err != nil {
			return "", false, fmt.Errorf("promote bootstrap account: %w", err)
		}
		authorizationVersion++
	}

	result, err := tx.ExecContext(ctx, `
INSERT INTO identity_account_roles (account_id, role_id)
SELECT ?, id FROM identity_roles WHERE code = 'super_admin' AND status = 'active'`, accountID)
	if err != nil {
		return "", false, fmt.Errorf("assign bootstrap role: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return "", false, errors.New("active super_admin role is missing")
	}

	operationID := uuid.NewString()
	var before any
	if !newAccount {
		before = map[string]any{
			"account_id": accountID, "account_type": accountType, "status": status,
			"authorization_version": authorizationVersion - 1,
		}
	}
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(map[string]any{
		"account_id": accountID, "account_type": "staff", "status": "active",
		"authorization_version": authorizationVersion, "roles": []string{"super_admin"},
	})
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_authorization_audit
    (id, operation_id, operator_account_id, target_account_id, action, before_data, after_data, request_id)
VALUES (?, ?, ?, ?, ?, ?, ?, 'deployment-bootstrap')`,
		uuid.NewString(), operationID, accountID, accountID, bootstrapAction, beforeJSON, afterJSON,
	); err != nil {
		return "", false, fmt.Errorf("record bootstrap audit: %w", err)
	}
	payload, _ := json.Marshal(contractevents.IdentityAuthorizationChangedV1Payload{
		OperationID: operationID, Action: bootstrapAction, AuthorizationVersion: authorizationVersion,
	})
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_outbox_events
    (event_id, aggregate_id, event_type, schema_version, payload, occurred_at)
VALUES (?, ?, ?, 1, ?, ?)`,
		uuid.NewString(), accountID, contractevents.EventTypeIdentityAuthorizationChangedV1, payload, time.Now().UTC(),
	); err != nil {
		return "", false, fmt.Errorf("record bootstrap event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", false, fmt.Errorf("commit bootstrap administrator: %w", err)
	}
	return accountID, true, nil
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
