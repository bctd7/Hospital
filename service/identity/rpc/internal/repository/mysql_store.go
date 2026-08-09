package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"hospital/common/authn"
	"hospital/service/identity/rpc/internal/account"
	"hospital/service/identity/rpc/internal/authorization"
)

type MySQLStore struct {
	db *sql.DB
}

func (s *MySQLStore) FindOrCreateWeChatAccount(ctx context.Context, appID, openID string) (string, error) {
	if appID == "" || openID == "" {
		return "", account.ErrInvalidExternalIdentity
	}
	var accountID string
	err := s.db.QueryRowContext(ctx, `
SELECT account_id
FROM identity_external_identities
WHERE provider = 'wechat' AND provider_app_id = ? AND provider_subject = ?`, appID, openID).Scan(&accountID)
	if err == nil {
		return accountID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("find wechat identity: %w", err)
	}

	accountID = uuid.NewString()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return "", fmt.Errorf("begin wechat registration: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_accounts (id, account_type, status)
VALUES (?, 'patient', 'active')`, accountID); err != nil {
		return "", fmt.Errorf("create patient account: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_external_identities
    (id, account_id, provider, provider_app_id, provider_subject)
VALUES (?, ?, 'wechat', ?, ?)`, uuid.NewString(), accountID, appID, openID); err != nil {
		if isDuplicateEntry(err) {
			_ = tx.Rollback()
			return s.findWeChatAccount(ctx, appID, openID)
		}
		return "", fmt.Errorf("bind wechat identity: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit wechat registration: %w", err)
	}
	return accountID, nil
}

// FindOrCreateVerifiedPhoneAccount makes the verified phone fingerprint the
// unique login identifier while keeping the immutable UUID as the account key.
func (s *MySQLStore) FindOrCreateVerifiedPhoneAccount(ctx context.Context, fingerprint []byte, masked string) (string, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return "", fmt.Errorf("begin phone registration: %w", err)
	}
	defer tx.Rollback()

	var accountID string
	err = tx.QueryRowContext(ctx, `
SELECT account_id
FROM identity_account_phones
WHERE phone_fingerprint = ?
FOR UPDATE`, fingerprint).Scan(&accountID)
	if err == nil {
		if _, err := tx.ExecContext(ctx, `
UPDATE identity_account_phones
SET phone_masked = ?, verification_status = 'verified', verification_source = 'sms',
    verified_at = CURRENT_TIMESTAMP(3), updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ?`, masked, accountID); err != nil {
			return "", fmt.Errorf("verify existing phone account: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return "", fmt.Errorf("commit existing phone login: %w", err)
		}
		return accountID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("find phone account: %w", err)
	}

	accountID = uuid.NewString()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_accounts (id, account_type, status)
VALUES (?, 'patient', 'active')`, accountID); err != nil {
		return "", fmt.Errorf("create phone patient account: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_account_phones
    (account_id, phone_fingerprint, phone_masked, verification_status, verification_source, verified_at)
VALUES (?, ?, ?, 'verified', 'sms', CURRENT_TIMESTAMP(3))`, accountID, fingerprint, masked); err != nil {
		if isDuplicateEntry(err) {
			_ = tx.Rollback()
			return s.markExistingPhoneVerified(ctx, fingerprint, masked)
		}
		return "", fmt.Errorf("bind verified phone: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit phone registration: %w", err)
	}
	return accountID, nil
}

