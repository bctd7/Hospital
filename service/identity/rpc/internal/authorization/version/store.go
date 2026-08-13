package version

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Store 是 Consumer 写入当前授权版本所需的最小端口，生产实现使用 Redis。
type Store interface {
	AdvanceAuthorizationVersion(ctx context.Context, accountID string, version int64) (bool, error)
}

// MessageReader 是 Consumer 使用的 Kafka 读取端口；只有 Redis 更新成功后才允许提交 Offset。
type MessageReader interface {
	FetchMessage(ctx context.Context) (*kgo.Record, error)
	CommitMessage(ctx context.Context, record *kgo.Record) error
}
