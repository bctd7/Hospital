package preparation

import "testing"

func TestParseClosedPreparationStatesAndReminders(t *testing.T) {
	preview, err := Parse("检查前空腹；排空膀胱；如可能怀孕请提前告知工作人员")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	want := map[RuleType]bool{RuleTypeFasting: true, RuleTypeNoWater: true}
	for _, rule := range preview.Rules {
		delete(want, rule.RuleType)
	}
	if len(want) != 0 {
		t.Fatalf("missing preparation states: %v; got %#v", want, preview.Rules)
	}
	if len(preview.Reminders) != 2 {
		t.Fatalf("reminders = %#v, want laxative and pregnancy reminders", preview.Reminders)
	}
}

func TestParseDrinkWaterUsesFlexibleRange(t *testing.T) {
	preview, err := Parse("检查前憋尿，提前三小时多喝水，有明显尿意即可检查")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(preview.Rules) != 1 {
		t.Fatalf("rules = %#v, want one drink-water rule", preview.Rules)
	}
	rule := preview.Rules[0]
	if rule.RuleType != RuleTypeDrinkWater || rule.MinAdvanceMinutes != 120 || rule.RecommendedAdvanceMinutes != 180 || rule.MaxAdvanceMinutes != 240 {
		t.Fatalf("unexpected drink-water rule: %#v", rule)
	}
}

func TestParseUnknownTextRemainsVisible(t *testing.T) {
	preview, err := Parse("携带既往纸质材料")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(preview.Rules) != 0 || len(preview.Reminders) != 0 || len(preview.UnresolvedFragments) != 1 {
		t.Fatalf("unexpected preview: %#v", preview)
	}
}
