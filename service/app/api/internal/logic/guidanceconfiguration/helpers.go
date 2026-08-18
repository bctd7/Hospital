package guidanceconfiguration

import (
	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/app/api/internal/types"
)

func preparationRuleRequests(values []types.GuidancePreparationRule) []*guidancev1.PreparationRule {
	result := make([]*guidancev1.PreparationRule, 0, len(values))
	for _, value := range values {
		result = append(result, &guidancev1.PreparationRule{
			RuleType: value.RuleType, StartMode: value.StartMode, MinAdvanceMinutes: value.MinAdvanceMinutes,
			RecommendedAdvanceMinutes: value.RecommendedAdvanceMinutes, MaxAdvanceMinutes: value.MaxAdvanceMinutes,
			PreviousDayTime: value.PreviousDayTime, ReadinessHint: value.ReadinessHint, Source: value.Source,
		})
	}
	return result
}

func reminderRequests(values []types.GuidancePatientReminder) []*guidancev1.PatientReminder {
	result := make([]*guidancev1.PatientReminder, 0, len(values))
	for _, value := range values {
		result = append(result, &guidancev1.PatientReminder{Text: value.Text, AdvanceMinutes: value.AdvanceMinutes})
	}
	return result
}

func precedenceRequests(values []types.ConfiguredPrecedenceRuleInput) []*guidancev1.ConfiguredPrecedenceRule {
	result := make([]*guidancev1.ConfiguredPrecedenceRule, 0, len(values))
	for _, value := range values {
		result = append(result, &guidancev1.ConfiguredPrecedenceRule{
			PredecessorItemId: value.PredecessorItemID, SuccessorItemId: value.SuccessorItemID,
			StaffReason: value.StaffReason, PatientMessage: value.PatientMessage,
		})
	}
	return result
}

func configurationResponse(value *guidancev1.ExaminationItemConfiguration) *types.ExaminationItemConfigurationResponse {
	return &types.ExaminationItemConfigurationResponse{
		ItemID: value.GetItemId(), OwnerDepartmentID: value.GetOwnerDepartmentId(), ItemName: value.GetItemName(),
		EstimatedDurationMinutes: value.GetEstimatedDurationMinutes(), Status: value.GetStatus(), ItemVersion: value.GetItemVersion(),
		Description: value.GetDescription(), PrecedenceRules: precedenceResponses(value.GetPrecedenceRules()),
		PreparationRules: preparationRuleResponses(value.GetPreparationRules()), Reminders: reminderResponses(value.GetReminders()),
		ConfigurationVersion: value.GetConfigurationVersion(), UpdatedAt: value.GetUpdatedAt(),
	}
}

func precedenceResponses(values []*guidancev1.ConfiguredPrecedenceRule) []types.ConfiguredPrecedenceRuleInput {
	result := make([]types.ConfiguredPrecedenceRuleInput, 0, len(values))
	for _, value := range values {
		result = append(result, types.ConfiguredPrecedenceRuleInput{
			PredecessorItemID: value.GetPredecessorItemId(), SuccessorItemID: value.GetSuccessorItemId(),
			StaffReason: value.GetStaffReason(), PatientMessage: value.GetPatientMessage(),
		})
	}
	return result
}

func preparationRuleResponses(values []*guidancev1.PreparationRule) []types.GuidancePreparationRule {
	result := make([]types.GuidancePreparationRule, 0, len(values))
	for _, value := range values {
		result = append(result, types.GuidancePreparationRule{
			RuleType: value.GetRuleType(), StartMode: value.GetStartMode(), MinAdvanceMinutes: value.GetMinAdvanceMinutes(),
			RecommendedAdvanceMinutes: value.GetRecommendedAdvanceMinutes(), MaxAdvanceMinutes: value.GetMaxAdvanceMinutes(),
			PreviousDayTime: value.GetPreviousDayTime(), ReadinessHint: value.GetReadinessHint(), Source: value.GetSource(),
		})
	}
	return result
}

func reminderResponses(values []*guidancev1.PatientReminder) []types.GuidancePatientReminder {
	result := make([]types.GuidancePatientReminder, 0, len(values))
	for _, value := range values {
		result = append(result, types.GuidancePatientReminder{Text: value.GetText(), AdvanceMinutes: value.GetAdvanceMinutes()})
	}
	return result
}
