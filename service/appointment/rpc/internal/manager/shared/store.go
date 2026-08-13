package shared

import (
	"context"

	"hospital/service/appointment/rpc/internal/manager/common"
)

// Store 是双方共用业务当前所需的持久化端口。
// 当前只有读取能力；以后只有确实被患者端和工作人员端共同使用的能力才能加入。
type Store interface {
	GetItem(ctx context.Context, itemID string) (common.ExaminationItem, error)
	ListItems(ctx context.Context, filter common.ProjectListFilter) ([]common.ExaminationItem, int64, error)
	GetItemSummary(ctx context.Context, itemID string) (common.ItemSummary, error)
	ListItemRooms(ctx context.Context, itemID string, activeOnly bool) ([]common.RoomItem, error)
	ListItemWindows(ctx context.Context, itemID string, activeOnly bool) ([]common.ItemWeeklyWindow, error)
}
