package planning

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"hospital/service/guidance/rpc/internal/projectconfiguration"
	descriptionrules "hospital/service/guidance/rpc/internal/rules/description"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

const (
	searchItemA = "10000000-0000-4000-8000-000000000001"
	searchItemB = "10000000-0000-4000-8000-000000000002"
	searchItemC = "10000000-0000-4000-8000-000000000003"
	searchRoomA = "20000000-0000-4000-8000-000000000001"
	searchRoomB = "20000000-0000-4000-8000-000000000002"
	searchRoomC = "20000000-0000-4000-8000-000000000003"
)

func TestGenerateFindsFeasiblePlanInsteadOfConsumingSharedCapacityGreedily(t *testing.T) {
	manager := newSearchTestManager(
		[]string{searchItemA, searchItemB},
		nil,
		map[string][]Option{
			searchItemA: {
				searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-18", "morning", 1),
				searchOption(searchItemA, searchRoomB, "2号楼", "2026-08-18", "morning", 1),
			},
			searchItemB: {searchOption(searchItemB, searchRoomA, "1号楼", "2026-08-18", "morning", 1)},
		},
		nil,
	)

	plans, err := manager.Generate(context.Background(), patientPrincipal(), searchCommand(searchItemA, searchItemB))
	if err != nil {
		t.Fatalf("Generate() should find the alternative room, got %v", err)
	}
	if got := plans[0].Items[0].RoomID; got != searchRoomB {
		t.Fatalf("first item should move to the alternative room, got %s", got)
	}
}

func TestGenerateChoosesGlobalBuildingOptimumInsteadOfFirstLocalOption(t *testing.T) {
	rules := []precedence.Rule{
		{PredecessorItemID: searchItemA, SuccessorItemID: searchItemB},
		{PredecessorItemID: searchItemB, SuccessorItemID: searchItemC},
	}
	manager := newSearchTestManager(
		[]string{searchItemA, searchItemB, searchItemC},
		nil,
		map[string][]Option{
			searchItemA: {
				searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-18", "morning", 3),
				searchOption(searchItemA, searchRoomB, "2号楼", "2026-08-18", "morning", 3),
			},
			searchItemB: {searchOption(searchItemB, searchRoomB, "2号楼", "2026-08-18", "morning", 3)},
			searchItemC: {searchOption(searchItemC, searchRoomC, "2号楼", "2026-08-18", "morning", 3)},
		},
		rules,
	)

	plans, err := manager.Generate(context.Background(), patientPrincipal(), searchCommand(searchItemA, searchItemB, searchItemC))
	if err != nil {
		t.Fatal(err)
	}
	if got := plans[0].Items[0].Building; got != "2号楼" {
		t.Fatalf("global optimum should keep all projects in 2号楼, got first building %s", got)
	}
}

func TestGenerateRejectsChronologicallyImpossiblePrecedence(t *testing.T) {
	manager := newSearchTestManager(
		[]string{searchItemA, searchItemB},
		nil,
		map[string][]Option{
			searchItemA: {searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-18", "afternoon", 2)},
			searchItemB: {searchOption(searchItemB, searchRoomB, "1号楼", "2026-08-18", "morning", 2)},
		},
		[]precedence.Rule{{PredecessorItemID: searchItemA, SuccessorItemID: searchItemB}},
	)

	_, err := manager.Generate(context.Background(), patientPrincipal(), searchCommand(searchItemA, searchItemB))
	if !errors.Is(err, ErrNoPlan) {
		t.Fatalf("expected ErrNoPlan, got %v", err)
	}
}

func TestGeneratePreparationOrderAndPrecedenceHaveStablePriority(t *testing.T) {
	tests := []struct {
		name           string
		rules          []precedence.Rule
		configurations map[string][]descriptionrules.Rule
		want           []string
	}{
		{
			name: "preparation order",
			configurations: map[string][]descriptionrules.Rule{
				searchItemA: {{RuleType: descriptionrules.RuleTypeDrinkWater}},
				searchItemB: nil,
				searchItemC: {{RuleType: descriptionrules.RuleTypeFasting}},
			},
			want: []string{searchItemC, searchItemB, searchItemA},
		},
		{
			name:  "hard precedence overrides preparation preference",
			rules: []precedence.Rule{{PredecessorItemID: searchItemA, SuccessorItemID: searchItemC}},
			configurations: map[string][]descriptionrules.Rule{
				searchItemA: nil,
				searchItemB: {{RuleType: descriptionrules.RuleTypeDrinkWater}},
				searchItemC: {{RuleType: descriptionrules.RuleTypeFasting}},
			},
			want: []string{searchItemA, searchItemC, searchItemB},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := map[string][]Option{}
			for index, itemID := range []string{searchItemA, searchItemB, searchItemC} {
				roomID := []string{searchRoomA, searchRoomB, searchRoomC}[index]
				options[itemID] = []Option{searchOption(itemID, roomID, "1号楼", "2026-08-18", "morning", 3)}
			}
			manager := newSearchTestManager([]string{searchItemA, searchItemB, searchItemC}, test.configurations, options, test.rules)
			plans, err := manager.Generate(context.Background(), patientPrincipal(), searchCommand(searchItemA, searchItemB, searchItemC))
			if err != nil {
				t.Fatal(err)
			}
			for index, itemID := range test.want {
				if got := plans[0].Items[index].ItemID; got != itemID {
					t.Fatalf("item %d = %s, want %s", index, got, itemID)
				}
			}
		})
	}
}

