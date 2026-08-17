package planning

import (
	"sort"
	"strings"

	"hospital/service/guidance/rpc/internal/projectconfiguration"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

const maximumGeneratedPlans = 3

type searchChoice struct {
	itemID string
	option Option
}

type searchOptionCandidate struct {
	option        Option
	slot          string
	capacityIndex int
}

type searchScore struct {
	preparationInversions  int
	preparationTransitions int
	serviceDays            int
	campusChanges          int
	buildingChanges        int
	completionSlot         string
}

type searchCandidate struct {
	choices []searchChoice
	score   searchScore
	key     string
}

// searchState 只保存会影响后续选择的状态。当前路径和容量占用由搜索器原地回溯，
// 避免在每个 DFS 分支复制切片和 map。
type searchState struct {
	selectedMask  uint16
	lastSlot      string
	lastDate      string
	lastCampus    string
	lastBuilding  string
	lastRank      int8
	dayRankCounts [3]uint8
	drinkToday    bool
	score         searchScore
}

// searchMemoKey 表示“从此处继续搜索时完全等价”的后缀状态。
// capacityUsage 只包含确实可能发生争抢的房间容量，专属房间和充足容量不会扩大状态空间。
type searchMemoKey struct {
	selectedMask  uint16
	lastSlot      string
	lastCampus    string
	lastBuilding  string
	lastRank      int8
	dayRankCounts [3]uint8
	drinkToday    bool
	capacityUsage string
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

	capacityIndexes map[string]int
	capacityLimits  []uint8
	capacityUsage   []uint8

	path    []searchChoice
	memo    map[searchMemoKey]searchMemoScores
	results []searchCandidate
}

// searchBestPlans 使用完整 DFS 搜索宏观顺序和预约选项；状态记忆化合并等价后缀，
// 再以当前前三名作为安全下界剪枝。项目数上限为十，selectedMask 可完整表达已安排集合。
func searchBestPlans(itemIDs []string, options map[string][]Option, rules []precedence.Rule, configurations map[string]projectconfiguration.Configuration) []searchCandidate {
	searcher := &planSearcher{
		itemIDs:         append([]string(nil), itemIDs...),
		itemIndex:       make(map[string]int, len(itemIDs)),
		options:         make([][]searchOptionCandidate, len(itemIDs)),
		configurations:  configurations,
		prerequisites:   make([]uint16, len(itemIDs)),
		capacityIndexes: map[string]int{},
		path:            make([]searchChoice, 0, len(itemIDs)),
		memo:            map[searchMemoKey]searchMemoScores{},
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
	// 只有“使用该容量的项目数 > 剩余容量”时，容量才可能影响可行性。
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
		searcher.options[itemIndex] = make([]searchOptionCandidate, 0, len(normalizedByItem[itemIndex]))
		for _, option := range normalizedByItem[itemIndex] {
			capacityIndex := -1
			if index, ok := searcher.capacityIndexes[optionCapacityKey(option)]; ok {
				capacityIndex = index
			}
			searcher.options[itemIndex] = append(searcher.options[itemIndex], searchOptionCandidate{
				option: option, slot: optionSlot(option), capacityIndex: capacityIndex,
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
	if len(s.path) == len(s.itemIDs) {
		s.addResult(state)
		return
	}
	// 所有评分项在继续搜索时只会保持或增大；部分路径已经严格差于第三名时不可能翻盘。
	if len(s.results) == maximumGeneratedPlans && compareSearchScore(state.score, s.results[maximumGeneratedPlans-1].score) > 0 {
		return
	}
	if !s.acceptState(state) {
		return
	}

	for itemIndex, itemID := range s.itemIDs {
		bit := uint16(1 << itemIndex)
		if state.selectedMask&bit != 0 || state.selectedMask&s.prerequisites[itemIndex] != s.prerequisites[itemIndex] {
			continue
		}
		for _, candidate := range s.options[itemIndex] {
			option := candidate.option
			if state.lastSlot != "" && candidate.slot < state.lastSlot {
				continue
			}
			capacityIndex := candidate.capacityIndex
			constrainedCapacity := capacityIndex >= 0
			if constrainedCapacity && s.capacityUsage[capacityIndex] >= s.capacityLimits[capacityIndex] {
				continue
			}

			rank := int8(descriptionRuleRank(s.configurations[itemID]))
			sameDate := state.lastDate == option.ServiceDate
			// 喝水状态已经生效后，同一天不能再安排禁水/禁食项目。
			if sameDate && state.drinkToday && rank == 0 {
				continue
			}

			next := state
			next.selectedMask |= bit
			next.lastSlot = candidate.slot
			next.lastDate = option.ServiceDate
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
				next.score.serviceDays++
			}
			next.dayRankCounts[rank]++
			if rank == 2 {
				next.drinkToday = true
			}
			next.score.completionSlot = candidate.slot

			if constrainedCapacity {
				s.capacityUsage[capacityIndex]++
			}
			s.path = append(s.path, searchChoice{itemID: itemID, option: option})
			s.visit(next)
			s.path = s.path[:len(s.path)-1]
			if constrainedCapacity {
				s.capacityUsage[capacityIndex]--
			}
		}
	}
}

// acceptState 对同一个后缀状态保留最多三个最优前缀。
// 只保留一个足以保证第一名正确；保留三个可以继续生成稳定且不同的备选方案。
func (s *planSearcher) acceptState(state searchState) bool {
	key := searchMemoKey{
		selectedMask:  state.selectedMask,
		lastSlot:      state.lastSlot,
		lastCampus:    state.lastCampus,
		lastBuilding:  state.lastBuilding,
		lastRank:      state.lastRank,
		dayRankCounts: state.dayRankCounts,
		drinkToday:    state.drinkToday,
		capacityUsage: string(s.capacityUsage),
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
		keyParts = append(keyParts, choice.itemID+"@"+optionCapacityKey(choice.option))
	}
	// 同一组项目到预约选项的分配，只是遍历顺序不同，并不是患者可选择的不同方案。
	// 保留评分最优的解释顺序，避免用大窗内部不可承诺的排列凑足三个方案。
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
		// 同一个房间、日期和时段本应只有一份容量事实；若上游意外返回重复值，
		// 使用更小的剩余容量，宁可少给方案也不能生成可能超卖的方案。
		if existing, ok := byKey[key]; !ok || value.RemainingCapacity < existing.RemainingCapacity {
			byKey[key] = value
		}
	}
	result := make([]Option, 0, len(byKey))
	for _, value := range byKey {
		result = append(result, value)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if left, right := optionSlot(result[i]), optionSlot(result[j]); left != right {
			return left < right
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
	if compared := compareInt(left.preparationInversions, right.preparationInversions); compared != 0 {
		return compared
	}
	if compared := compareInt(left.preparationTransitions, right.preparationTransitions); compared != 0 {
		return compared
	}
	if compared := compareInt(left.serviceDays, right.serviceDays); compared != 0 {
		return compared
	}
	if compared := compareInt(left.campusChanges, right.campusChanges); compared != 0 {
		return compared
	}
	if compared := compareInt(left.buildingChanges, right.buildingChanges); compared != 0 {
		return compared
	}
	if left.completionSlot < right.completionSlot {
		return -1
	}
	if left.completionSlot > right.completionSlot {
		return 1
	}
	return 0
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
