package description

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

func TestParseExplicitFastingRangeOverridesHospitalDefault(t *testing.T) {
	preview, err := Parse("检查前需空腹 8～12 小时")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(preview.Rules) != 2 {
		t.Fatalf("rules = %#v, want fasting and no-water", preview.Rules)
	}
	for _, rule := range preview.Rules {
		if rule.StartMode != StartModeAdvanceRange || rule.MinAdvanceMinutes != 480 || rule.MaxAdvanceMinutes != 720 || rule.Source != "explicit" {
			t.Fatalf("explicit duration must remain relative to the expected examination time: %#v", rule)
		}
	}
}

func TestParseVagueFastingUsesHospitalDefault(t *testing.T) {
	preview, err := Parse("检查前请空腹")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	for _, rule := range preview.Rules {
		if rule.StartMode != StartModePreviousDayTime || rule.PreviousDayTime != "20:00" || rule.Source != "default" {
			t.Fatalf("vague fasting should use the configured hospital default: %#v", rule)
		}
	}
}

func TestValidateAllowsManuallyAdjustedPreviousDayTime(t *testing.T) {
	rules := []Rule{{RuleType: RuleTypeFasting, StartMode: StartModePreviousDayTime, PreviousDayTime: "19:30", Source: "manual"}}
	if err := Validate(rules, nil); err != nil {
		t.Fatalf("a valid manually selected previous-day time must be accepted: %v", err)
	}
	rules[0].PreviousDayTime = "25:00"
	if err := Validate(rules, nil); err == nil {
		t.Fatal("invalid clock time must be rejected")
	}
}
