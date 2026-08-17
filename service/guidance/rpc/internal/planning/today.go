package planning

import (
	"context"
	"sort"
	"time"

	"hospital/common/authn"
	"hospital/service/guidance/rpc/internal/projectconfiguration"
)

var shanghai = func() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
}()

// Today 根据患者当天真实预约状态生成被动查看的分阶段顺序建议。
func (m *Manager) Today(ctx context.Context, patient authn.Principal) (TodayRecommendation, error) {
	if err := requirePatient(patient); err != nil {
		return TodayRecommendation{}, err
	}
	active, err := m.appointment.ListMyBookings(ctx, "active")
	if err != nil {
		return TodayRecommendation{}, err
	}
	completed, err := m.appointment.ListMyBookings(ctx, "completed")
	if err != nil {
		return TodayRecommendation{}, err
	}
	today := time.Now().In(shanghai).Format("2006-01-02")
	bookings := make([]Booking, 0)
	for _, value := range append(active, completed...) {
		if value.ServiceDate == today {
			bookings = append(bookings, value)
		}
	}
	if len(bookings) == 0 {
		return TodayRecommendation{ServiceDate: today, Stages: []Stage{}, UpdatedAt: time.Now().UTC()}, nil
	}
	itemIDs := make([]string, 0, len(bookings))
	configurations := make(map[string]projectconfiguration.Configuration)
	for _, booking := range bookings {
		itemIDs = append(itemIDs, booking.ItemID)
		if configuration, loadErr := m.store.GetConfiguration(ctx, booking.ItemID); loadErr == nil {
			configurations[booking.ItemID] = configuration
		}
	}
	rules, err := m.store.ListAll(ctx)
	if err != nil {
		return TodayRecommendation{}, err
	}
	order := orderItems(unique(itemIDs), rules, configurations)
	orderIndex := make(map[string]int, len(order))
	for index, itemID := range order {
		orderIndex[itemID] = index
	}
	sort.SliceStable(bookings, func(i, j int) bool {
		left, right := statusRank(bookings[i].Status), statusRank(bookings[j].Status)
		if left != right {
			return left < right
		}
		return orderIndex[bookings[i].ItemID] < orderIndex[bookings[j].ItemID]
	})
	groups := []struct {
		title, status, focus string
		statuses             map[string]bool
	}{
		{"正在检查", "in_progress", "请按现场工作人员指引完成当前检查。", map[string]bool{"in_progress": true}},
		{"正在叫号", "called", "请留意现场广播并尽快前往检查房间。", map[string]bool{"called": true}},
		{"候检与待到院", "waiting", "建议按以下顺序前往；同楼栋项目已尽量集中。", map[string]bool{"queued": true, "confirmed": true}},
		{"今日已完成", "completed", "以下项目已完成，不再参与后续顺序计算。", map[string]bool{"report_pending": true, "completed": true, "no_show": true}},
	}
	stages := make([]Stage, 0, len(groups))
	for _, group := range groups {
		items := make([]PlanItem, 0)
		for _, booking := range bookings {
			if group.statuses[booking.Status] {
				items = append(items, bookingPlanItem(booking))
			}
		}
		if len(items) > 0 {
			stages = append(stages, Stage{StageNo: int32(len(stages) + 1), Title: group.title, Status: group.status, Focus: group.focus, Items: items})
		}
	}
	return TodayRecommendation{ServiceDate: today, Stages: stages, UpdatedAt: time.Now().UTC()}, nil
}
