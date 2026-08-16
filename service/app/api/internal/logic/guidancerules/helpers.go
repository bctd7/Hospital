package guidancerules

import (
	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/app/api/internal/types"
)

func precedenceRuleResponse(rule *guidancev1.PrecedenceRule) types.PrecedenceRuleResponse {
	if rule == nil {
		return types.PrecedenceRuleResponse{}
	}
	return types.PrecedenceRuleResponse{
		RuleID:                  rule.RuleId,
		PredecessorItemID:       rule.PredecessorItemId,
		PredecessorDepartmentID: rule.PredecessorDepartmentId,
		PredecessorItemName:     rule.PredecessorItemName,
		SuccessorItemID:         rule.SuccessorItemId,
		SuccessorDepartmentID:   rule.SuccessorDepartmentId,
		SuccessorItemName:       rule.SuccessorItemName,
		StaffReason:             rule.StaffReason,
		PatientMessage:          rule.PatientMessage,
		Direct:                  rule.Direct,
		PathLength:              rule.PathLength,
		Version:                 rule.Version,
		CreatedAt:               rule.CreatedAt,
		UpdatedAt:               rule.UpdatedAt,
	}
}
