package projectconfiguration

import (
	"context"
	"time"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"

	"github.com/google/uuid"
)

// Configure 执行一次完整的项目配置。
//
// 顺序固定为：规范化输入 → 建立幂等事务 → Appointment Try → 写入 Guidance
// → Appointment Confirm。Appointment 最后确认，患者不会看到只有项目事实、
// 尚无规则的半成品。
func (m *Manager) Configure(ctx context.Context, operator authn.Principal, command Command) (Result, error) {
	if err := requirePermission(operator, contractauthz.PermissionRuleEdit); err != nil {
		return Result{}, err
	}
	command, err := m.normalizeAndResolve(ctx, operator, command)
	if err != nil {
		return Result{}, err
	}
	fingerprint, err := commandFingerprint(command)
	if err != nil {
		return Result{}, err
	}
	now := time.Now().UTC()
	transaction := Transaction{
		TransactionID: uuid.NewString(), OperationID: command.OperationID, Action: command.Action,
		ItemID: command.ItemID, OperatorAccountID: operator.AccountID, State: StateTrying,
		RequestFingerprint: fingerprint, Payload: command, CreatedAt: now, UpdatedAt: now,
	}

	err = m.store.WithinTransaction(ctx, func(tx TxStore) error {
		existing, found, findErr := tx.FindTransactionByOperation(ctx, command.OperationID)
		if findErr != nil {
			return findErr
		}
		if found {
			if existing.OperatorAccountID != operator.AccountID || existing.RequestFingerprint != fingerprint {
				return ErrConflict
			}
			transaction = existing
			return nil
		}
		return tx.CreateTransaction(ctx, transaction)
	})
	if err != nil {
		return Result{}, err
	}
	command = transaction.Payload

	switch transaction.State {
	case StateConfirmed:
		return m.Get(ctx, operator, command.ItemID)
	case StateCancelled, StateCancelling:
		return Result{}, ErrConflict
	case StateConfirming:
		return m.finishConfirmation(ctx, operator, transaction)
	}

	if err := m.appointment.Prepare(ctx, transaction.TransactionID, command); err != nil {
		m.recordFailure(ctx, transaction, err)
		return Result{}, err
	}
	if err := m.setState(ctx, transaction.TransactionID, StatePrepared, "", transaction.RetryCount); err != nil {
		_ = m.appointment.Cancel(ctx, transaction.TransactionID)
		return Result{}, err
	}
	transaction.State = StatePrepared

	if err := m.applyGuidanceConfiguration(ctx, operator, &transaction); err != nil {
		_ = m.appointment.Cancel(ctx, transaction.TransactionID)
		_ = m.setState(ctx, transaction.TransactionID, StateCancelled, err.Error(), transaction.RetryCount+1)
		return Result{}, err
	}
	transaction.State = StateConfirming
	return m.finishConfirmation(ctx, operator, transaction)
}
