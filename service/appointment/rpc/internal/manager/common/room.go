package common

import (
	"fmt"
	"time"
)

// Room 保存严格的院区、楼栋、楼层和房间号信息。
type Room struct {
	RoomID       string     `json:"room_id"`
	DepartmentID string     `json:"department_id"`
	CampusID     string     `json:"campus_id"`
	Building     string     `json:"building"`
	FloorNumber  int32      `json:"floor_number"`
	RoomNumber   string     `json:"room_number"`
	DisplayName  string     `json:"display_name"`
	RetiredAt    *time.Time `json:"retired_at,omitempty"`
	Version      int64      `json:"version"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// RoomItem 表示房间能够执行某个检查项目的关系。
type RoomItem struct {
	RelationID      string    `json:"relation_id"`
	RoomID          string    `json:"room_id"`
	ItemID          string    `json:"item_id"`
	RoomDisplayName string    `json:"room_display_name,omitempty"`
	CampusID        string    `json:"campus_id,omitempty"`
	Building        string    `json:"building,omitempty"`
	FloorNumber     int32     `json:"floor_number,omitempty"`
	RoomNumber      string    `json:"room_number,omitempty"`
	ItemName        string    `json:"item_name,omitempty"`
	Status          Status    `json:"status"`
	Version         int64     `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// RoomWeeklyWindow 表示房间自身的每周开放时间和容量。
type RoomWeeklyWindow struct {
	WindowID       string    `json:"window_id"`
	RoomID         string    `json:"room_id"`
	Weekday        int32     `json:"weekday"`
	Session        Session   `json:"session"`
	OpenTime       string    `json:"open_time"`
	CloseTime      string    `json:"close_time"`
	ActiveCapacity int64     `json:"active_capacity"`
	Status         Status    `json:"status"`
	Version        int64     `json:"version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// FormatRoomDisplayName 根据严格地址字段生成稳定的房间展示名。
func FormatRoomDisplayName(building string, floorNumber int32, roomNumber string) string {
	floor := fmt.Sprintf("%d层", floorNumber)
	if floorNumber < 0 {
		floor = fmt.Sprintf("B%d层", -floorNumber)
	}
	return fmt.Sprintf("%s · %s · %s室", building, floor, roomNumber)
}
