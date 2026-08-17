package planning

import "hospital/service/guidance/rpc/internal/rules/precedence"

type slotBounds struct {
	lower string
	upper string
}

func (b slotBounds) allows(slot string) bool {
	return (b.lower == "" || slot >= b.lower) && (b.upper == "" || slot <= b.upper)
}

// existingPrecedenceBounds 把候选日期内已有预约作为不可移动的顺序锚点。
// no_show 没有实际完成检查，不能满足前置关系，也不能约束新的前置项目。
func existingPrecedenceBounds(itemIDs []string, selected map[string]struct{}, rules []precedence.Rule, bookings []Booking, allowedDates map[string]struct{}) map[string]slotBounds {
	bookingsByItem := make(map[string][]Booking)
	for _, booking := range bookings {
		if booking.Status == "no_show" {
			continue
		}
		if _, allowed := allowedDates[booking.ServiceDate]; !allowed {
			continue
		}
		bookingsByItem[booking.ItemID] = append(bookingsByItem[booking.ItemID], booking)
	}

	result := make(map[string]slotBounds, len(itemIDs))
	for _, itemID := range itemIDs {
		bounds := slotBounds{}
		for _, rule := range rules {
			switch {
			case rule.SuccessorItemID == itemID:
				if _, counterpartSelected := selected[rule.PredecessorItemID]; counterpartSelected {
					continue
				}
				earliest := earliestBookingSlot(bookingsByItem[rule.PredecessorItemID])
				if earliest != "" && earliest > bounds.lower {
					bounds.lower = earliest
				}
			case rule.PredecessorItemID == itemID:
				if _, counterpartSelected := selected[rule.SuccessorItemID]; counterpartSelected {
					continue
				}
				latest := latestBookingSlot(bookingsByItem[rule.SuccessorItemID])
				if latest != "" && (bounds.upper == "" || latest < bounds.upper) {
					bounds.upper = latest
				}
			}
		}
		result[itemID] = bounds
	}
	return result
}

func earliestBookingSlot(bookings []Booking) string {
	result := ""
	for _, booking := range bookings {
		slot := bookingSlot(booking)
		if result == "" || slot < result {
			result = slot
		}
	}
	return result
}

func latestBookingSlot(bookings []Booking) string {
	result := ""
	for _, booking := range bookings {
		slot := bookingSlot(booking)
		if slot > result {
			result = slot
		}
	}
	return result
}

func bookingSlot(booking Booking) string {
	if booking.Session == "morning" {
		return booking.ServiceDate + ":0"
	}
	return booking.ServiceDate + ":1"
}
