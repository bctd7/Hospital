package logic

import (
	"errors"

	"hospital/common/authn"
	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/projectconfiguration"
	descriptionrules "hospital/service/guidance/rpc/internal/rules/description"
	"hospital/service/guidance/rpc/internal/rules/precedence"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func projectConfigurationRPCError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, projectconfiguration.ErrInvalid):
		return status.Error(codes.InvalidArgument, "invalid examination item configuration")
	case errors.Is(err, projectconfiguration.ErrForbidden):
		return status.Error(codes.PermissionDenied, "permission denied")
	case errors.Is(err, projectconfiguration.ErrNotFound):
		return status.Error(codes.NotFound, "examination item configuration not found")
	case errors.Is(err, projectconfiguration.ErrConflict):
		return status.Error(codes.AlreadyExists, "examination item configuration conflicts with existing data")
	case errors.Is(err, projectconfiguration.ErrCycle):
		return status.Error(codes.FailedPrecondition, "precedence rules contain a circular dependency")
	case errors.Is(err, projectconfiguration.ErrVersionConflict):
		return status.Error(codes.Aborted, "examination item configuration version conflict")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

func configurationCommand(in *guidancev1.ConfigureExaminationItemRequest) projectconfiguration.Command {
	return projectconfiguration.Command{
		Action: in.GetAction(), ItemID: in.GetItemId(), OwnerDepartmentID: in.GetOwnerDepartmentId(),
		ItemName: in.GetItemName(), EstimatedDurationMinutes: in.GetEstimatedDurationMinutes(),
		ExpectedItemVersion: in.GetExpectedItemVersion(), Description: in.GetDescription(),
		PrecedenceRules:  precedenceInputs(in.GetPrecedenceRules()),
		PreparationRules: preparationInputs(in.GetPreparationRules()), Reminders: reminderInputs(in.GetReminders()),
		ExpectedConfigurationVersion: in.GetExpectedConfigurationVersion(), OperationID: in.GetOperationId(), RequestID: in.GetRequestId(),
	}
}

func precedenceInputs(values []*guidancev1.ConfiguredPrecedenceRule) []precedence.Rule {
	result := make([]precedence.Rule, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		result = append(result, precedence.Rule{
			PredecessorItemID: value.GetPredecessorItemId(), SuccessorItemID: value.GetSuccessorItemId(),
			StaffReason: value.GetStaffReason(), PatientMessage: value.GetPatientMessage(),
		})
	}
	return result
}

func preparationInputs(values []*guidancev1.PreparationRule) []descriptionrules.Rule {
	result := make([]descriptionrules.Rule, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		result = append(result, descriptionrules.Rule{
			RuleType: descriptionrules.RuleType(value.GetRuleType()), StartMode: descriptionrules.StartMode(value.GetStartMode()),
			MinAdvanceMinutes: value.GetMinAdvanceMinutes(), RecommendedAdvanceMinutes: value.GetRecommendedAdvanceMinutes(),
			MaxAdvanceMinutes: value.GetMaxAdvanceMinutes(), PreviousDayTime: value.GetPreviousDayTime(), ReadinessHint: value.GetReadinessHint(),
		})
	}
	return result
}

func reminderInputs(values []*guidancev1.PatientReminder) []descriptionrules.Reminder {
	result := make([]descriptionrules.Reminder, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, descriptionrules.Reminder{Text: value.GetText(), AdvanceMinutes: value.GetAdvanceMinutes()})
		}
	}
	return result
}

func configurationResponse(result projectconfiguration.Result) *guidancev1.ExaminationItemConfiguration {
	return &guidancev1.ExaminationItemConfiguration{
		ItemId: result.Project.ItemID, OwnerDepartmentId: result.Project.OwnerDepartmentID, ItemName: result.Project.Name,
		EstimatedDurationMinutes: result.Project.EstimatedDurationMinutes, Status: result.Project.Status, ItemVersion: result.Project.Version,
		Description: result.Configuration.Description, PrecedenceRules: configuredPrecedenceResponses(result.Rules),
		PreparationRules: preparationResponses(result.Configuration.PreparationRules), Reminders: reminderResponses(result.Configuration.Reminders),
		ConfigurationVersion: result.Configuration.Version, UpdatedAt: formatGuidanceTime(result.Configuration.UpdatedAt),
	}
}

func configuredPrecedenceResponses(values []precedence.Rule) []*guidancev1.ConfiguredPrecedenceRule {
	result := make([]*guidancev1.ConfiguredPrecedenceRule, 0, len(values))
	for _, value := range values {
		result = append(result, &guidancev1.ConfiguredPrecedenceRule{
			PredecessorItemId: value.PredecessorItemID, SuccessorItemId: value.SuccessorItemID,
			StaffReason: value.StaffReason, PatientMessage: value.PatientMessage,
		})
	}
	return result
}

func preparationResponses(values []descriptionrules.Rule) []*guidancev1.PreparationRule {
	result := make([]*guidancev1.PreparationRule, 0, len(values))
	for _, value := range values {
		result = append(result, &guidancev1.PreparationRule{
			RuleType: string(value.RuleType), StartMode: string(value.StartMode), MinAdvanceMinutes: value.MinAdvanceMinutes,
			RecommendedAdvanceMinutes: value.RecommendedAdvanceMinutes, MaxAdvanceMinutes: value.MaxAdvanceMinutes,
			PreviousDayTime: value.PreviousDayTime, ReadinessHint: value.ReadinessHint,
		})
	}
	return result
}

func reminderResponses(values []descriptionrules.Reminder) []*guidancev1.PatientReminder {
	result := make([]*guidancev1.PatientReminder, 0, len(values))
	for _, value := range values {
		result = append(result, &guidancev1.PatientReminder{Text: value.Text, AdvanceMinutes: value.AdvanceMinutes})
	}
	return result
}

func requireGuidanceStaff(principal authn.Principal) error {
	if principal.AccountType != authn.AccountTypeStaff {
		return status.Error(codes.PermissionDenied, "staff account required")
	}
	return nil
}