func TestSearchMatchesIndependentExhaustiveOracle(t *testing.T) {
	itemIDs := []string{searchItemA, searchItemB, searchItemC}
	allEdges := []precedence.Rule{
		{PredecessorItemID: searchItemA, SuccessorItemID: searchItemB},
		{PredecessorItemID: searchItemA, SuccessorItemID: searchItemC},
		{PredecessorItemID: searchItemB, SuccessorItemID: searchItemC},
	}
	options := map[string][]Option{
		searchItemA: {
			searchOption(searchItemA, "21000000-0000-4000-8000-000000000001", "1号楼", "2026-08-18", "morning", 3),
			searchOption(searchItemA, "22000000-0000-4000-8000-000000000001", "2号楼", "2026-08-18", "morning", 3),
		},
		searchItemB: {
			searchOption(searchItemB, "21000000-0000-4000-8000-000000000002", "1号楼", "2026-08-18", "morning", 3),
			searchOption(searchItemB, "22000000-0000-4000-8000-000000000002", "2号楼", "2026-08-18", "morning", 3),
		},
		searchItemC: {
			searchOption(searchItemC, "21000000-0000-4000-8000-000000000003", "1号楼", "2026-08-18", "morning", 3),
			searchOption(searchItemC, "22000000-0000-4000-8000-000000000003", "2号楼", "2026-08-18", "morning", 3),
		},
	}

	caseCount := 0
	for rankCode := 0; rankCode < 27; rankCode++ {
		configurations := exhaustiveConfigurations(itemIDs, rankCode)
		for edgeMask := 0; edgeMask < 1<<len(allEdges); edgeMask++ {
			rules := make([]precedence.Rule, 0, len(allEdges))
			for edgeIndex, edge := range allEdges {
				if edgeMask&(1<<edgeIndex) != 0 {
					rules = append(rules, edge)
				}
			}
			want, feasible := exhaustiveOracle(itemIDs, options, rules, configurations)
			got := searchBestPlans(itemIDs, options, rules, configurations)
			caseCount++
			if !feasible {
				if len(got) != 0 {
					t.Fatalf("rankCode=%d edgeMask=%d: oracle found no feasible plan, search returned %#v", rankCode, edgeMask, got[0])
				}
				continue
			}
			if len(got) == 0 {
				t.Fatalf("rankCode=%d edgeMask=%d: search missed a feasible plan", rankCode, edgeMask)
			}
			if actual := oracleScoreForChoices(got[0].choices, configurations); actual != want {
				t.Fatalf("rankCode=%d edgeMask=%d: score=%v, exhaustive optimum=%v", rankCode, edgeMask, actual, want)
			}
		}
	}
	if caseCount != 216 {
		t.Fatalf("unexpected exhaustive case count %d", caseCount)
	}
}

