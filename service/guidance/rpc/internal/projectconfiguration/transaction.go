package projectconfiguration

import (
	"context"
	"time"

	"hospital/common/authn"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

// finishConfirmation 完成 Appointment 的最终确认；确认结果不确定时先尝试取消，
// 只有 Appointment 明确接受取消后才补偿 Guidance 本地数据。
func (m *Manager) finishConfirmation(ctx context.Context, operator authn.Principal, transaction Transaction) (Result, error) {
	project, err := m.appointment.Confirm(ctx, transaction.TransactionID)
	if err == nil {
		if stateErr := m.setState(ctx, transaction.TransactionID, StateConfirmed, "", transaction.RetryCount); stateErr != nil {
			return Result{}, stateErr
		}
		configuration, loadErr := m.store.GetConfiguration(ctx, transaction.ItemID)
		if loadErr != nil {
			return Result{}, loadErr
		}
		rules, loadErr := m.store.ListRulesByOwner(ctx, transaction.ItemID)
		return Result{Project: project, Configuration: configuration, Rules: rules}, loadErr
	}

	if cancelErr := m.appointment.Cancel(ctx, transaction.TransactionID); cancelErr != nil {
		m.recordFailure(ctx, transaction, err)
		return Result{}, err
	}
	if compensationErr := m.compensate(ctx, transaction); compensationErr != nil {
		return Result{}, compensationErr
	}
	_ = m.setState(ctx, transaction.TransactionID, StateCancelled, err.Error(), transaction.RetryCount+1)
	return Result{}, err
}

// applyGuidanceConfiguration 在 Guidance 单库事务中保存说明规则和直接先后关系。
// 全局图锁保证并发修改时循环校验看到的是一致的规则图。
func (m *Manager) applyGuidanceConfiguration(ctx context.Context, operator authn.Principal, transaction *Transaction) error {
	command := transaction.Payload
	return m.store.WithinTransaction(ctx, func(tx TxStore) error {
		if err := tx.LockGraph(ctx); err != nil {
			return err
		}
		currentConfiguration, found, err := tx.GetConfigurationForUpdate(ctx, command.ItemID)
		if err != nil {
			return err
		}
		if command.Action == ActionCreate {
			if found || command.ExpectedConfigurationVersion != 0 {
				return ErrVersionConflict
			}
		} else if !found || currentConfiguration.Version != command.ExpectedConfigurationVersion {
			return ErrVersionConflict
		}
		beforeRules, err := tx.ListRulesByOwnerForUpdate(ctx, command.ItemID)
		if err != nil {
			return err
		}
		var before *Configuration
		if found {
			copy := currentConfiguration
			before = &copy
		}
		if err := tx.SetTransactionBefore(ctx, transaction.TransactionID, before, beforeRules, time.Now().UTC()); err != nil {
			return err
		}
		transaction.BeforeConfiguration = before
		transaction.BeforePrecedenceRules = beforeRules

		allRules, err := tx.ListAllRules(ctx)
		if err != nil {
			return err
		}
		remaining := make([]precedence.Rule, 0, len(allRules)+len(command.PrecedenceRules))
		for _, rule := range allRules {
			if rule.OwnerItemID != command.ItemID {
				remaining = append(remaining, rule)
			}
		}
		remaining = append(remaining, command.PrecedenceRules...)
		if hasCycle(remaining) {
			return ErrCycle
		}
		if err := tx.DeleteRulesByOwner(ctx, command.ItemID); err != nil {
			return err
		}
		for _, rule := range command.PrecedenceRules {
			if err := tx.InsertRule(ctx, rule); err != nil {
				return err
			}
		}
		now := time.Now().UTC()
		version := int64(1)
		createdAt := now
		if found {
			version = currentConfiguration.Version + 1
			createdAt = currentConfiguration.CreatedAt
		}
		configuration := Configuration{
			ItemID: command.ItemID, Description: command.Description,
			PreparationRules: command.PreparationRules, Reminders: command.Reminders,
			Version: version, UpdatedBy: operator.AccountID, CreatedAt: createdAt, UpdatedAt: now,
		}
		if err := tx.UpsertConfiguration(ctx, configuration); err != nil {
			return err
		}
		return tx.UpdateTransaction(ctx, transaction.TransactionID, StateConfirming, "", transaction.RetryCount, now)
	})
}

// compensate 恢复本次配置前的 Guidance 快照，可安全重试。
func (m *Manager) compensate(ctx context.Context, transaction Transaction) error {
	return m.store.WithinTransaction(ctx, func(tx TxStore) error {
		current, found, err := tx.GetTransactionForUpdate(ctx, transaction.TransactionID)
		if err != nil || !found {
			return err
		}
		if current.State == StateCancelled {
			return nil
		}
		if err := tx.LockGraph(ctx); err != nil {
			return err
		}
		if err := tx.DeleteRulesByOwner(ctx, transaction.ItemID); err != nil {
			return err
		}
		for _, rule := range current.BeforePrecedenceRules {
			if err := tx.InsertRule(ctx, rule); err != nil {
				return err
			}
		}
		if current.BeforeConfiguration == nil {
			if err := tx.DeleteConfiguration(ctx, transaction.ItemID); err != nil {
				return err
			}
		} else if err := tx.UpsertConfiguration(ctx, *current.BeforeConfiguration); err != nil {
			return err
		}
		return tx.UpdateTransaction(ctx, transaction.TransactionID, StateCancelled, current.LastError, current.RetryCount+1, time.Now().UTC())
	})
}

func (m *Manager) setState(ctx context.Context, transactionID, state, lastError string, retryCount int) error {
	return m.store.WithinTransaction(ctx, func(tx TxStore) error {
		return tx.UpdateTransaction(ctx, transactionID, state, lastError, retryCount, time.Now().UTC())
	})
}

func (m *Manager) recordFailure(ctx context.Context, transaction Transaction, cause error) {
	_ = m.setState(ctx, transaction.TransactionID, transaction.State, cause.Error(), transaction.RetryCount+1)
}
