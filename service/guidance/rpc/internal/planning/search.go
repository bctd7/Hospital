package planning

import (
	"sort"
	"strings"

	"hospital/service/guidance/rpc/internal/projectconfiguration"
	descriptionrules "hospital/service/guidance/rpc/internal/rules/description"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

const maximumGeneratedPlans = 3

type searchChoice struct {
	itemID              string
	assignmentKey       string
	option              Option
	plannedStartMinutes int32
	plannedEndMinutes   int32
	travelMinutes       int32
	travelEstimated     bool
}

type searchOptionCandidate struct {
	option        Option
	assignmentKey string
	windowStart   int32
	windowEnd     int32
	duration      int32
	capacityIndex int
}

type searchScore struct {
	preparationInversions      int
	preparationTransitions     int
	preparationMaturityPenalty int
	serviceDays                int
	campusChanges              int
	buildingChanges            int
	travelMinutes              int
	completionDate             string
	completionMinutes          int32
}

type searchCandidate struct {
	choices []searchChoice
	score   searchScore
	key     string
}

// searchState 只保留会影响后续可行性或评分的事实。具体路径与容量占用由搜索器原地回溯。
type searchState struct {
	selectedMask          uint16
	lastDate              string
	cursorMinutes         int32
	lastRoomID            string
	lastCampus            string
	lastBuilding          string
	lastNoWaterEndMinutes int32
	lastRank              int8
	dayRankCounts         [3]uint8
	drinkToday            bool
	score                 searchScore
}

type searchMemoKey struct {
	selectedMask          uint16
	lastDate              string
	cursorMinutes         int32
	lastRoomID            string
	lastCampus            string
	lastBuilding          string
	lastNoWaterEndMinutes int32
	lastRank              int8
	dayRankCounts         [3]uint8
	drinkToday            bool
	capacityUsage         string
}

type searchMemoScores struct {
	values [maximumGeneratedPlans]searchScore
	count  uint8
}

type planSearcher struct {
	itemIDs        []string
	itemIndex      map[string]int
	options        [][]searchOptionCandidate
	configurations map[string]projectconfiguration.Configuration
	prerequisites  []uint16
	travel         map[string]travelEstimate

	capacityIndexes map[string]int
	capacityLimits  []uint8
	capacityUsage   []uint8

	path    []searchChoice
	memo    map[searchMemoKey]searchMemoScores
	results []searchCandidate
}

// searchBestPlans 供不关心地图的包内测试和兼容调用使用；跨楼栋按 15 分钟保守估算。
func searchBestPlans(itemIDs []string, options map[string][]Option, rules []precedence.Rule, configurations map[string]projectconfiguration.Configuration) []searchCandidate {
	return searchBestPlansWithTravel(itemIDs, options, rules, configurations, estimateBuildingTravel(nil, nil, options))
}

// searchBestPlansWithTravel 使用完整 DFS 枚举项目顺序和预约选项，并通过记忆化合并等价后缀。
// 每个分支按“能开始的最早时刻”排程，预计时长和楼栋移动均计入窗口可行性。
func searchBestPlansWithTravel(itemIDs []string, options map[string][]Option, rules []precedence.Rule, configurations map[string]projectconfiguration.Configuration, travel map[string]travelEstimate) []searchCandidate {
	searcher := &planSearcher{
		itemIDs: append([]string(nil), itemIDs...), itemIndex: make(map[string]int, len(itemIDs)),
		options: make([][]searchOptionCandidate, len(itemIDs)), configurations: configurations,
		prerequisites: make([]uint16, len(itemIDs)), travel: travel,
		capacityIndexes: map[string]int{}, path: make([]searchChoice, 0, len(itemIDs)), memo: map[searchMemoKey]searchMemoScores{},
	}
	rawCapacityLimits := map[string]int64{}
	capacityUsers := map[string]map[string]struct{}{}
	normalizedByItem := make([][]Option, len(itemIDs))
	for index, itemID := range searcher.itemIDs {
		searcher.itemIndex[itemID] = index
		normalizedByItem[index] = normalizedSearchOptions(options[itemID])
		for _, option := range normalizedByItem[index] {
			key := optionCapacityKey(option)
			if current, ok := rawCapacityLimits[key]; !ok || option.RemainingCapacity < current {
				rawCapacityLimits[key] = option.RemainingCapacity
			}
			if capacityUsers[key] == nil {
				capacityUsers[key] = map[string]struct{}{}
			}
			capacityUsers[key][itemID] = struct{}{}
		}
	}
	constrainedKeys := make([]string, 0)
	for key, limit := range rawCapacityLimits {
		if int64(len(capacityUsers[key])) > limit {
			constrainedKeys = append(constrainedKeys, key)
		}
	}
	sort.Strings(constrainedKeys)
	searcher.capacityLimits = make([]uint8, len(constrainedKeys))
	searcher.capacityUsage = make([]uint8, len(constrainedKeys))
	for index, key := range constrainedKeys {
		searcher.capacityIndexes[key] = index
		limit := rawCapacityLimits[key]
		if limit > int64(len(itemIDs)) {
			limit = int64(len(itemIDs))
		}
		if limit > 0 {
			searcher.capacityLimits[index] = uint8(limit)
		}
	}
	for itemIndex := range normalizedByItem {
		for _, option := range normalizedByItem[itemIndex] {
			windowStart, windowEnd, duration, valid := effectiveOptionWindow(option)
			if !valid {
				continue
			}
			capacityIndex := -1
			if index, ok := searcher.capacityIndexes[optionCapacityKey(option)]; ok {
				capacityIndex = index
			}
			searcher.options[itemIndex] = append(searcher.options[itemIndex], searchOptionCandidate{
				option: option, assignmentKey: itemIDs[itemIndex] + "@" + optionCapacityKey(option), windowStart: windowStart, windowEnd: windowEnd, duration: duration, capacityIndex: capacityIndex,
			})
		}
	}
	for _, rule := range rules {
		predecessor, predecessorSelected := searcher.itemIndex[rule.PredecessorItemID]
		successor, successorSelected := searcher.itemIndex[rule.SuccessorItemID]
		if predecessorSelected && successorSelected {
			searcher.prerequisites[successor] |= 1 << predecessor
		}
	}
	searcher.visit(searchState{lastRank: -1})
	return searcher.results
}

func (s *planSearcher) visit(state searchState) {
	if !s.canStillReachTopPlans(state) {
		return
	}
	if !s.acceptState(state) {
		return
	}
	if len(s.path) == len(s.itemIDs) {
		s.addResult(state)
		return
	}

	for itemIndex, itemID := range s.itemIDs {
		bit := uint16(1 << itemIndex)
		if state.selectedMask&bit != 0 || state.selectedMask&s.prerequisites[itemIndex] != s.prerequisites[itemIndex] {
			continue
		}
		configuration := s.configurations[itemID]
		rank := int8(descriptionRuleRank(configuration))
		for _, candidate := range s.options[itemIndex] {
			option := candidate.option
			if state.lastDate != "" && option.ServiceDate < state.lastDate {
				continue
			}
			capacityIndex := candidate.capacityIndex
			if capacityIndex >= 0 && s.capacityUsage[capacityIndex] >= s.capacityLimits[capacityIndex] {
				continue
			}

			sameDate := state.lastDate == option.ServiceDate
			// 当前产品把“空腹”解释为禁食禁水；饮水准备已经开始后，
			// 同一天不再安排任何受限准备项目。
			if sameDate && state.drinkToday && rank == 0 {
				continue
			}
			travel := travelEstimate{}
			earliestStart := candidate.windowStart
			lastNoWaterEnd := int32(0)
			if sameDate {
				lastNoWaterEnd = state.lastNoWaterEndMinutes
				if len(s.path) > 0 {
					travel = travelBetween(s.path[len(s.path)-1], option, s.travel)
				}
				if cursor := state.cursorMinutes + travel.minutes; cursor > earliestStart {
					earliestStart = cursor
				}
			}
			maturityPenalty := 0
			if drinkRule, ok := preparationRule(configuration, descriptionrules.RuleTypeDrinkWater); ok && lastNoWaterEnd > 0 {
				readyAt := lastNoWaterEnd + drinkRule.MinAdvanceMinutes
				if readyAt > earliestStart {
					earliestStart = readyAt
				}
				elapsed := earliestStart - lastNoWaterEnd
				maturityPenalty = preparationMaturityPenalty(drinkRule, elapsed)
			}
			end := earliestStart + candidate.duration
			if end > candidate.windowEnd {
				continue
			}

			next := state
			next.selectedMask |= bit
			next.lastDate = option.ServiceDate
			next.cursorMinutes = end
			next.lastRoomID = option.RoomID
			next.lastCampus = option.CampusID
			next.lastBuilding = option.Building
			next.lastRank = rank
			if sameDate {
				for previousRank := int(rank) + 1; previousRank < len(state.dayRankCounts); previousRank++ {
					next.score.preparationInversions += int(state.dayRankCounts[previousRank])
				}
				if state.lastRank >= 0 && state.lastRank != rank {
					next.score.preparationTransitions++
				}
				if state.lastCampus != "" && state.lastCampus != option.CampusID {
					next.score.campusChanges++
				}
				if state.lastBuilding != "" && state.lastBuilding != option.Building {
					next.score.buildingChanges++
				}
			} else {
				next.dayRankCounts = [3]uint8{}
				next.drinkToday = false
				next.lastNoWaterEndMinutes = 0
				next.score.serviceDays++
			}
			next.dayRankCounts[rank]++
			// 空腹在本系统中等同于禁食加禁水，因此同样会延后后续饮水准备的起点。
			if rank == 0 {
				next.lastNoWaterEndMinutes = end
			}
			if hasPreparationRule(configuration, descriptionrules.RuleTypeDrinkWater) {
				next.drinkToday = true
			}
			next.score.preparationMaturityPenalty += maturityPenalty
			next.score.travelMinutes += int(travel.minutes)
			next.score.completionDate = option.ServiceDate
			next.score.completionMinutes = end

			if capacityIndex >= 0 {
				s.capacityUsage[capacityIndex]++
			}
			s.path = append(s.path, searchChoice{
				itemID: itemID, assignmentKey: candidate.assignmentKey, option: option, plannedStartMinutes: earliestStart, plannedEndMinutes: end,
				travelMinutes: travel.minutes, travelEstimated: travel.estimated,
			})
			s.visit(next)
			s.path = s.path[:len(s.path)-1]
			if capacityIndex >= 0 {
				s.capacityUsage[capacityIndex]--
			}
		}
	}
}

// canStillReachTopPlans 使用乐观下界剪枝。准备顺序、服务日和累计惩罚均只会增加；
// 当前结束时刻加上剩余项目最短检查时长，是最终完成时刻的安全下界。
func (s *planSearcher) canStillReachTopPlans(state searchState) bool {
	if len(s.results) < maximumGeneratedPlans {
		return true
	}
	third := s.results[maximumGeneratedPlans-1].score
	partialNumbers := []int{state.score.preparationInversions, state.score.preparationTransitions, state.score.serviceDays}
	thirdNumbers := []int{third.preparationInversions, third.preparationTransitions, third.serviceDays}
	for index := range partialNumbers {
		if partialNumbers[index] < thirdNumbers[index] {
			return true
		}
		if partialNumbers[index] > thirdNumbers[index] {
			return false
		}
	}
	if state.lastDate == "" {
		return true
	}
	lowerBound := state.cursorMinutes
	for itemIndex := range s.itemIDs {
		if state.selectedMask&(1<<itemIndex) != 0 {
			continue
		}
		minimumDuration := int32(24 * 60)
		for _, option := range s.options[itemIndex] {
			if option.duration < minimumDuration {
				minimumDuration = option.duration
			}
		}
		if minimumDuration < 24*60 {
			lowerBound += minimumDuration
		}
	}
	if state.lastDate < third.completionDate {
		return true
	}
	if state.lastDate > third.completionDate {
		return false
	}
	if lowerBound >= 24*60 {
		return false
	}
	if lowerBound < third.completionMinutes {
		return true
	}
	if lowerBound > third.completionMinutes {
		return false
	}
	remainingNumbers := []int{state.score.preparationMaturityPenalty, state.score.campusChanges, state.score.buildingChanges, state.score.travelMinutes}
	thirdRemainingNumbers := []int{third.preparationMaturityPenalty, third.campusChanges, third.buildingChanges, third.travelMinutes}
	for index := range remainingNumbers {
		if remainingNumbers[index] < thirdRemainingNumbers[index] {
			return true
		}
		if remainingNumbers[index] > thirdRemainingNumbers[index] {
			return false
		}
	}
	return true
}

func (s *planSearcher) acceptState(state searchState) bool {
	key := searchMemoKey{
		selectedMask: state.selectedMask, lastDate: state.lastDate, cursorMinutes: state.cursorMinutes,
		lastRoomID: state.lastRoomID, lastCampus: state.lastCampus, lastBuilding: state.lastBuilding,
		lastNoWaterEndMinutes: state.lastNoWaterEndMinutes, lastRank: state.lastRank,
		dayRankCounts: state.dayRankCounts, drinkToday: state.drinkToday, capacityUsage: string(s.capacityUsage),
	}
	best := s.memo[key]
	if best.count == maximumGeneratedPlans && compareSearchScore(state.score, best.values[best.count-1]) >= 0 {
		return false
	}
	insertAt := int(best.count)
	for index := 0; index < int(best.count); index++ {
		if compareSearchScore(state.score, best.values[index]) < 0 {
			insertAt = index
			break
		}
	}
	limit := int(best.count)
	if limit < maximumGeneratedPlans {
		limit++
	}
	for index := limit - 1; index > insertAt; index-- {
		best.values[index] = best.values[index-1]
	}
	if insertAt < maximumGeneratedPlans {
		best.values[insertAt] = state.score
	}
	best.count = uint8(limit)
	s.memo[key] = best
	return true
}

func (s *planSearcher) addResult(state searchState) {
	keyParts := make([]string, 0, len(s.path))
	for _, choice := range s.path {
		keyParts = append(keyParts, choice.assignmentKey)
	}
	sort.Strings(keyParts)
	key := strings.Join(keyParts, "|")
	for index, candidate := range s.results {
		if candidate.key != key {
			continue
		}
		if compareSearchScore(state.score, candidate.score) >= 0 {
			return
		}
		s.results[index] = searchCandidate{choices: append([]searchChoice(nil), s.path...), score: state.score, key: key}
		s.sortAndTrimResults()
		return
	}
	s.results = append(s.results, searchCandidate{choices: append([]searchChoice(nil), s.path...), score: state.score, key: key})
	s.sortAndTrimResults()
}

func (s *planSearcher) sortAndTrimResults() {
	sort.SliceStable(s.results, func(i, j int) bool {
		if compared := compareSearchScore(s.results[i].score, s.results[j].score); compared != 0 {
			return compared < 0
		}
		return s.results[i].key < s.results[j].key
	})
	if len(s.results) > maximumGeneratedPlans {
		s.results = s.results[:maximumGeneratedPlans]
	}
}

func normalizedSearchOptions(values []Option) []Option {
	byKey := make(map[string]Option, len(values))
	for _, value := range values {
		key := optionCapacityKey(value)
		if existing, ok := byKey[key]; !ok || value.RemainingCapacity < existing.RemainingCapacity {
			byKey[key] = value
		}
	}
	result := make([]Option, 0, len(byKey))
	for _, value := range byKey {
		result = append(result, value)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].ServiceDate != result[j].ServiceDate {
			return result[i].ServiceDate < result[j].ServiceDate
		}
		leftStart, _, _, _ := effectiveOptionWindow(result[i])
		rightStart, _, _, _ := effectiveOptionWindow(result[j])
		if leftStart != rightStart {
			return leftStart < rightStart
		}
		if result[i].CampusID != result[j].CampusID {
			return result[i].CampusID < result[j].CampusID
		}
		if result[i].Building != result[j].Building {
			return result[i].Building < result[j].Building
		}
		return result[i].RoomID < result[j].RoomID
	})
	return result
}

