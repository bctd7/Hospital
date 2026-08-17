package planning

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMorningStartMinutes    = 9 * 60
	defaultMorningEndMinutes      = 12 * 60
	defaultAfternoonStartMinutes  = 13 * 60
	defaultAfternoonEndMinutes    = 18 * 60
	defaultDurationMinutes        = 5
	sameBuildingTravelMinutes     = 5
	fallbackBuildingTravelMinutes = 15
	successfulTravelCacheTTL      = 24 * time.Hour
	fallbackTravelCacheTTL        = 5 * time.Minute
)

type travelEstimate struct {
	minutes   int32
	estimated bool
}

func effectiveOptionWindow(option Option) (int32, int32, int32, bool) {
	defaultStart, defaultEnd := defaultSessionWindow(option.Session)
	start, end := int32(0), int32(24*60)
	hasStart, hasEnd := false, false
	for _, value := range []string{option.RoomOpenTime, option.ItemStartTime} {
		if parsed, ok := parseClockMinutes(value); ok {
			if !hasStart || parsed > start {
				start = parsed
			}
			hasStart = true
		}
	}
	for _, value := range []string{option.RoomCloseTime, option.ItemEndTime} {
		if parsed, ok := parseClockMinutes(value); ok {
			if !hasEnd || parsed < end {
				end = parsed
			}
			hasEnd = true
		}
	}
	if !hasStart {
		start = defaultStart
	}
	if !hasEnd {
		end = defaultEnd
	}
	duration := option.EstimatedDurationMinutes
	if duration <= 0 {
		duration = defaultDurationMinutes
	}
	return start, end, duration, start < end && start+duration <= end
}

func defaultSessionWindow(session string) (int32, int32) {
	if session == "afternoon" {
		return defaultAfternoonStartMinutes, defaultAfternoonEndMinutes
	}
	return defaultMorningStartMinutes, defaultMorningEndMinutes
}

func parseClockMinutes(value string) (int32, bool) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) < 2 {
		return 0, false
	}
	hour, hourErr := strconv.Atoi(parts[0])
	minute, minuteErr := strconv.Atoi(parts[1])
	if hourErr != nil || minuteErr != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, false
	}
	return int32(hour*60 + minute), true
}

func formatClockMinutes(value int32) string {
	if value < 0 {
		value = 0
	}
	return strconv.Itoa(int(value/60)/10) + strconv.Itoa(int(value/60)%10) + ":" +
		strconv.Itoa(int(value%60)/10) + strconv.Itoa(int(value%60)%10)
}

func travelPairKey(originBuilding, destinationBuilding string) string {
	if originBuilding > destinationBuilding {
		originBuilding, destinationBuilding = destinationBuilding, originBuilding
	}
	return originBuilding + "\x00" + destinationBuilding
}

func hospitalBuildingKeyword(building string) string {
	building = strings.TrimSpace(building)
	if strings.Contains(building, "上海市第二人民医院") {
		return building
	}
	return "上海市第二人民医院" + building
}

// estimateBuildingTravel 在进入搜索前一次性取得楼栋间耗时，避免 DFS 分支调用外部地图。
// 地图不可用时保留可执行方案，但按 15 分钟保守估算并在结果中标记。
func estimateBuildingTravel(ctx context.Context, estimator TravelEstimator, options map[string][]Option) map[string]travelEstimate {
	buildings := uniqueOptionBuildings(options)
	result := make(map[string]travelEstimate, len(buildings)*len(buildings)/2)
	for origin := 0; origin < len(buildings); origin++ {
		for destination := origin + 1; destination < len(buildings); destination++ {
			result[travelPairKey(buildings[origin], buildings[destination])] = estimateBuildingPair(ctx, estimator, buildings[origin], buildings[destination])
		}
	}
	return result
}

func uniqueOptionBuildings(options map[string][]Option) []string {
	buildingsSet := map[string]struct{}{}
	for _, values := range options {
		for _, option := range values {
			if value := strings.TrimSpace(option.Building); value != "" {
				buildingsSet[value] = struct{}{}
			}
		}
	}
	buildings := make([]string, 0, len(buildingsSet))
	for building := range buildingsSet {
		buildings = append(buildings, building)
	}
	sort.Strings(buildings)
	return buildings
}

func estimateBuildingPair(ctx context.Context, estimator TravelEstimator, origin, destination string) travelEstimate {
	estimate := travelEstimate{minutes: fallbackBuildingTravelMinutes, estimated: true}
	if estimator == nil {
		return estimate
	}
	minutes, err := estimator.EstimateWalkingMinutes(ctx, "上海市", hospitalBuildingKeyword(origin), hospitalBuildingKeyword(destination))
	if err == nil && minutes > 0 {
		estimate.minutes = roundUpFiveMinutes(minutes)
		estimate.estimated = false
	}
	return estimate
}

// estimateBuildingTravel 对高频楼栋对做进程内缓存。真实路线缓存一天，
// 外部地图失败的 15 分钟兜底只缓存五分钟，避免短暂故障长期污染方案。
func (m *Manager) estimateBuildingTravel(ctx context.Context, options map[string][]Option) map[string]travelEstimate {
	buildings := uniqueOptionBuildings(options)
	result := make(map[string]travelEstimate, len(buildings)*len(buildings)/2)
	now := time.Now()
	for origin := 0; origin < len(buildings); origin++ {
		for destination := origin + 1; destination < len(buildings); destination++ {
			key := travelPairKey(buildings[origin], buildings[destination])
			m.travelMu.RLock()
			cached, found := m.travelCache[key]
			m.travelMu.RUnlock()
			if found && now.Before(cached.expiresAt) {
				result[key] = cached.value
				continue
			}
			value := estimateBuildingPair(ctx, m.travel, buildings[origin], buildings[destination])
			ttl := successfulTravelCacheTTL
			if value.estimated {
				ttl = fallbackTravelCacheTTL
			}
			m.travelMu.Lock()
			m.travelCache[key] = cachedTravelEstimate{value: value, expiresAt: now.Add(ttl)}
			m.travelMu.Unlock()
			result[key] = value
		}
	}
	return result
}

func roundUpFiveMinutes(value int32) int32 {
	if value <= 0 {
		return 5
	}
	return (value + 4) / 5 * 5
}

func travelBetween(previous searchChoice, next Option, estimates map[string]travelEstimate) travelEstimate {
	if previous.option.RoomID == "" || previous.option.RoomID == next.RoomID {
		return travelEstimate{}
	}
	if strings.TrimSpace(previous.option.Building) == strings.TrimSpace(next.Building) {
		return travelEstimate{minutes: sameBuildingTravelMinutes, estimated: true}
	}
	if estimate, ok := estimates[travelPairKey(previous.option.Building, next.Building)]; ok {
		return estimate
	}
	return travelEstimate{minutes: fallbackBuildingTravelMinutes, estimated: true}
}