func TestGenerateDoesNotPlaceRestrictedPreparationAfterDrinkingOnSameDay(t *testing.T) {
	preparation := map[string][]descriptionrules.Rule{
		searchItemA: {{RuleType: descriptionrules.RuleTypeDrinkWater}},
		searchItemB: {{RuleType: descriptionrules.RuleTypeFasting}},
	}
	rules := []precedence.Rule{{PredecessorItemID: searchItemA, SuccessorItemID: searchItemB}}
	options := map[string][]Option{
		searchItemA: {searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-18", "morning", 2)},
		searchItemB: {
			searchOption(searchItemB, searchRoomB, "1号楼", "2026-08-18", "afternoon", 2),
			searchOption(searchItemB, searchRoomB, "1号楼", "2026-08-19", "morning", 2),
		},
	}
	manager := newSearchTestManager([]string{searchItemA, searchItemB}, preparation, options, rules)
	plans, err := manager.Generate(context.Background(), patientPrincipal(), GenerateCommand{
		ItemIDs: []string{searchItemA, searchItemB},
		CandidateAvailability: []CandidateAvailability{
			{ServiceDate: "2026-08-18", Sessions: []string{"morning", "afternoon"}},
			{ServiceDate: "2026-08-19", Sessions: []string{"morning"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := plans[0].Items[1].ServiceDate; got != "2026-08-19" {
		t.Fatalf("restricted project must move to the next day after drinking, got %s", got)
	}

	options[searchItemB] = options[searchItemB][:1]
	manager = newSearchTestManager([]string{searchItemA, searchItemB}, preparation, options, rules)
	_, err = manager.Generate(context.Background(), patientPrincipal(), searchCommand(searchItemA, searchItemB))
	if !errors.Is(err, ErrNoPlan) {
		t.Fatalf("same-day drink-before-fasting plan must be rejected, got %v", err)
	}
}

func TestGenerateRejectsCyclicPrecedence(t *testing.T) {
	manager := newSearchTestManager(
		[]string{searchItemA, searchItemB},
		nil,
		map[string][]Option{
			searchItemA: {searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-18", "morning", 2)},
			searchItemB: {searchOption(searchItemB, searchRoomB, "1号楼", "2026-08-18", "morning", 2)},
		},
		[]precedence.Rule{
			{PredecessorItemID: searchItemA, SuccessorItemID: searchItemB},
			{PredecessorItemID: searchItemB, SuccessorItemID: searchItemA},
		},
	)

	_, err := manager.Generate(context.Background(), patientPrincipal(), searchCommand(searchItemA, searchItemB))
	if !errors.Is(err, ErrNoPlan) {
		t.Fatalf("cyclic precedence must not produce a plan, got %v", err)
	}
}

func TestSearchUsesExactWindowIntersectionAndEstimatedDuration(t *testing.T) {
	option := searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-18", "morning", 1)
	option.RoomOpenTime, option.RoomCloseTime = "08:00:00", "11:00:00"
	option.ItemStartTime, option.ItemEndTime = "08:30:00", "10:00:00"
	option.EstimatedDurationMinutes = 60

	plans := searchBestPlans([]string{searchItemA}, map[string][]Option{searchItemA: {option}}, nil, map[string]projectconfiguration.Configuration{searchItemA: {ItemID: searchItemA}})
	if len(plans) != 1 {
		t.Fatalf("expected an executable plan, got %d", len(plans))
	}
	if got := plans[0].choices[0]; got.plannedStartMinutes != 8*60+30 || got.plannedEndMinutes != 9*60+30 {
		t.Fatalf("planned interval = %s-%s, want 08:30-09:30", independentClock(got.plannedStartMinutes), independentClock(got.plannedEndMinutes))
	}
}

func TestSearchRejectsDurationsThatCannotFitSameWindow(t *testing.T) {
	itemIDs := []string{searchItemA, searchItemB, searchItemC}
	options := map[string][]Option{}
	configurations := map[string]projectconfiguration.Configuration{}
	for index, itemID := range itemIDs {
		option := searchOption(itemID, []string{searchRoomA, searchRoomB, searchRoomC}[index], "1号楼", "2026-08-18", "morning", 3)
		option.EstimatedDurationMinutes = []int32{90, 60, 60}[index]
		option.ItemStartTime, option.ItemEndTime = "09:00", "12:00"
		options[itemID] = []Option{option}
		configurations[itemID] = projectconfiguration.Configuration{ItemID: itemID}
	}
	if plans := searchBestPlans(itemIDs, options, nil, configurations); len(plans) != 0 {
		t.Fatalf("210 minutes of examinations plus movement cannot fit in a 180-minute window")
	}
}

func TestSearchStartsDrinkPreparationAfterLastNoWaterExamination(t *testing.T) {
	noWater := searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-18", "morning", 2)
	noWater.ItemStartTime, noWater.ItemEndTime, noWater.EstimatedDurationMinutes = "09:00", "13:00", 60
	drink := searchOption(searchItemB, searchRoomA, "1号楼", "2026-08-18", "morning", 2)
	drink.ItemStartTime, drink.ItemEndTime, drink.EstimatedDurationMinutes = "09:00", "13:00", 30
	configurations := map[string]projectconfiguration.Configuration{
		searchItemA: {ItemID: searchItemA, PreparationRules: []descriptionrules.Rule{{RuleType: descriptionrules.RuleTypeNoWater}}},
		searchItemB: {ItemID: searchItemB, PreparationRules: []descriptionrules.Rule{{RuleType: descriptionrules.RuleTypeDrinkWater, StartMode: descriptionrules.StartModeAdvanceRange, MinAdvanceMinutes: 120, RecommendedAdvanceMinutes: 180, MaxAdvanceMinutes: 240}}},
	}
	rules := []precedence.Rule{{PredecessorItemID: searchItemA, SuccessorItemID: searchItemB}}
	plans := searchBestPlans([]string{searchItemA, searchItemB}, map[string][]Option{searchItemA: {noWater}, searchItemB: {drink}}, rules, configurations)
	if len(plans) == 0 {
		t.Fatal("expected drink project to become ready two hours after no-water project finishes")
	}
	if got := plans[0].choices[1]; got.plannedStartMinutes != 12*60 || got.plannedEndMinutes != 12*60+30 {
		t.Fatalf("drink project interval = %s-%s, want 12:00-12:30", independentClock(got.plannedStartMinutes), independentClock(got.plannedEndMinutes))
	}

	drink.ItemEndTime = "12:00"
	if plans := searchBestPlans([]string{searchItemA, searchItemB}, map[string][]Option{searchItemA: {noWater}, searchItemB: {drink}}, rules, configurations); len(plans) != 0 {
		t.Fatal("readiness plus examination duration does not fit before 12:00")
	}
}

func TestSearchTreatsDrinkMaximumAsSoftPreference(t *testing.T) {
	noWater := searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-18", "morning", 2)
	noWater.ItemStartTime, noWater.ItemEndTime, noWater.EstimatedDurationMinutes = "09:00", "10:00", 30
	drink := searchOption(searchItemB, searchRoomA, "1号楼", "2026-08-18", "afternoon", 2)
	drink.ItemStartTime, drink.ItemEndTime, drink.EstimatedDurationMinutes = "14:00", "15:00", 15
	configurations := map[string]projectconfiguration.Configuration{
		searchItemA: {ItemID: searchItemA, PreparationRules: []descriptionrules.Rule{{RuleType: descriptionrules.RuleTypeNoWater}}},
		searchItemB: {ItemID: searchItemB, PreparationRules: []descriptionrules.Rule{{RuleType: descriptionrules.RuleTypeDrinkWater, StartMode: descriptionrules.StartModeAdvanceRange, MinAdvanceMinutes: 120, RecommendedAdvanceMinutes: 180, MaxAdvanceMinutes: 240}}},
	}
	plans := searchBestPlans([]string{searchItemA, searchItemB}, map[string][]Option{searchItemA: {noWater}, searchItemB: {drink}}, []precedence.Rule{{PredecessorItemID: searchItemA, SuccessorItemID: searchItemB}}, configurations)
	if len(plans) == 0 {
		t.Fatal("exceeding the recommended maximum remains feasible")
	}
}

func TestSearchIncludesMovementInWindowFeasibility(t *testing.T) {
	first := searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-18", "morning", 2)
	first.ItemStartTime, first.ItemEndTime, first.EstimatedDurationMinutes = "11:00", "12:00", 50
	second := searchOption(searchItemB, searchRoomB, "2号楼", "2026-08-18", "morning", 2)
	second.ItemStartTime, second.ItemEndTime, second.EstimatedDurationMinutes = "11:00", "12:00", 10
	configurations := map[string]projectconfiguration.Configuration{searchItemA: {ItemID: searchItemA}, searchItemB: {ItemID: searchItemB}}
	rules := []precedence.Rule{{PredecessorItemID: searchItemA, SuccessorItemID: searchItemB}}
	if plans := searchBestPlans([]string{searchItemA, searchItemB}, map[string][]Option{searchItemA: {first}, searchItemB: {second}}, rules, configurations); len(plans) != 0 {
		t.Fatal("cross-building fallback travel makes the second examination miss its window")
	}

	second.RoomID, second.Building = searchRoomA, "1号楼"
	plans := searchBestPlans([]string{searchItemA, searchItemB}, map[string][]Option{searchItemA: {first}, searchItemB: {second}}, rules, configurations)
	if len(plans) == 0 || plans[0].choices[1].travelMinutes != 0 || plans[0].choices[1].plannedEndMinutes != 12*60 {
		t.Fatalf("same room should require no movement and finish at 12:00: %#v", plans)
	}
}

type travelEstimatorStub struct {
	minutes int32
	err     error
	calls   int
}

func (s *travelEstimatorStub) EstimateWalkingMinutes(context.Context, string, string, string) (int32, error) {
	s.calls++
	return s.minutes, s.err
}

func TestEstimateBuildingTravelUsesProviderAndConservativeFallback(t *testing.T) {
	options := map[string][]Option{
		searchItemA: {searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-18", "morning", 1)},
		searchItemB: {searchOption(searchItemB, searchRoomB, "2号楼", "2026-08-18", "morning", 1)},
	}
	provider := &travelEstimatorStub{minutes: 7}
	got := estimateBuildingTravel(context.Background(), provider, options)[travelPairKey("1号楼", "2号楼")]
	if provider.calls != 1 || got.minutes != 10 || got.estimated {
		t.Fatalf("provider estimate = %#v, calls=%d, want 10 exact minutes", got, provider.calls)
	}
	provider = &travelEstimatorStub{err: errors.New("map unavailable")}
	got = estimateBuildingTravel(context.Background(), provider, options)[travelPairKey("1号楼", "2号楼")]
	if got.minutes != 15 || !got.estimated {
		t.Fatalf("fallback estimate = %#v, want 15 estimated minutes", got)
	}
}

func TestManagerCachesBuildingTravelAcrossPlanRequests(t *testing.T) {
	options := map[string][]Option{
		searchItemA: {searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-18", "morning", 1)},
		searchItemB: {searchOption(searchItemB, searchRoomB, "2号楼", "2026-08-18", "morning", 1)},
	}
	provider := &travelEstimatorStub{minutes: 8}
	manager := &Manager{travel: provider, travelCache: map[string]cachedTravelEstimate{}}
	manager.estimateBuildingTravel(context.Background(), options)
	manager.estimateBuildingTravel(context.Background(), options)
	if provider.calls != 1 {
		t.Fatalf("same building pair should be fetched once while cached, calls=%d", provider.calls)
	}
}

func TestGeneratePrefersFewerServiceDays(t *testing.T) {
	manager := newSearchTestManager(
		[]string{searchItemA, searchItemB},
		nil,
		map[string][]Option{
			searchItemA: {
				searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-18", "morning", 2),
				searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-19", "morning", 2),
			},
			searchItemB: {
				searchOption(searchItemB, searchRoomB, "1号楼", "2026-08-18", "afternoon", 2),
				searchOption(searchItemB, searchRoomB, "1号楼", "2026-08-19", "afternoon", 2),
			},
		},
		nil,
	)

	plans, err := manager.Generate(context.Background(), patientPrincipal(), GenerateCommand{
		ItemIDs: []string{searchItemA, searchItemB},
		CandidateAvailability: []CandidateAvailability{
			{ServiceDate: "2026-08-18", Sessions: []string{"morning", "afternoon"}},
			{ServiceDate: "2026-08-19", Sessions: []string{"morning", "afternoon"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plans[0].Items[0].ServiceDate != plans[0].Items[1].ServiceDate {
		t.Fatalf("best plan should keep both projects on one service day: %#v", plans[0].Items)
	}
}

func TestGenerateUsesExistingBookingsAsPrecedenceAnchors(t *testing.T) {
	tests := []struct {
		name         string
		selectedItem string
		bookings     map[string][]Booking
		wantSession  string
	}{
		{
			name:         "existing successor keeps new predecessor before it",
			selectedItem: searchItemA,
			bookings: map[string][]Booking{
				"active": {{ItemID: searchItemB, ServiceDate: "2026-08-18", Session: "morning", Status: "confirmed"}},
			},
			wantSession: "morning",
		},
		{
			name:         "completed predecessor allows only later new successor",
			selectedItem: searchItemB,
			bookings: map[string][]Booking{
				"completed": {{ItemID: searchItemA, ServiceDate: "2026-08-18", Session: "afternoon", Status: "completed"}},
			},
			wantSession: "afternoon",
		},
		{
			name:         "no show does not satisfy predecessor",
			selectedItem: searchItemB,
			bookings: map[string][]Booking{
				"completed": {{ItemID: searchItemA, ServiceDate: "2026-08-18", Session: "afternoon", Status: "no_show"}},
			},
			wantSession: "morning",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			roomID := searchRoomA
			if test.selectedItem == searchItemB {
				roomID = searchRoomB
			}
			manager := newSearchTestManager(
				[]string{test.selectedItem},
				nil,
				map[string][]Option{
					test.selectedItem: {
						searchOption(test.selectedItem, roomID, "1号楼", "2026-08-18", "morning", 2),
						searchOption(test.selectedItem, roomID, "1号楼", "2026-08-18", "afternoon", 2),
					},
				},
				[]precedence.Rule{{PredecessorItemID: searchItemA, SuccessorItemID: searchItemB}},
			)
			manager.appointment.(*planningAppointmentStub).bookings = test.bookings

			plans, err := manager.Generate(context.Background(), patientPrincipal(), searchCommand(test.selectedItem))
			if err != nil {
				t.Fatal(err)
			}
			if got := plans[0].Items[0].Session; got != test.wantSession {
				t.Fatalf("session=%s, want %s", got, test.wantSession)
			}
		})
	}
}

func TestSearchUsesConservativeCapacityWhenUpstreamReturnsDuplicateOptions(t *testing.T) {
	low := searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-18", "morning", 1)
	high := low
	high.RemainingCapacity = 3

	got := normalizedSearchOptions([]Option{high, low})
	if len(got) != 1 || got[0].RemainingCapacity != 1 {
		t.Fatalf("duplicate capacity facts must use the conservative minimum, got %#v", got)
	}
}

func TestSearchDoesNotReturnPermutationOnlyDuplicates(t *testing.T) {
	itemIDs := []string{searchItemA, searchItemB, searchItemC}
	options := map[string][]Option{
		searchItemA: {searchOption(searchItemA, searchRoomA, "1号楼", "2026-08-18", "morning", 3)},
		searchItemB: {searchOption(searchItemB, searchRoomB, "1号楼", "2026-08-18", "morning", 3)},
		searchItemC: {searchOption(searchItemC, searchRoomC, "1号楼", "2026-08-18", "morning", 3)},
	}
	configurations := exhaustiveConfigurations(itemIDs, 13) // 三个项目都没有准备限制。

	plans := searchBestPlans(itemIDs, options, nil, configurations)
	if len(plans) != 1 {
		t.Fatalf("permutations of the same booking assignment are one plan, got %d", len(plans))
	}
}

func TestSearchMatchesRandomizedExhaustiveOracle(t *testing.T) {
	random := rand.New(rand.NewSource(20260818))
	itemIDs := []string{
		"10000000-0000-4000-8000-000000000011",
		"10000000-0000-4000-8000-000000000012",
		"10000000-0000-4000-8000-000000000013",
		"10000000-0000-4000-8000-000000000014",
	}
	possibleEdges := []precedence.Rule{}
	for predecessor := 0; predecessor < len(itemIDs); predecessor++ {
		for successor := predecessor + 1; successor < len(itemIDs); successor++ {
			possibleEdges = append(possibleEdges, precedence.Rule{
				PredecessorItemID: itemIDs[predecessor], SuccessorItemID: itemIDs[successor],
			})
		}
	}

	for testCase := 0; testCase < 300; testCase++ {
		configurations := map[string]projectconfiguration.Configuration{}
		options := map[string][]Option{}
		for _, itemID := range itemIDs {
			configurations[itemID] = exhaustiveConfigurations([]string{itemID}, random.Intn(3))[itemID]
			selectedOptions := map[int]bool{}
			optionCount := 1 + random.Intn(3)
			for len(selectedOptions) < optionCount {
				selectedOptions[random.Intn(12)] = true
			}
			for optionCode := range selectedOptions {
				roomIndex := optionCode % 3
				dateIndex := optionCode / 6
				sessionIndex := (optionCode / 3) % 2
				roomID := fmt.Sprintf("29000000-0000-4000-8000-%012d", roomIndex+1)
				date := []string{"2026-08-18", "2026-08-19"}[dateIndex]
				session := []string{"morning", "afternoon"}[sessionIndex]
				option := searchOption(itemID, roomID, fmt.Sprintf("%d号楼", roomIndex+1), date, session, int64(roomIndex%2+1))
				option.CampusID = fmt.Sprintf("30000000-0000-4000-8000-%012d", roomIndex/2+1)
				options[itemID] = append(options[itemID], option)
			}
		}
		rules := make([]precedence.Rule, 0, len(possibleEdges))
		for _, edge := range possibleEdges {
			if random.Intn(2) == 1 {
				rules = append(rules, edge)
			}
		}

		want, feasible := exhaustiveOracle(itemIDs, options, rules, configurations)
		got := searchBestPlans(itemIDs, options, rules, configurations)
		if !feasible {
			if len(got) != 0 {
				t.Fatalf("case %d: oracle found no plan, search returned %#v", testCase, got[0])
			}
			continue
		}
		if len(got) == 0 {
			t.Fatalf("case %d: search missed a feasible plan", testCase)
		}
		if actual := oracleScoreForChoices(got[0].choices, configurations); actual != want {
			t.Fatalf("case %d: score=%v, internal=%+v, exhaustive optimum=%v, choices=%#v", testCase, actual, got[0].score, want, got[0].choices)
		}
		for candidateIndex, candidate := range got {
			if !independentlyExecutable(candidate.choices, configurations) || !respectsPrecedence(candidate.choices, rules) {
				t.Fatalf("case %d candidate %d is not independently executable: %#v", testCase, candidateIndex, candidate)
			}
		}
	}
}

type independentScore struct {
	preparationInversions, preparationTransitions, preparationTimingDeviation int
	serviceDays, campusChanges, buildingChanges, travelMinutes                int
	completionSlot                                                            string
}

func exhaustiveConfigurations(itemIDs []string, rankCode int) map[string]projectconfiguration.Configuration {
	result := make(map[string]projectconfiguration.Configuration, len(itemIDs))
	for _, itemID := range itemIDs {
		rank := rankCode % 3
		rankCode /= 3
		var rules []descriptionrules.Rule
		switch rank {
		case 0:
			rules = []descriptionrules.Rule{{RuleType: descriptionrules.RuleTypeFasting}}
		case 2:
			rules = []descriptionrules.Rule{{RuleType: descriptionrules.RuleTypeDrinkWater}}
		}
		result[itemID] = projectconfiguration.Configuration{ItemID: itemID, PreparationRules: rules}
	}
	return result
}

func exhaustiveOracle(itemIDs []string, options map[string][]Option, rules []precedence.Rule, configurations map[string]projectconfiguration.Configuration) (independentScore, bool) {
	best, found := independentScore{}, false
	permutations(itemIDs, func(order []string) {
		positions := make(map[string]int, len(order))
		for index, itemID := range order {
			positions[itemID] = index
		}
		for _, rule := range rules {
			if positions[rule.PredecessorItemID] >= positions[rule.SuccessorItemID] {
				return
			}
		}
		choices := make([]searchChoice, len(order))
		var selectOptions func(int)
		selectOptions = func(index int) {
			if index == len(order) {
				if !independentlyExecutable(choices, configurations) {
					return
				}
				score := oracleScoreForChoices(choices, configurations)
				if !found || compareIndependentScore(score, best) < 0 {
					best, found = score, true
				}
				return
			}
			for _, option := range options[order[index]] {
				choices[index] = searchChoice{itemID: order[index], option: option}
				selectOptions(index + 1)
			}
		}
		selectOptions(0)
	})
	return best, found
}

func independentlyExecutable(choices []searchChoice, configurations map[string]projectconfiguration.Configuration) bool {
	_, feasible := simulateChoices(choices, configurations)
	return feasible
}

// simulateChoices 是独立于正式搜索器的顺序执行判定器，用于反复核对 DFS 的可行性和最优性。
func simulateChoices(choices []searchChoice, configurations map[string]projectconfiguration.Configuration) (independentScore, bool) {
	result := independentScore{}
	reserved, capacity := map[string]int64{}, map[string]int64{}
	lastDate, lastRoom, lastCampus, lastBuilding := "", "", "", ""
	cursor, lastNoWaterEnd := int32(0), int32(0)
	lastRank := -1
	dayRankCounts := [3]int{}
	drankToday := false
	for _, choice := range choices {
		option := choice.option
		if lastDate != "" && option.ServiceDate < lastDate {
			return independentScore{}, false
		}
		sameDate := option.ServiceDate == lastDate
		if !sameDate {
			result.serviceDays++
			cursor, lastNoWaterEnd, lastRank = 0, 0, -1
			dayRankCounts = [3]int{}
			drankToday = false
		}
		rank := descriptionRuleRank(configurations[choice.itemID])
		if drankToday && rank == 0 {
			return independentScore{}, false
		}
		start, endWindow := independentWindow(option)
		travel := int32(0)
		if sameDate && lastRoom != "" && lastRoom != option.RoomID {
			if lastBuilding == option.Building {
				travel = 5
			} else {
				travel = 15
			}
		}
		if sameDate && cursor+travel > start {
			start = cursor + travel
		}
		if rule, ok := independentRule(configurations[choice.itemID], descriptionrules.RuleTypeDrinkWater); ok && lastNoWaterEnd > 0 {
			if ready := lastNoWaterEnd + rule.MinAdvanceMinutes; ready > start {
				start = ready
			}
			result.preparationTimingDeviation += absoluteInt(int(start - lastNoWaterEnd - rule.RecommendedAdvanceMinutes))
		}
		duration := option.EstimatedDurationMinutes
		if duration <= 0 {
			duration = 5
		}
		end := start + duration
		if end > endWindow {
			return independentScore{}, false
		}
		if sameDate {
			for previousRank := rank + 1; previousRank < 3; previousRank++ {
				result.preparationInversions += dayRankCounts[previousRank]
			}
			if lastRank >= 0 && lastRank != rank {
				result.preparationTransitions++
			}
			if lastCampus != "" && lastCampus != option.CampusID {
				result.campusChanges++
			}
			if lastBuilding != "" && lastBuilding != option.Building {
				result.buildingChanges++
			}
		}
		dayRankCounts[rank]++
		if rank == 0 {
			lastNoWaterEnd = end
		}
		if rank == 2 {
			drankToday = true
		}
		result.travelMinutes += int(travel)
		result.completionSlot = option.ServiceDate + ":" + independentClock(end)
		key := optionCapacityKey(option)
		reserved[key]++
		if current, ok := capacity[key]; !ok || option.RemainingCapacity < current {
			capacity[key] = option.RemainingCapacity
		}
		if reserved[key] > capacity[key] {
			return independentScore{}, false
		}
		cursor, lastDate, lastRoom, lastCampus, lastBuilding, lastRank = end, option.ServiceDate, option.RoomID, option.CampusID, option.Building, rank
	}
	return result, true
}

func independentWindow(option Option) (int32, int32) {
	defaultStart, defaultEnd := int32(9*60), int32(12*60)
	if option.Session == "afternoon" {
		defaultStart, defaultEnd = 13*60, 18*60
	}
	start, end := int32(0), int32(24*60)
	hasStart, hasEnd := false, false
	for _, value := range []string{option.RoomOpenTime, option.ItemStartTime} {
		if parsed, ok := independentParseClock(value); ok {
			if !hasStart || parsed > start {
				start = parsed
			}
			hasStart = true
		}
	}
	for _, value := range []string{option.RoomCloseTime, option.ItemEndTime} {
		if parsed, ok := independentParseClock(value); ok {
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
	return start, end
}

func independentParseClock(value string) (int32, bool) {
	var hour, minute int
	if _, err := fmt.Sscanf(value, "%d:%d", &hour, &minute); err != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, false
	}
	return int32(hour*60 + minute), true
}

func independentClock(value int32) string {
	return fmt.Sprintf("%02d:%02d", value/60, value%60)
}

func independentRule(configuration projectconfiguration.Configuration, ruleType descriptionrules.RuleType) (descriptionrules.Rule, bool) {
	for _, rule := range configuration.PreparationRules {
		if rule.RuleType == ruleType {
			return rule, true
		}
	}
	return descriptionrules.Rule{}, false
}

func oracleScoreForChoices(choices []searchChoice, configurations map[string]projectconfiguration.Configuration) independentScore {
	result, _ := simulateChoices(choices, configurations)
	return result
}

func compareIndependentScore(left, right independentScore) int {
	leftValues := []int{left.preparationInversions, left.preparationTransitions, left.preparationTimingDeviation, left.serviceDays, left.campusChanges, left.buildingChanges, left.travelMinutes}
	rightValues := []int{right.preparationInversions, right.preparationTransitions, right.preparationTimingDeviation, right.serviceDays, right.campusChanges, right.buildingChanges, right.travelMinutes}
	for index := range leftValues {
		if leftValues[index] < rightValues[index] {
			return -1
		}
		if leftValues[index] > rightValues[index] {
			return 1
		}
	}
	if left.completionSlot < right.completionSlot {
		return -1
	}
	if left.completionSlot > right.completionSlot {
		return 1
	}
	return 0
}

func respectsPrecedence(choices []searchChoice, rules []precedence.Rule) bool {
	positions := make(map[string]int, len(choices))
	for index, choice := range choices {
		positions[choice.itemID] = index
	}
	for _, rule := range rules {
		predecessor, hasPredecessor := positions[rule.PredecessorItemID]
		successor, hasSuccessor := positions[rule.SuccessorItemID]
		if hasPredecessor && hasSuccessor && predecessor >= successor {
			return false
		}
	}
	return true
}

func permutations(values []string, visit func([]string)) {
	working := append([]string(nil), values...)
	var generate func(int)
	generate = func(index int) {
		if index == len(working) {
			visit(append([]string(nil), working...))
			return
		}
		for candidate := index; candidate < len(working); candidate++ {
			working[index], working[candidate] = working[candidate], working[index]
			generate(index + 1)
			working[index], working[candidate] = working[candidate], working[index]
		}
	}
	generate(0)
}

func BenchmarkSearchBestPlansTenItems(b *testing.B) {
	benchmarkSearchBestPlans(b, 4)
}

func BenchmarkSearchBestPlansTenItemsWideOptions(b *testing.B) {
	benchmarkSearchBestPlans(b, 12)
}

func benchmarkSearchBestPlans(b *testing.B, optionCount int) {
	itemIDs := make([]string, 10)
	options := map[string][]Option{}
	configurations := map[string]projectconfiguration.Configuration{}
	for index := range itemIDs {
		itemID := fmt.Sprintf("10000000-0000-4000-8000-%012d", index+1)
		itemIDs[index] = itemID
		configurations[itemID] = projectconfiguration.Configuration{ItemID: itemID}
		for optionIndex := 0; optionIndex < optionCount; optionIndex++ {
			date := fmt.Sprintf("2026-08-%02d", 18+optionIndex/4)
			building := fmt.Sprintf("%d号楼", optionIndex%2+1)
			roomID := fmt.Sprintf("20000000-0000-4000-%04d-%012d", index+1, optionIndex+1)
			session := []string{"morning", "afternoon"}[(optionIndex/2)%2]
			options[itemID] = append(options[itemID], searchOption(itemID, roomID, building, date, session, 10))
		}
	}
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		if plans := searchBestPlans(itemIDs, options, nil, configurations); len(plans) == 0 {
			b.Fatal("expected plans")
		}
	}
}

func newSearchTestManager(itemIDs []string, preparation map[string][]descriptionrules.Rule, options map[string][]Option, rules []precedence.Rule) *Manager {
	configurations := make(map[string]projectconfiguration.Configuration, len(itemIDs))
	projects := make(map[string]projectconfiguration.Project, len(itemIDs))
	for _, itemID := range itemIDs {
		configurations[itemID] = projectconfiguration.Configuration{ItemID: itemID, PreparationRules: preparation[itemID]}
		projects[itemID] = projectconfiguration.Project{ItemID: itemID, Name: "项目" + itemID[len(itemID)-1:], Status: "active"}
	}
	store := &planningStoreStub{configurations: configurations, rules: rules, plans: map[string]Plan{}}
	appointment := &planningAppointmentStub{projects: projects, options: options, bookings: map[string][]Booking{}}
	manager, err := NewManager(store, appointment, nil)
	if err != nil {
		panic(err)
	}
	return manager
}

func searchOption(itemID, roomID, building, date, session string, capacity int64) Option {
	return Option{
		ItemID: itemID, RoomID: roomID, RoomDisplayName: building + "检查室", CampusID: "30000000-0000-4000-8000-000000000001",
		Building: building, FloorNumber: 1, RoomNumber: roomID[len(roomID)-1:], ServiceDate: date, Session: session,
		RemainingCapacity: capacity, EstimatedDurationMinutes: 15,
	}
}

func searchCommand(itemIDs ...string) GenerateCommand {
	return GenerateCommand{
		ItemIDs:               itemIDs,
		CandidateAvailability: []CandidateAvailability{{ServiceDate: "2026-08-18", Sessions: []string{"morning", "afternoon"}}},
	}
}
