package support

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"hospital/service/appointment/rpc/internal/manager/common"
)

const (
	maxProjectNameRunes        = 128
	maxProjectDescriptionRunes = 8192
	maxRequestIDBytes          = 64
	defaultPage                = 1
	defaultPageSize            = 20
	maxPageSize                = 100
)

// NormalizeUUID 去除首尾空白并返回标准 UUID 字符串。
func NormalizeUUID(value, field string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("%w: %s must be a UUID", common.ErrInvalid, field)
	}
	return parsed.String(), nil
}

// NormalizeRequestID 规范化可选的链路请求编号。
func NormalizeRequestID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) > maxRequestIDBytes {
		return "", fmt.Errorf("%w: request_id exceeds %d bytes", common.ErrInvalid, maxRequestIDBytes)
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("%w: request_id contains control characters", common.ErrInvalid)
	}
	return value, nil
}

// NormalizeOperation 规范化幂等操作编号和链路请求编号。
func NormalizeOperation(operationID, requestID string) (string, string, error) {
	operationID, err := NormalizeUUID(operationID, "operation_id")
	if err != nil {
		return "", "", err
	}
	requestID, err = NormalizeRequestID(requestID)
	if err != nil {
		return "", "", err
	}
	return operationID, requestID, nil
}

// NormalizePage 应用分页默认值并计算安全的偏移量。
func NormalizePage(page, pageSize int64) (int64, int64, int64, error) {
	if page == 0 {
		page = defaultPage
	}
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if page < 1 || pageSize < 1 || pageSize > maxPageSize {
		return 0, 0, 0, fmt.Errorf("%w: invalid page or page_size", common.ErrInvalid)
	}
	if page-1 > (1<<63-1)/pageSize {
		return 0, 0, 0, fmt.Errorf("%w: page offset exceeds supported range", common.ErrInvalid)
	}
	return page, pageSize, (page - 1) * pageSize, nil
}

// NormalizeRoomLocation 校验并规范化房间的严格地址字段。
func NormalizeRoomLocation(campusID, building string, floorNumber int32, roomNumber string) (string, string, int32, string, string, error) {
	campusID, err := NormalizeUUID(campusID, "campus_id")
	if err != nil {
		return "", "", 0, "", "", err
	}
	building, err = normalizeRequiredText(building, "building", 64)
	if err != nil {
		return "", "", 0, "", "", err
	}
	if floorNumber == 0 || floorNumber < -9 || floorNumber > 99 {
		return "", "", 0, "", "", fmt.Errorf("%w: floor_number must be -9..-1 or 1..99", common.ErrInvalid)
	}
	roomNumber, err = normalizeRequiredText(roomNumber, "room_number", 32)
	if err != nil {
		return "", "", 0, "", "", err
	}
	for _, r := range roomNumber {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return "", "", 0, "", "", fmt.Errorf("%w: room_number contains unsupported characters", common.ErrInvalid)
		}
	}
	return campusID, building, floorNumber, roomNumber, common.FormatRoomDisplayName(building, floorNumber, roomNumber), nil
}

// NormalizeSlot 校验星期和上午、下午时段。
func NormalizeSlot(weekday int32, session common.Session) (int32, common.Session, error) {
	if weekday < 1 || weekday > 7 {
		return 0, "", fmt.Errorf("%w: weekday must be between 1 and 7", common.ErrInvalid)
	}
	session = common.Session(strings.TrimSpace(string(session)))
	if !session.Valid() {
		return 0, "", fmt.Errorf("%w: invalid session", common.ErrInvalid)
	}
	return weekday, session, nil
}

// NormalizeClock 只接受 HH:MM，并同时返回从零点开始的分钟数。
func NormalizeClock(value, field string) (string, int, error) {
	value = strings.TrimSpace(value)
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return "", 0, fmt.Errorf("%w: %s must use HH:MM", common.ErrInvalid, field)
	}
	return parsed.Format("15:04"), parsed.Hour()*60 + parsed.Minute(), nil
}

// FitsSessionBoundary 判断时间是否完整位于上午或下午大窗口内。
func FitsSessionBoundary(session common.Session, startMinutes, endMinutes int) bool {
	const noon = 12 * 60
	if session == common.SessionMorning {
		return startMinutes < endMinutes && endMinutes <= noon
	}
	if session == common.SessionAfternoon {
		return noon <= startMinutes && startMinutes < endMinutes
	}
	return false
}

// NormalizeProjectName 校验项目名称。
func NormalizeProjectName(value string) (string, error) {
	return normalizeSingleLineText(value, "name", maxProjectNameRunes)
}

// NormalizeProjectDescription 保留说明中的换行和制表符，拒绝其他控制字符。
func NormalizeProjectDescription(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: description is required", common.ErrInvalid)
	}
	if utf8.RuneCountInString(value) > maxProjectDescriptionRunes {
		return "", fmt.Errorf("%w: description exceeds %d characters", common.ErrInvalid, maxProjectDescriptionRunes)
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t'
	}) >= 0 {
		return "", fmt.Errorf("%w: description contains unsupported control characters", common.ErrInvalid)
	}
	return value, nil
}

func normalizeRequiredText(value, field string, maxRunes int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: %s is required", common.ErrInvalid, field)
	}
	if utf8.RuneCountInString(value) > maxRunes {
		return "", fmt.Errorf("%w: %s exceeds %d characters", common.ErrInvalid, field, maxRunes)
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("%w: %s contains control characters", common.ErrInvalid, field)
	}
	return value, nil
}

func normalizeSingleLineText(value, field string, maxRunes int) (string, error) {
	return normalizeRequiredText(value, field, maxRunes)
}
