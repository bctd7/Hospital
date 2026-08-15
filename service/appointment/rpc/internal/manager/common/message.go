package common

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type MessageType string

const (
	MessageTypeBookingCreated  MessageType = "booking_created"
	MessageTypeArrival60       MessageType = "arrival_60m"
	MessageTypeArrival30       MessageType = "arrival_30m"
	MessageTypeBookingCanceled MessageType = "booking_canceled"
	MessageTypeBookingNoShow   MessageType = "booking_no_show"
	MessageTypeReportDue       MessageType = "report_due"
	MessageTypeReportOverdue   MessageType = "report_overdue"
	MessageTypeReportPublished MessageType = "report_published"
	MessageTypeReportCorrected MessageType = "report_corrected"
)

// MessageReportFact 是生成患者报告消息所需的最小事实，不包含报告正文。
type MessageReportFact struct {
	ReportID    string
	VersionID   string
	VersionNo   int64
	VersionKind ReportVersionKind
	Booking     Booking
	PublishedAt time.Time
}

// Message 是由预约和报告事实动态形成的只读消息。
type Message struct {
	MessageKey      string      `json:"message_key"`
	MessageType     MessageType `json:"message_type"`
	OccurredAt      time.Time   `json:"occurred_at"`
	ReadAt          *time.Time  `json:"read_at,omitempty"`
	Booking         Booking     `json:"booking"`
	ReportID        string      `json:"report_id,omitempty"`
	ReportVersionID string      `json:"report_version_id,omitempty"`
	ReportVersionNo int64       `json:"report_version_no,omitempty"`
}

type DepartmentUnreadCount struct {
	DepartmentID string `json:"department_id"`
	UnreadCount  int64  `json:"unread_count"`
}

type MessagePage struct {
	Items                  []Message               `json:"items"`
	Page                   int64                   `json:"page"`
	PageSize               int64                   `json:"page_size"`
	Total                  int64                   `json:"total"`
	UnreadCount            int64                   `json:"unread_count"`
	DepartmentUnreadCounts []DepartmentUnreadCount `json:"department_unread_counts,omitempty"`
}

// MessageStore 只保存已读游标；消息正文始终从预约、报告及报告版本动态计算。
type MessageStore interface {
	ListMessageBookings(ctx context.Context, patientAccountID, departmentID string) ([]Booking, error)
	ListPatientMessageReportFacts(ctx context.Context, patientAccountID string) ([]MessageReportFact, error)
	ListMessageReads(ctx context.Context, accountID string, messageKeys []string) (map[string]time.Time, error)
	MarkMessageRead(ctx context.Context, accountID, messageKey string, readAt time.Time) error
}

func PatientMessages(bookings []Booking, reports []MessageReportFact, now time.Time) []Message {
	values := make([]Message, 0, len(bookings)*3+len(reports))
	for _, booking := range bookings {
		values = append(values, Message{MessageKey: "booking:" + booking.BookingID + ":patient:created", MessageType: MessageTypeBookingCreated, OccurredAt: booking.CreatedAt, Booking: booking})
		cutoff, err := messageDateTime(booking.ServiceDate, booking.BookingCutoffTime)
		if err != nil {
			continue
		}
		for _, reminder := range []struct {
			delta  time.Duration
			suffix string
			type_  MessageType
		}{{time.Hour, "arrival-60m", MessageTypeArrival60}, {30 * time.Minute, "arrival-30m", MessageTypeArrival30}} {
			at := cutoff.Add(-reminder.delta)
			if now.Before(at) || bookingStartedBefore(booking, at) || bookingCanceledBefore(booking, at) {
				continue
			}
			values = append(values, Message{MessageKey: "booking:" + booking.BookingID + ":patient:" + reminder.suffix, MessageType: reminder.type_, OccurredAt: at, Booking: booking})
		}
	}
	for _, fact := range reports {
		messageType := MessageTypeReportPublished
		messageKey := "report:" + fact.ReportID + ":patient:published"
		if fact.VersionKind == ReportVersionKindCorrection {
			messageType = MessageTypeReportCorrected
			messageKey = "report-version:" + fact.VersionID + ":patient:corrected"
		}
		values = append(values, Message{
			MessageKey:  messageKey,
			MessageType: messageType, OccurredAt: fact.PublishedAt, Booking: fact.Booking,
			ReportID: fact.ReportID, ReportVersionID: fact.VersionID, ReportVersionNo: fact.VersionNo,
		})
	}
	sortMessages(values)
	return values
}

