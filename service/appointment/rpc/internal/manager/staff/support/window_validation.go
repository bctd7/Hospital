package support

import (
	"context"
	"fmt"

	"hospital/service/appointment/rpc/internal/manager/common"
)

// ValidatePair 校验项目预约时间是否完整落在对应房间的开放窗口内。
func ValidatePair(ctx context.Context, tx common.ConfigurationTxStore, roomID, itemID string, overrideRoom *common.RoomWeeklyWindow, overrideItem *common.ItemWeeklyWindow) error {
	rooms, err := tx.ListRoomWindowsForUpdate(ctx, roomID)
	if err != nil {
		return err
	}
	items, err := tx.ListItemWindowsForUpdate(ctx, itemID)
	if err != nil {
		return err
	}
	if overrideRoom != nil {
		rooms = replaceRoomWindow(rooms, *overrideRoom)
	}
	if overrideItem != nil {
		items = replaceItemWindow(items, *overrideItem)
	}
	for _, item := range items {
		if item.Status != common.StatusActive {
			continue
		}
		matched := false
		for _, room := range rooms {
			if ContainsWindow(room, item) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%w: room=%s item=%s weekday=%d session=%s", common.ErrWindowConflict, roomID, itemID, item.Weekday, item.Session)
		}
	}
	return nil
}

// ValidateRoomWindowChange 校验房间窗口修改后，所有已关联项目仍满足包含关系。
func ValidateRoomWindowChange(ctx context.Context, tx common.ConfigurationTxStore, window common.RoomWeeklyWindow) error {
	relations, err := tx.ListRoomRelationsForUpdate(ctx, window.RoomID)
	if err != nil {
		return err
	}
	for _, relation := range relations {
		if relation.Status == common.StatusActive {
			if err := ValidatePair(ctx, tx, relation.RoomID, relation.ItemID, &window, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateItemWindowChange 校验项目窗口修改后，所有已关联房间仍满足包含关系。
func ValidateItemWindowChange(ctx context.Context, tx common.ConfigurationTxStore, window common.ItemWeeklyWindow) error {
	relations, err := tx.ListItemRelationsForUpdate(ctx, window.ItemID)
	if err != nil {
		return err
	}
	for _, relation := range relations {
		if relation.Status == common.StatusActive {
			if err := ValidatePair(ctx, tx, relation.RoomID, relation.ItemID, nil, &window); err != nil {
				return err
			}
		}
	}
	return nil
}

func replaceRoomWindow(values []common.RoomWeeklyWindow, value common.RoomWeeklyWindow) []common.RoomWeeklyWindow {
	for i := range values {
		if values[i].WindowID == value.WindowID || (values[i].Weekday == value.Weekday && values[i].Session == value.Session) {
			values[i] = value
			return values
		}
	}
	return append(values, value)
}

func replaceItemWindow(values []common.ItemWeeklyWindow, value common.ItemWeeklyWindow) []common.ItemWeeklyWindow {
	for i := range values {
		if values[i].WindowID == value.WindowID || (values[i].Weekday == value.Weekday && values[i].Session == value.Session) {
			values[i] = value
			return values
		}
	}
	return append(values, value)
}

// ContainsWindow 判断项目窗口是否与房间属于同一天、同一时段且完整包含在房间窗口内。
func ContainsWindow(room common.RoomWeeklyWindow, item common.ItemWeeklyWindow) bool {
	if room.Status != common.StatusActive || item.Status != common.StatusActive || room.Weekday != item.Weekday || room.Session != item.Session {
		return false
	}
	return room.OpenTime <= item.StartTime && item.EndTime <= room.CloseTime
}
