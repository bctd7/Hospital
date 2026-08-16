package support

import (
	"context"
	"encoding/json"
	"fmt"

	"hospital/common/authn"
	"hospital/service/appointment/rpc/internal/manager/common"
)

// OperationStore 是幂等写操作模板需要的最小事务入口。
type OperationStore interface {
	WithinConfigurationTransaction(ctx context.Context, fn func(common.ConfigurationTxStore) error) error
}

// OperationMeta 只携带幂等编号和链路请求编号。
type OperationMeta struct {
	OperationID string
	RequestID   string
}

// Mutate 统一处理房间和开放时间配置写操作的幂等校验、事务执行与审计记录。
// 具体业务规则仍由调用方传入的 apply 执行，本函数不决定业务状态。
func Mutate(ctx context.Context, store OperationStore, operator authn.Principal, meta OperationMeta, resourceType, resourceID, action string, payload any, departmentID func() string, apply func(common.ConfigurationTxStore) (any, string, error), replay func([]byte) error) error {
	fingerprint := common.RequestFingerprint(payload)
	return store.WithinConfigurationTransaction(ctx, func(tx common.ConfigurationTxStore) error {
		operation, exists, err := tx.FindOperation(ctx, meta.OperationID)
		if err != nil {
			return err
		}
		if exists {
			if operation.OperatorAccountID != operator.AccountID || operation.ResourceType != resourceType || operation.Action != action || operation.RequestFingerprint != fingerprint || (resourceID != "" && operation.ResourceID != resourceID) {
				return fmt.Errorf("%w: operation_id was already used", common.ErrConflict)
			}
			return replay(operation.ResultData)
		}
		result, actualID, err := apply(tx)
		if err != nil {
			return err
		}
		before, after := splitResult(result)
		return tx.RecordChange(ctx, common.ConfigurationChange{OperationID: meta.OperationID, OperatorAccountID: operator.AccountID, DepartmentID: departmentID(), ResourceType: resourceType, ResourceID: actualID, Action: action, RequestFingerprint: fingerprint, Before: before, After: after, RequestID: meta.RequestID})
	})
}

// DecodeAfter 从审计结果中恢复写操作完成后的业务数据，用于幂等重放。
func DecodeAfter(data []byte, target any) error {
	var envelope struct{ After json.RawMessage }
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	if len(envelope.After) == 0 {
		return json.Unmarshal(data, target)
	}
	return json.Unmarshal(envelope.After, target)
}

func splitResult(value any) (any, any) {
	data, _ := json.Marshal(value)
	var envelope map[string]json.RawMessage
	if json.Unmarshal(data, &envelope) == nil {
		if after, ok := envelope["After"]; ok {
			var a any
			_ = json.Unmarshal(after, &a)
			var b any
			if before, exists := envelope["Before"]; exists && string(before) != "null" {
				_ = json.Unmarshal(before, &b)
			}
			return b, a
		}
	}
	return nil, value
}