func (s *MySQLStore) markExistingPhoneVerified(ctx context.Context, fingerprint []byte, masked string) (string, error) {
	var accountID string
	err := s.db.QueryRowContext(ctx, `
SELECT account_id
FROM identity_account_phones
WHERE phone_fingerprint = ?`, fingerprint).Scan(&accountID)
	if err != nil {
		return "", fmt.Errorf("find concurrent phone registration: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
UPDATE identity_account_phones
SET phone_masked = ?, verification_status = 'verified', verification_source = 'sms',
    verified_at = CURRENT_TIMESTAMP(3), updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ?`, masked, accountID); err != nil {
		return "", fmt.Errorf("verify concurrent phone registration: %w", err)
	}
	return accountID, nil
}

func (s *MySQLStore) findWeChatAccount(ctx context.Context, appID, openID string) (string, error) {
	var accountID string
	err := s.db.QueryRowContext(ctx, `
SELECT account_id
FROM identity_external_identities
WHERE provider = 'wechat' AND provider_app_id = ? AND provider_subject = ?`, appID, openID).Scan(&accountID)
	if err != nil {
		return "", fmt.Errorf("find concurrent wechat registration: %w", err)
	}
	return accountID, nil
}

func (s *MySQLStore) SetSelfReportedPhone(ctx context.Context, accountID string, fingerprint []byte, masked string) (account.PhoneBinding, error) {
	var currentStatus string
	err := s.db.QueryRowContext(ctx, `
SELECT verification_status
FROM identity_account_phones
WHERE account_id = ?`, accountID).Scan(&currentStatus)
	if err == nil && currentStatus == account.PhoneStatusVerified {
		return account.PhoneBinding{}, account.ErrVerifiedPhoneChange
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return account.PhoneBinding{}, fmt.Errorf("read current phone binding: %w", err)
	}

	result, err := s.db.ExecContext(ctx, `
UPDATE identity_account_phones
SET phone_fingerprint = ?, phone_masked = ?, verification_status = 'self_reported',
    verification_source = 'self_reported', verified_at = NULL,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ?`, fingerprint, masked, accountID)
	if err != nil {
		if isDuplicateEntry(err) {
			return account.PhoneBinding{}, account.ErrPhoneInUse
		}
		return account.PhoneBinding{}, fmt.Errorf("save self-reported phone: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return account.PhoneBinding{}, fmt.Errorf("read phone update result: %w", err)
	}
	if affected == 0 {
		var exists bool
		if err := s.db.QueryRowContext(ctx, `
SELECT EXISTS(SELECT 1 FROM identity_account_phones WHERE account_id = ?)`, accountID).Scan(&exists); err != nil {
			return account.PhoneBinding{}, fmt.Errorf("check existing phone binding: %w", err)
		}
		if exists {
			return account.PhoneBinding{
				Masked: masked, VerificationStatus: account.PhoneStatusSelfReported,
				VerificationSource: account.PhoneSourceSelfReported,
			}, nil
		}
		_, err = s.db.ExecContext(ctx, `
INSERT INTO identity_account_phones
    (account_id, phone_fingerprint, phone_masked, verification_status, verification_source, verified_at)
VALUES (?, ?, ?, 'self_reported', 'self_reported', NULL)`, accountID, fingerprint, masked)
		if err != nil {
			if isDuplicateEntry(err) {
				return account.PhoneBinding{}, account.ErrPhoneInUse
			}
			return account.PhoneBinding{}, fmt.Errorf("insert self-reported phone: %w", err)
		}
	}
	return account.PhoneBinding{
		Masked: masked, VerificationStatus: account.PhoneStatusSelfReported,
		VerificationSource: account.PhoneSourceSelfReported,
	}, nil
}

func (s *MySQLStore) FindAccountByPhone(ctx context.Context, fingerprint []byte) (account.Lookup, error) {
	var accountID string
	var binding account.PhoneBinding
	err := s.db.QueryRowContext(ctx, `
SELECT account_id, phone_masked, verification_status, verification_source
FROM identity_account_phones
WHERE phone_fingerprint = ?`, fingerprint).Scan(
		&accountID, &binding.Masked, &binding.VerificationStatus, &binding.VerificationSource,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return account.Lookup{}, authorization.ErrNotFound
	}
	if err != nil {
		return account.Lookup{}, fmt.Errorf("find account by phone: %w", err)
	}
	principal, err := s.GetAuthorizationContext(ctx, accountID)
	if err != nil {
		return account.Lookup{}, err
	}
	return account.Lookup{Principal: principal, Phone: binding}, nil
}

func NewMySQLStore(dataSource string) (*MySQLStore, error) {
	if dataSource == "" {
		return nil, errors.New("identity mysql data source is required")
	}
	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		return nil, fmt.Errorf("open identity mysql: %w", err)
	}
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping identity mysql: %w", err)
	}
	return &MySQLStore{db: db}, nil
}

func (s *MySQLStore) Close() error {
	return s.db.Close()
}

func (s *MySQLStore) GetAuthorizationContext(ctx context.Context, accountID string) (authn.Principal, error) {
	return readAuthorizationContext(ctx, s.db, accountID, false)
}

func (s *MySQLStore) WithinTransaction(ctx context.Context, fn func(authorization.TxStore) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin identity transaction: %w", err)
	}
	store := &mysqlTxStore{tx: tx}
	if err := fn(store); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback identity transaction: %w", rollbackErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit identity transaction: %w", err)
	}
	return nil
}

type mysqlTxStore struct {
	tx *sql.Tx
}

func (s *mysqlTxStore) GetAuthorizationContext(ctx context.Context, accountID string) (authn.Principal, error) {
	return readAuthorizationContext(ctx, s.tx, accountID, true)
}

func (s *mysqlTxStore) FindOperation(ctx context.Context, operationID string) (authorization.Operation, bool, error) {
	var operation authorization.Operation
	err := s.tx.QueryRowContext(ctx, `
SELECT operator_account_id, target_account_id
FROM identity_authorization_audit
WHERE operation_id = ?`, operationID).Scan(&operation.OperatorAccountID, &operation.TargetAccountID)
	if errors.Is(err, sql.ErrNoRows) {
		return authorization.Operation{}, false, nil
	}
	if err != nil {
		return authorization.Operation{}, false, fmt.Errorf("find authorization operation: %w", err)
	}
	return operation, true, nil
}

func (s *mysqlTxStore) SetRole(ctx context.Context, accountID, roleCode string) error {
	result, err := s.tx.ExecContext(ctx, `
INSERT INTO identity_account_roles (account_id, role_id)
SELECT ?, id FROM identity_roles WHERE code = ? AND status = 'active'
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id), created_at = CURRENT_TIMESTAMP(3)`, accountID, roleCode)
	if err != nil {
		return fmt.Errorf("assign identity role: %w", err)
	}
	_ = result // assigning the current role is a valid audited no-op
	return s.bumpAuthorizationVersion(ctx, accountID)
}

func (s *mysqlTxStore) SetDepartment(ctx context.Context, accountID, departmentID string) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_staff_profiles
SET department_id = ?, updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ?`, departmentID, accountID)
	if err != nil {
		return fmt.Errorf("change identity staff department: %w", err)
	}
	if err := requireAffected(result, "staff account"); err != nil {
		return err
	}
	return s.bumpAuthorizationVersion(ctx, accountID)
}

