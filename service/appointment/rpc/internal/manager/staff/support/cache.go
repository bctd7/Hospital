package support

import (
	"context"
	"errors"
	"time"

	"hospital/service/appointment/rpc/internal/manager/common"
)

type (
	// Cache 是 support 操作热点读取副本所需的最小缓存契约。
	Cache = common.Cache
	// FlightGroup 合并同一进程内针对同一缓存键的并发回源。
	FlightGroup = common.FlightGroup
)

// HotReadCacheTTL 是工作人员资源热点读取的基础缓存时间。
const HotReadCacheTTL = common.HotReadCacheTTL

// LoadCached 统一处理缓存读取、并发回源合并和缓存写入。
func LoadCached[T any](ctx context.Context, cache Cache, flights *FlightGroup, key string, ttl time.Duration, loader func() (T, bool, error)) (T, bool, error) {
	return common.LoadCached(ctx, cache, flights, key, ttl, loader)
}

// JitteredTTL 为基础缓存时间添加小幅随机抖动，避免热点键同时失效。
func JitteredTTL(base time.Duration) time.Duration { return common.JitteredTTL(base) }

// ItemSummaryReader 是缓存辅助函数读取项目摘要所需的最小接口。
type ItemSummaryReader interface {
	GetItemSummary(ctx context.Context, itemID string) (common.ItemSummary, error)
}

// Generation 返回科室缓存版本；缓存不可用时退化为固定版本，不阻断主业务。
func Generation(ctx context.Context, cache common.Cache, departmentID string) string {
	if cache == nil {
		return "0"
	}
	value, err := cache.DepartmentGeneration(ctx, departmentID)
	if err != nil {
		return "0"
	}
	return value
}

// GetItemSummary 读取带热点缓存的项目摘要。
func GetItemSummary(ctx context.Context, reader ItemSummaryReader, cache common.Cache, flights *common.FlightGroup, itemID string) (common.ItemSummary, error) {
	value, found, err := common.LoadCached(ctx, cache, flights, "item-summary:"+itemID, common.HotReadCacheTTL, func() (common.ItemSummary, bool, error) {
		item, loadErr := reader.GetItemSummary(ctx, itemID)
		if errors.Is(loadErr, common.ErrNotFound) {
			return common.ItemSummary{}, false, nil
		}
		return item, loadErr == nil, loadErr
	})
	if err != nil {
		return common.ItemSummary{}, err
	}
	if !found {
		return common.ItemSummary{}, common.ErrNotFound
	}
	return value, nil
}

// Invalidate 让科室范围缓存切换版本，并删除需要立即失效的精确键。
func Invalidate(ctx context.Context, cache common.Cache, departmentID string, keys ...string) {
	if cache == nil {
		return
	}
	if departmentID != "" {
		_ = cache.BumpDepartment(ctx, departmentID)
	}
	_ = cache.Delete(ctx, keys...)
}
