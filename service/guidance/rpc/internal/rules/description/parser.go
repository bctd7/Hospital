package description

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const maxDescriptionRunes = 4000

func Parse(description string) (Preview, error) {
	description = strings.TrimSpace(description)
	if !utf8.ValidString(description) || utf8.RuneCountInString(description) > maxDescriptionRunes {
		return Preview{}, fmt.Errorf("invalid preparation description")
	}
	preview := Preview{Description: description, Rules: []Rule{}, Reminders: []Reminder{}, UnresolvedFragments: []string{}, ParserMode: "deterministic"}
	if description == "" {
		return preview, nil
	}
	lower := strings.ToLower(description)
	seen := map[RuleType]bool{}
	addPreviousDayRule := func(ruleType RuleType, source string) {
		if seen[ruleType] {
			return
		}
		seen[ruleType] = true
		preview.Rules = append(preview.Rules, Rule{RuleType: ruleType, StartMode: StartModePreviousDayTime, PreviousDayTime: "20:00", Source: source})
	}
	addRangeRule := func(ruleType RuleType, duration durationRange) {
		if seen[ruleType] {
			return
		}
		seen[ruleType] = true
		preview.Rules = append(preview.Rules, Rule{
			RuleType: ruleType, StartMode: StartModeAdvanceRange,
			MinAdvanceMinutes: duration.MinMinutes, RecommendedAdvanceMinutes: duration.MinMinutes,
			MaxAdvanceMinutes: duration.MaxMinutes, Source: "explicit",
		})
	}
	if duration, ok := findDurationRange(lower, "空腹"); ok {
		addRangeRule(RuleTypeFasting, duration)
		addRangeRule(RuleTypeNoWater, duration)
	} else if strings.Contains(lower, "空腹") || strings.Contains(lower, "禁食禁水") || strings.Contains(lower, "禁水禁食") {
		addPreviousDayRule(RuleTypeFasting, "default")
		addPreviousDayRule(RuleTypeNoWater, "default")
	} else {
		if duration, ok := findDurationRange(lower, "禁食"); ok {
			addRangeRule(RuleTypeFasting, duration)
		} else if strings.Contains(lower, "禁食") {
			addPreviousDayRule(RuleTypeFasting, "default")
		}
		if duration, ok := findDurationRange(lower, "禁水"); ok {
			addRangeRule(RuleTypeNoWater, duration)
		} else if strings.Contains(lower, "禁水") {
			addPreviousDayRule(RuleTypeNoWater, "default")
		}
	}
	if strings.Contains(lower, "排空膀胱") {
		addPreviousDayRule(RuleTypeNoWater, "default")
		preview.Reminders = append(preview.Reminders, Reminder{Text: "请在检查前一天 20:00 前按医嘱服用泻药。"})
	}
	if strings.Contains(lower, "憋尿") || strings.Contains(lower, "多喝水") || strings.Contains(lower, "饮水") || strings.Contains(lower, "喝水") {
		if !seen[RuleTypeDrinkWater] {
			seen[RuleTypeDrinkWater] = true
			preview.Rules = append(preview.Rules, Rule{
				RuleType: RuleTypeDrinkWater, StartMode: StartModeAdvanceRange,
				MinAdvanceMinutes: 120, RecommendedAdvanceMinutes: 180, MaxAdvanceMinutes: 240,
				ReadinessHint: "出现明显尿意即可前往检查。",
				Source:        "default",
			})
		}
	}
	if containsAny(lower, "服药", "用药", "药物", "泻药") && !strings.Contains(lower, "排空膀胱") {
		preview.Reminders = append(preview.Reminders, Reminder{Text: "请按检查说明和医嘱提前用药。"})
	}
	if containsAny(lower, "怀孕", "妊娠", "孕妇", "可能怀孕") {
		preview.Reminders = append(preview.Reminders, Reminder{Text: "怀孕或可能怀孕时，请在检查前主动告知工作人员。"})
	}
	if containsAny(lower, "月经", "经期") {
		preview.Reminders = append(preview.Reminders, Reminder{Text: "处于月经期时，请按检查说明提前咨询工作人员。"})
	}
	if len(preview.Rules) == 0 && len(preview.Reminders) == 0 {
		preview.UnresolvedFragments = append(preview.UnresolvedFragments, description)
	}
	return preview, nil
}

func Validate(rules []Rule, reminders []Reminder) error {
	seen := make(map[RuleType]struct{}, len(rules))
	for _, rule := range rules {
		if rule.RuleType != RuleTypeFasting && rule.RuleType != RuleTypeNoWater && rule.RuleType != RuleTypeDrinkWater {
			return fmt.Errorf("unsupported preparation rule type")
		}
		if _, exists := seen[rule.RuleType]; exists {
			return fmt.Errorf("duplicate preparation rule type")
		}
		seen[rule.RuleType] = struct{}{}
		switch rule.StartMode {
		case StartModePreviousDayTime:
			if _, err := time.Parse("15:04", rule.PreviousDayTime); err != nil || rule.RuleType == RuleTypeDrinkWater {
				return fmt.Errorf("invalid previous-day preparation rule")
			}
		case StartModeAdvanceRange:
			if rule.MinAdvanceMinutes < 0 ||
				rule.MinAdvanceMinutes > rule.RecommendedAdvanceMinutes ||
				rule.RecommendedAdvanceMinutes > rule.MaxAdvanceMinutes || rule.MaxAdvanceMinutes > 24*60 {
				return fmt.Errorf("invalid preparation advance range")
			}
		default:
			return fmt.Errorf("unsupported preparation start mode")
		}
		if rule.Source != "" && rule.Source != "explicit" && rule.Source != "default" && rule.Source != "model" && rule.Source != "manual" {
			return fmt.Errorf("unsupported preparation rule source")
		}
		if utf8.RuneCountInString(strings.TrimSpace(rule.ReadinessHint)) > 256 {
			return fmt.Errorf("preparation readiness hint is too long")
		}
	}
	for _, reminder := range reminders {
		text := strings.TrimSpace(reminder.Text)
		if text == "" || !utf8.ValidString(text) || utf8.RuneCountInString(text) > 512 || reminder.AdvanceMinutes < 0 || reminder.AdvanceMinutes > 7*24*60 {
			return fmt.Errorf("invalid patient reminder")
		}
	}
	return nil
}

type durationRange struct {
	MinMinutes int32
	MaxMinutes int32
}

func findDurationRange(value, keyword string) (durationRange, bool) {
	pattern := regexp.MustCompile(regexp.QuoteMeta(keyword) + `[^\d]{0,12}(\d+(?:\.\d+)?)\s*(?:[-~～至到]\s*(\d+(?:\.\d+)?))?\s*(小时|h|分钟|min)`)
	match := pattern.FindStringSubmatch(value)
	if len(match) == 0 {
		return durationRange{}, false
	}
	minimum, err := durationMinutes(match[1], match[3])
	if err != nil {
		return durationRange{}, false
	}
	maximum := minimum
	if match[2] != "" {
		maximum, err = durationMinutes(match[2], match[3])
		if err != nil {
			return durationRange{}, false
		}
	}
	if minimum <= 0 || minimum > maximum || maximum > 24*60 {
		return durationRange{}, false
	}
	return durationRange{MinMinutes: minimum, MaxMinutes: maximum}, true
}

func durationMinutes(value, unit string) (int32, error) {
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	if unit == "小时" || unit == "h" {
		number *= 60
	}
	return int32(number + 0.5), nil
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}
