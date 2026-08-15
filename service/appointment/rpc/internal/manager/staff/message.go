package staff

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	"hospital/service/appointment/rpc/internal/manager/common"
	staffsupport "hospital/service/appointment/rpc/internal/manager/staff/support"
)

func (m *Manager) ListMessages(ctx context.Context, operator authn.Principal, requestedDepartmentID string, page, pageSize int64) (common.MessagePage, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return common.MessagePage{}, err
	}
	departmentID := ""
	var err error
	if strings.TrimSpace(requestedDepartmentID) != "" || !operator.HasRole(authn.RoleSuperAdmin) {
		departmentID, err = staffsupport.ScopedDepartment(operator, requestedDepartmentID)
		if err != nil {
			return common.MessagePage{}, err
		}
	}
	page, pageSize, _, err = staffsupport.NormalizePage(page, pageSize)
	if err != nil {
		return common.MessagePage{}, err
	}
	selectedBookings := []common.Booking{}
	if departmentID != "" {
		selectedBookings, err = m.messages.ListMessageBookings(ctx, "", departmentID)
		if err != nil {
			return common.MessagePage{}, err
		}
	}
	values := common.DepartmentMessages(selectedBookings, time.Now())
	reads, err := m.messages.ListMessageReads(ctx, operator.AccountID, staffMessageKeys(values))
	if err != nil {
		return common.MessagePage{}, err
	}
	unread := common.ApplyMessageReads(values, reads)

	allBookings := selectedBookings
	if operator.HasRole(authn.RoleSuperAdmin) {
		allBookings, err = m.messages.ListMessageBookings(ctx, "", "")
		if err != nil {
			return common.MessagePage{}, err
		}
	}
	allMessages := common.DepartmentMessages(allBookings, time.Now())
	allReads, err := m.messages.ListMessageReads(ctx, operator.AccountID, staffMessageKeys(allMessages))
	if err != nil {
		return common.MessagePage{}, err
	}
	common.ApplyMessageReads(allMessages, allReads)
	counts := make(map[string]int64)
	for _, message := range allMessages {
		if message.ReadAt == nil {
			counts[message.Booking.DepartmentID]++
		}
	}
	departmentCounts := make([]common.DepartmentUnreadCount, 0, len(counts))
	for id, count := range counts {
		departmentCounts = append(departmentCounts, common.DepartmentUnreadCount{DepartmentID: id, UnreadCount: count})
	}
	if operator.HasRole(authn.RoleSuperAdmin) {
		unread = 0
		for _, count := range counts {
			unread += count
		}
	}
	sort.Slice(departmentCounts, func(i, j int) bool { return departmentCounts[i].DepartmentID < departmentCounts[j].DepartmentID })
	return common.MessagePage{Items: common.PaginateMessages(values, page, pageSize), Page: page, PageSize: pageSize, Total: int64(len(values)), UnreadCount: unread, DepartmentUnreadCounts: departmentCounts}, nil
}

func (m *Manager) MarkMessageRead(ctx context.Context, operator authn.Principal, requestedDepartmentID, messageKey string) (common.Message, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return common.Message{}, err
	}
	departmentID, err := staffsupport.ScopedDepartment(operator, requestedDepartmentID)
	if err != nil {
		return common.Message{}, err
	}
	messageKey = strings.TrimSpace(messageKey)
	if messageKey == "" || len(messageKey) > 160 {
		return common.Message{}, fmt.Errorf("%w: invalid message_key", ErrInvalid)
	}
	bookings, err := m.messages.ListMessageBookings(ctx, "", departmentID)
	if err != nil {
		return common.Message{}, err
	}
	for _, message := range common.DepartmentMessages(bookings, time.Now()) {
		if message.MessageKey != messageKey {
			continue
		}
		now := time.Now().UTC()
		if err := m.messages.MarkMessageRead(ctx, operator.AccountID, messageKey, now); err != nil {
			return common.Message{}, err
		}
		message.ReadAt = &now
		return message, nil
	}
	return common.Message{}, ErrNotFound
}

func staffMessageKeys(values []common.Message) []string {
	keys := make([]string, 0, len(values))
	for _, value := range values {
		keys = append(keys, value.MessageKey)
	}
	return keys
}