func DepartmentMessages(bookings []Booking, now time.Time) []Message {
	values := make([]Message, 0, len(bookings)*3)
	for _, booking := range bookings {
		values = append(values, Message{MessageKey: "booking:" + booking.BookingID + ":department:created", MessageType: MessageTypeBookingCreated, OccurredAt: booking.CreatedAt, Booking: booking})
		if booking.Status == BookingStatusCanceled {
			values = append(values, Message{MessageKey: "booking:" + booking.BookingID + ":department:canceled", MessageType: MessageTypeBookingCanceled, OccurredAt: booking.UpdatedAt, Booking: booking})
		}
		cutoff, cutoffErr := messageDateTime(booking.ServiceDate, booking.BookingCutoffTime)
		if booking.Status == BookingStatusNoShow && cutoffErr == nil {
			values = append(values, Message{MessageKey: "booking:" + booking.BookingID + ":department:no-show", MessageType: MessageTypeBookingNoShow, OccurredAt: cutoff, Booking: booking})
		}
		end, err := messageDateTime(booking.ServiceDate, booking.ItemEndTime)
		if err != nil || booking.StartedAt == nil {
			continue
		}
		for _, reminder := range []struct {
			at     time.Time
			suffix string
			type_  MessageType
		}{{end, "report-due", MessageTypeReportDue}, {end.Add(time.Hour), "report-overdue", MessageTypeReportOverdue}} {
			if now.Before(reminder.at) || booking.StartedAt.After(reminder.at) || (booking.CompletedAt != nil && !booking.CompletedAt.After(reminder.at)) {
				continue
			}
			values = append(values, Message{MessageKey: "booking:" + booking.BookingID + ":department:" + reminder.suffix, MessageType: reminder.type_, OccurredAt: reminder.at, Booking: booking})
		}
	}
	sortMessages(values)
	return values
}

func ApplyMessageReads(values []Message, reads map[string]time.Time) int64 {
	var unread int64
	for index := range values {
		if readAt, ok := reads[values[index].MessageKey]; ok {
			copy := readAt
			values[index].ReadAt = &copy
		} else {
			unread++
		}
	}
	return unread
}

func PaginateMessages(values []Message, page, pageSize int64) []Message {
	start := (page - 1) * pageSize
	if start >= int64(len(values)) {
		return []Message{}
	}
	end := start + pageSize
	if end > int64(len(values)) {
		end = int64(len(values))
	}
	return values[start:end]
}

func sortMessages(values []Message) {
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].OccurredAt.Equal(values[j].OccurredAt) {
			return values[i].MessageKey > values[j].MessageKey
		}
		return values[i].OccurredAt.After(values[j].OccurredAt)
	})
}

func messageDateTime(date time.Time, clock string) (time.Time, error) {
	parsed, err := time.Parse("15:04:05", strings.TrimSpace(clock))
	if err != nil {
		return time.Time{}, fmt.Errorf("parse message clock: %w", err)
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	date = date.In(location)
	return time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, location), nil
}

func bookingStartedBefore(booking Booking, at time.Time) bool {
	return booking.StartedAt != nil && !booking.StartedAt.After(at)
}

func bookingCanceledBefore(booking Booking, at time.Time) bool {
	return booking.Status == BookingStatusCanceled && !booking.UpdatedAt.After(at)
}