func (s *mysqlTxStore) SetAccountStatus(ctx context.Context, accountID, status string) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_accounts
SET status = ?, authorization_version = authorization_version + 1, updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?`, status, accountID)
	if err != nil {
		return fmt.Errorf("change identity account status: %w", err)
	}
	return requireAffected(result, "account")
}

func (s *mysqlTxStore) PromoteToDepartmentDoctor(ctx context.Context, accountID, departmentID string, verifyPhone bool) error {
	var departmentExists bool
	if err := s.tx.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1 FROM identity_departments WHERE id = ? AND status = 'active'
)`, departmentID).Scan(&departmentExists); err != nil {
		return fmt.Errorf("check doctor department: %w", err)
	}
	if !departmentExists {
		return fmt.Errorf("%w: active department", authorization.ErrNotFound)
	}

	var phoneStatus string
	if err := s.tx.QueryRowContext(ctx, `
SELECT verification_status
FROM identity_account_phones
WHERE account_id = ?
FOR UPDATE`, accountID).Scan(&phoneStatus); errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: account phone", authorization.ErrNotFound)
	} else if err != nil {
		return fmt.Errorf("lock doctor phone: %w", err)
	}
	if phoneStatus != account.PhoneStatusVerified && !verifyPhone {
		return fmt.Errorf("%w: phone verification is required", authorization.ErrInvalid)
	}
	if verifyPhone {
		if _, err := s.tx.ExecContext(ctx, `
UPDATE identity_account_phones
SET verification_status = 'verified', verification_source = 'admin',
    verified_at = CURRENT_TIMESTAMP(3), updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ?`, accountID); err != nil {
			return fmt.Errorf("verify doctor phone: %w", err)
		}
	}

	if _, err := s.tx.ExecContext(ctx, `
UPDATE identity_accounts
SET account_type = 'staff', authorization_version = authorization_version + 1,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?`, accountID); err != nil {
		return fmt.Errorf("promote staff account: %w", err)
	}
	if _, err := s.tx.ExecContext(ctx, `
INSERT INTO identity_staff_profiles (account_id, department_id)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE department_id = VALUES(department_id), updated_at = CURRENT_TIMESTAMP(3)`, accountID, departmentID); err != nil {
		return fmt.Errorf("save doctor department: %w", err)
	}
	_, err := s.tx.ExecContext(ctx, `
INSERT INTO identity_account_roles (account_id, role_id)
SELECT ?, id FROM identity_roles WHERE code = 'department_doctor' AND status = 'active'
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id), created_at = CURRENT_TIMESTAMP(3)`, accountID)
	if err != nil {
		return fmt.Errorf("assign department doctor role: %w", err)
	}
	var roleCode string
	if err := s.tx.QueryRowContext(ctx, `
SELECT r.code
FROM identity_account_roles ar
JOIN identity_roles r ON r.id = ar.role_id
WHERE ar.account_id = ?`, accountID).Scan(&roleCode); err != nil {
		return fmt.Errorf("verify department doctor role: %w", err)
	}
	if roleCode != authn.RoleDepartmentDoctor {
		return fmt.Errorf("%w: active department doctor role", authorization.ErrNotFound)
	}
	return nil
}

