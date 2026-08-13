package resource

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

func normalizeUUID(value, field string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("%w: %s must be a UUID", ErrInvalid, field)
	}
	return parsed.String(), nil
}

func normalizeLocationText(value, field string, max int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: %s is required", ErrInvalid, field)
	}
	if utf8.RuneCountInString(value) > max {
		return "", fmt.Errorf("%w: %s exceeds %d characters", ErrInvalid, field, max)
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("%w: %s contains control characters", ErrInvalid, field)
	}
	return value, nil
}

func normalizeRoomLocation(campusID, building string, floorNumber int32, roomNumber string) (string, string, int32, string, string, error) {
	campusID, err := normalizeUUID(campusID, "campus_id")
	if err != nil {
		return "", "", 0, "", "", err
	}
	building, err = normalizeLocationText(building, "building", 64)
	if err != nil {
		return "", "", 0, "", "", err
	}
	if floorNumber == 0 || floorNumber < -9 || floorNumber > 99 {
		return "", "", 0, "", "", fmt.Errorf("%w: floor_number must be -9..-1 or 1..99", ErrInvalid)
	}
	roomNumber, err = normalizeLocationText(roomNumber, "room_number", 32)
	if err != nil {
		return "", "", 0, "", "", err
	}
	for _, r := range roomNumber {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return "", "", 0, "", "", fmt.Errorf("%w: room_number contains unsupported characters", ErrInvalid)
		}
	}
	return campusID, building, floorNumber, roomNumber, FormatRoomDisplayName(building, floorNumber, roomNumber), nil
}

func FormatRoomDisplayName(building string, floorNumber int32, roomNumber string) string {
	floor := fmt.Sprintf("%d层", floorNumber)
	if floorNumber < 0 {
		floor = fmt.Sprintf("B%d层", -floorNumber)
	}
	return fmt.Sprintf("%s · %s · %s室", building, floor, roomNumber)
}

func normalizeOperation(meta OperationMeta) (OperationMeta, error) {
	var err error
	meta.OperationID, err = normalizeUUID(meta.OperationID, "operation_id")
	if err != nil {
		return OperationMeta{}, err
	}
	meta.RequestID = strings.TrimSpace(meta.RequestID)
	if len(meta.RequestID) > 64 || strings.IndexFunc(meta.RequestID, unicode.IsControl) >= 0 {
		return OperationMeta{}, fmt.Errorf("%w: invalid request_id", ErrInvalid)
	}
	return meta, nil
}

func normalizePage(page, pageSize int64) (int64, int64, int64, error) {
	if page == 0 {
		page = defaultPage
	}
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if page < 1 || pageSize < 1 || pageSize > maxPageSize {
		return 0, 0, 0, fmt.Errorf("%w: invalid page or page_size", ErrInvalid)
	}
	if page-1 > (1<<63-1)/pageSize {
		return 0, 0, 0, fmt.Errorf("%w: page offset exceeds supported range", ErrInvalid)
	}
	return page, pageSize, (page - 1) * pageSize, nil
}

func normalizeSlot(weekday int32, session Session) (int32, Session, error) {
	if weekday < 1 || weekday > 7 {
		return 0, "", fmt.Errorf("%w: weekday must be between 1 and 7", ErrInvalid)
	}
	session = Session(strings.TrimSpace(string(session)))
	if !session.Valid() {
		return 0, "", fmt.Errorf("%w: invalid session", ErrInvalid)
	}
	return weekday, session, nil
}

func normalizeClock(value, field string) (string, int, error) {
	value = strings.TrimSpace(value)
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return "", 0, fmt.Errorf("%w: %s must use HH:MM", ErrInvalid, field)
	}
	minutes := parsed.Hour()*60 + parsed.Minute()
	return parsed.Format("15:04"), minutes, nil
}

func containsWindow(room RoomWeeklyWindow, item ItemWeeklyWindow) bool {
	if room.Status != StatusActive || item.Status != StatusActive || room.Weekday != item.Weekday || room.Session != item.Session {
		return false
	}
	_, roomStart, _ := normalizeClock(room.OpenTime, "open_time")
	_, roomEnd, _ := normalizeClock(room.CloseTime, "close_time")
	_, itemStart, _ := normalizeClock(item.StartTime, "start_time")
	_, itemEnd, _ := normalizeClock(item.EndTime, "end_time")
	return roomStart <= itemStart && itemEnd <= roomEnd
}