func compareSearchScore(left, right searchScore) int {
	leftValues := []int{left.preparationInversions, left.preparationTransitions, left.serviceDays}
	rightValues := []int{right.preparationInversions, right.preparationTransitions, right.serviceDays}
	for index := range leftValues {
		if compared := compareInt(leftValues[index], rightValues[index]); compared != 0 {
			return compared
		}
	}
	if left.completionDate < right.completionDate {
		return -1
	}
	if left.completionDate > right.completionDate {
		return 1
	}
	if left.completionMinutes < right.completionMinutes {
		return -1
	}
	if left.completionMinutes > right.completionMinutes {
		return 1
	}
	leftValues = []int{left.preparationMaturityPenalty, left.campusChanges, left.buildingChanges, left.travelMinutes}
	rightValues = []int{right.preparationMaturityPenalty, right.campusChanges, right.buildingChanges, right.travelMinutes}
	for index := range leftValues {
		if compared := compareInt(leftValues[index], rightValues[index]); compared != 0 {
			return compared
		}
	}
	return 0
}

// preparationMaturityPenalty 统一描述禁食、禁水和饮水等准备规则的成熟度：
// 最短时长是可执行硬门槛；推荐区间内越接近充分准备上限越好；超过上限仍可执行，
// 但不会因为无限延长准备时间而得到更高评分。
func preparationMaturityPenalty(rule descriptionrules.Rule, elapsedMinutes int32) int {
	target := rule.MaxAdvanceMinutes
	if target <= 0 {
		target = rule.RecommendedAdvanceMinutes
	}
	if target <= 0 {
		target = rule.MinAdvanceMinutes
	}
	return absoluteInt(int(elapsedMinutes - target))
}

func hasPreparationRule(configuration projectconfiguration.Configuration, ruleType descriptionrules.RuleType) bool {
	_, ok := preparationRule(configuration, ruleType)
	return ok
}

func preparationRule(configuration projectconfiguration.Configuration, ruleType descriptionrules.RuleType) (descriptionrules.Rule, bool) {
	for _, rule := range configuration.PreparationRules {
		if rule.RuleType == ruleType {
			return rule, true
		}
	}
	return descriptionrules.Rule{}, false
}

func absoluteInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func compareInt(left, right int) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
