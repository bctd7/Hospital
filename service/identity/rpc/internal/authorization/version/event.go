// Package version 拥有授权版本变更事件及其 Kafka → Redis 消费流程。
// 它只同步用于旧 Token 失效的版本号，不维护完整角色或权限读模型。
package version

import (
	"encoding/json"
	"fmt"

	contractevents "hospital/contracts/events"
)

// Event 是账号写事务保存到 Outbox 的授权版本事件内容。
// MySQL 适配器负责补充事件 ID 和发生时间。
type Event struct {
	AggregateID   string
	EventType     string
	SchemaVersion int
	Payload       json.RawMessage
}

// BuildChangedEvent 仅在授权版本发生变化时构造事件；资料字段更新不会产生失效事件。
func BuildChangedEvent(accountID, operationID, action string, beforeVersion, afterVersion int64) (Event, bool, error) {
	if afterVersion == beforeVersion {
		return Event{}, false, nil
	}
	if afterVersion < beforeVersion {
		return Event{}, false, fmt.Errorf("authorization version decreased from %d to %d", beforeVersion, afterVersion)
	}
	payload, err := json.Marshal(contractevents.IdentityAuthorizationChangedV1Payload{
		OperationID: operationID, Action: action, AuthorizationVersion: afterVersion,
	})
	if err != nil {
		return Event{}, false, fmt.Errorf("marshal authorization changed payload: %w", err)
	}
	return Event{
		AggregateID: accountID, EventType: contractevents.EventTypeIdentityAuthorizationChangedV1,
		SchemaVersion: 1, Payload: payload,
	}, true, nil
}
