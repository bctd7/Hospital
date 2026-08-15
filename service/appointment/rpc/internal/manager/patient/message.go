package patient

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hospital/common/authn"
	"hospital/service/appointment/rpc/internal/manager/common"
)

func (m *Manager) ListMyMessages(ctx context.Context, patient authn.Principal, page, pageSize int64) (common.MessagePage, error) {
	if err := requirePatient(patient); err != nil {
		return common.MessagePage{}, err
	}
	page, pageSize, _, err := normalizeBookingPage(page, pageSize)
	if err != nil {
		return common.MessagePage{}, err
	}
	bookings, err := m.messages.ListMessageBookings(ctx, patient.AccountID, "")
	if err != nil {
		return common.MessagePage{}, err
	}
	reports, err := m.messages.ListPatientMessageReportFacts(ctx, patient.AccountID)
	if err != nil {
		return common.MessagePage{}, err
	}
	values := common.PatientMessages(bookings, reports, time.Now())
	reads, err := m.messages.ListMessageReads(ctx, patient.AccountID, messageKeys(values))
	if err != nil {
		return common.MessagePage{}, err
	}
	unread := common.ApplyMessageReads(values, reads)
	return common.MessagePage{Items: common.PaginateMessages(values, page, pageSize), Page: page, PageSize: pageSize, Total: int64(len(values)), UnreadCount: unread}, nil
}

func (m *Manager) MarkMyMessageRead(ctx context.Context, patient authn.Principal, messageKey string) (common.Message, error) {
	if err := requirePatient(patient); err != nil {
		return common.Message{}, err
	}
	messageKey = strings.TrimSpace(messageKey)
	if messageKey == "" || len(messageKey) > 160 {
		return common.Message{}, fmt.Errorf("%w: invalid message_key", ErrInvalid)
	}
	bookings, err := m.messages.ListMessageBookings(ctx, patient.AccountID, "")
	if err != nil {
		return common.Message{}, err
	}
	reports, err := m.messages.ListPatientMessageReportFacts(ctx, patient.AccountID)
	if err != nil {
		return common.Message{}, err
	}
	for _, message := range common.PatientMessages(bookings, reports, time.Now()) {
		if message.MessageKey != messageKey {
			continue
		}
		now := time.Now().UTC()
		if err := m.messages.MarkMessageRead(ctx, patient.AccountID, messageKey, now); err != nil {
			return common.Message{}, err
		}
		message.ReadAt = &now
		return message, nil
	}
	return common.Message{}, ErrNotFound
}

func messageKeys(values []common.Message) []string {
	keys := make([]string, 0, len(values))
	for _, value := range values {
		keys = append(keys, value.MessageKey)
	}
	return keys
}