func (s *mysqlTxStore) RecordChange(ctx context.Context, change authorization.Change) error {
	beforeData, err := json.Marshal(change.Before)
	if err != nil {
		return fmt.Errorf("marshal authorization before state: %w", err)
	}
	afterData, err := json.Marshal(change.After)
	if err != nil {
		return fmt.Errorf("marshal authorization after state: %w", err)
	}

	auditID := uuid.NewString()
	_, err = s.tx.ExecContext(ctx, `
INSERT INTO identity_authorization_audit
    (id, operation_id, operator_account_id, target_account_id, action, before_data, after_data, request_id)
VALUES (?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''))`,
		auditID, change.OperationID, change.OperatorAccountID, change.TargetAccountID,
		change.Action, beforeData, afterData, change.RequestID)
	if err != nil {
		return fmt.Errorf("insert identity authorization audit: %w", err)
	}

	payload, err := json.Marshal(map[string]any{
		"operation_id":          change.OperationID,
		"operator_account_id":   change.OperatorAccountID,
		"target_account_id":     change.TargetAccountID,
		"action":                change.Action,
		"authorization_version": change.After.AuthorizationVersion,
	})
	if err != nil {
		return fmt.Errorf("marshal identity authorization event: %w", err)
	}
	_, err = s.tx.ExecContext(ctx, `
INSERT INTO identity_outbox_events
    (event_id, aggregate_id, event_type, schema_version, payload, occurred_at)
VALUES (?, ?, 'identity.authorization.changed.v1', 1, ?, ?)`,
		uuid.NewString(), change.TargetAccountID, payload, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("insert identity authorization outbox: %w", err)
	}
	return nil
}

func (s *mysqlTxStore) bumpAuthorizationVersion(ctx context.Context, accountID string) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_accounts
SET authorization_version = authorization_version + 1, updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?`, accountID)
	if err != nil {
		return fmt.Errorf("bump identity authorization version: %w", err)
	}
	return requireAffected(result, "account")
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func readAuthorizationContext(ctx context.Context, db queryer, accountID string, forUpdate bool) (authn.Principal, error) {
	query := `
SELECT
    a.id,
    a.account_type,
    a.status,
    a.authorization_version,
    COALESCE(sp.department_id, ''),
    COALESCE(r.code, ''),
    COALESCE(p.code, '')
FROM identity_accounts a
LEFT JOIN identity_staff_profiles sp ON sp.account_id = a.id
LEFT JOIN identity_account_roles ar ON ar.account_id = a.id
LEFT JOIN identity_roles r ON r.id = ar.role_id AND r.status = 'active'
LEFT JOIN identity_role_permissions rp ON rp.role_id = r.id
LEFT JOIN identity_permissions p ON p.id = rp.permission_id AND p.status = 'active'
WHERE a.id = ?`
	if forUpdate {
		query += " FOR UPDATE"
	}
	rows, err := db.QueryContext(ctx, query, accountID)
	if err != nil {
		return authn.Principal{}, fmt.Errorf("query identity authorization context: %w", err)
	}
	defer rows.Close()

	var principal authn.Principal
	roles := make(map[string]struct{})
	permissions := make(map[string]struct{})
	found := false
	for rows.Next() {
		var role, permission string
		if err := rows.Scan(
			&principal.AccountID, &principal.AccountType, &principal.Status,
			&principal.AuthorizationVersion, &principal.DepartmentID, &role, &permission,
		); err != nil {
			return authn.Principal{}, fmt.Errorf("scan identity authorization context: %w", err)
		}
		found = true
		if role != "" {
			roles[role] = struct{}{}
		}
		if permission != "" {
			permissions[permission] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return authn.Principal{}, fmt.Errorf("iterate identity authorization context: %w", err)
	}
	if !found {
		return authn.Principal{}, authorization.ErrNotFound
	}
	principal.Roles = sortedKeys(roles)
	principal.Permissions = sortedKeys(permissions)
	return principal, nil
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func requireAffected(result sql.Result, resource string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("%w: %s", authorization.ErrNotFound, resource)
	}
	return nil
}

func isDuplicateEntry(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
