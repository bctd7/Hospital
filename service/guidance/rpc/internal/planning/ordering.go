package planning

import (
	"sort"
	"strconv"
	"strings"

	"hospital/service/guidance/rpc/internal/projectconfiguration"
	descriptionrules "hospital/service/guidance/rpc/internal/rules/description"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

// orderItems 先保证直接先后关系，再用说明解析出的准备状态为同层项目排序。
func orderItems(itemIDs []string, rules []precedence.Rule, configurations map[string]projectconfiguration.Configuration) []string {
	selected := make(map[string]bool, len(itemIDs))
	for _, id := range itemIDs {
		selected[id] = true
	}
	graph, indegree := map[string][]string{}, map[string]int{}
	for _, id := range itemIDs {
		indegree[id] = 0
	}
	for _, rule := range rules {
		if selected[rule.PredecessorItemID] && selected[rule.SuccessorItemID] {
			graph[rule.PredecessorItemID] = append(graph[rule.PredecessorItemID], rule.SuccessorItemID)
			indegree[rule.SuccessorItemID]++
		}
	}
	result := make([]string, 0, len(itemIDs))
	for len(result) < len(itemIDs) {
		ready := make([]string, 0)
		for _, id := range itemIDs {
			if indegree[id] == 0 && !contains(result, id) {
				ready = append(ready, id)
			}
		}
		if len(ready) == 0 {
			return append([]string(nil), itemIDs...)
		}
		sort.SliceStable(ready, func(i, j int) bool {
			left, right := descriptionRuleRank(configurations[ready[i]]), descriptionRuleRank(configurations[ready[j]])
			return left < right || (left == right && ready[i] < ready[j])
		})
		id := ready[0]
		result = append(result, id)
		for _, next := range graph[id] {
			indegree[next]--
		}
	}
	return result
}

// descriptionRuleRank 表达当前已确认的宏观偏好：禁食/禁水靠前，无要求居中，喝水靠后。
func descriptionRuleRank(configuration projectconfiguration.Configuration) int {
	hasRestricted, hasDrink := false, false
	for _, rule := range configuration.PreparationRules {
		if rule.RuleType == descriptionrules.RuleTypeFasting || rule.RuleType == descriptionrules.RuleTypeNoWater {
			hasRestricted = true
		}
		if rule.RuleType == descriptionrules.RuleTypeDrinkWater {
			hasDrink = true
		}
	}
	if hasRestricted {
		return 0
	}
	if hasDrink {
		return 2
	}
	return 1
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func optionSlot(value Option) string {
	return value.ServiceDate + ":" + strconv.Itoa(sessionRank(value.Session))
}

func optionCapacityKey(value Option) string {
	return value.RoomID + ":" + value.ServiceDate + ":" + value.Session
}

func sessionRank(value string) int {
	if value == "morning" {
		return 0
	}
	return 1
}

func planTitle(variant int) string {
	return []string{"优先推荐", "备选方案一", "备选方案二"}[variant]
}

func planSummary(items []PlanItem) string {
	if len(items) == 0 {
		return ""
	}
	buildings := unique(func() []string {
		values := make([]string, 0, len(items))
		for _, item := range items {
			values = append(values, item.Building)
		}
		return values
	}())
	return "共安排" + strconv.Itoa(len(items)) + "项检查，涉及" + strings.Join(buildings, "、")
}

func planReason(itemID string, configuration projectconfiguration.Configuration, rules []precedence.Rule, previous []PlanItem) string {
	if descriptionRuleRank(configuration) == 0 {
		return "优先完成禁食或禁水项目"
	}
	if descriptionRuleRank(configuration) == 2 {
		return "安排在饮水准备状态满足后"
	}
	for _, rule := range rules {
		if rule.SuccessorItemID == itemID {
			if message := strings.TrimSpace(rule.PatientMessage); message != "" {
				return message
			}
			return "按项目先后关系安排"
		}
	}
	if len(previous) > 0 && previous[len(previous)-1].Building != "" {
		return "同楼栋项目尽量集中"
	}
	return "按可预约窗口安排"
}

func statusRank(status string) int {
	switch status {
	case "in_progress":
		return 0
	case "called":
		return 1
	case "queued":
		return 2
	case "confirmed":
		return 3
	default:
		return 4
	}
}

func bookingPlanItem(value Booking) PlanItem {
	return PlanItem{ItemID: value.ItemID, ItemName: value.ItemName, RoomID: value.RoomID, RoomDisplayName: value.RoomDisplayName, CampusID: value.CampusID, Building: value.Building, FloorNumber: value.FloorNumber, RoomNumber: value.RoomNumber, ServiceDate: value.ServiceDate, Session: value.Session, EstimatedDurationMinutes: value.EstimatedDurationMinutes, Reason: value.Status}
}
