// Package preparation 定义项目准备状态的封闭业务模型和确定性解析。
package preparation

type RuleType string

const (
	RuleTypeFasting    RuleType = "fasting"
	RuleTypeNoWater    RuleType = "no_water"
	RuleTypeDrinkWater RuleType = "drink_water"
)

type StartMode string

const (
	StartModeAdvanceRange    StartMode = "advance_range"
	StartModePreviousDayTime StartMode = "previous_day_time"
)

type Rule struct {
	RuleType                  RuleType  `json:"rule_type"`
	StartMode                 StartMode `json:"start_mode"`
	MinAdvanceMinutes         int32     `json:"min_advance_minutes,omitempty"`
	RecommendedAdvanceMinutes int32     `json:"recommended_advance_minutes,omitempty"`
	MaxAdvanceMinutes         int32     `json:"max_advance_minutes,omitempty"`
	PreviousDayTime           string    `json:"previous_day_time,omitempty"`
	ReadinessHint             string    `json:"readiness_hint,omitempty"`
}

type Reminder struct {
	Text           string `json:"text"`
	AdvanceMinutes int32  `json:"advance_minutes,omitempty"`
}

type Preview struct {
	Description         string
	Rules               []Rule
	Reminders           []Reminder
	UnresolvedFragments []string
}
