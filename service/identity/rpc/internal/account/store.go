package account

import "context"

type Store interface {
	GetDisplayProfile(ctx context.Context, accountID string) (DisplayProfile, error)
	UpdateDisplayProfile(ctx context.Context, accountID, nickname string) (DisplayProfile, error)
	ListDoctors(ctx context.Context, departmentID string, page, pageSize int64) (DoctorPage, error)
	ListAccounts(ctx context.Context, filter AccountFilter) (AccountPage, error)
	GetAccount(ctx context.Context, accountID string) (Account, error)
	FindAccountIDByPhone(ctx context.Context, fingerprint []byte) (string, error)
	WithinAccountTransaction(ctx context.Context, fn func(TxStore) error) error
}

type TxStore interface {
	FindAccountOperation(ctx context.Context, operationID string) (Operation, bool, error)
	GetAccountForUpdate(ctx context.Context, accountID string) (Account, error)
	PromoteDoctor(ctx context.Context, accountID, departmentID string, profile DoctorProfileInput, expectedVersion int64, verifyPhone bool) error
	UpdateDoctor(ctx context.Context, accountID string, profile OptionalDoctorProfileInput, expectedVersion int64) error
	ChangeDoctorDepartment(ctx context.Context, accountID, departmentID string, expectedVersion int64) error
	RevokeDoctor(ctx context.Context, accountID string, expectedVersion int64) error
	SetManagedAccountStatus(ctx context.Context, accountID, status string, expectedVersion int64) error
	RecordAccountChange(ctx context.Context, change Change) error
}
