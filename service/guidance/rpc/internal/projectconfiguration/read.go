package projectconfiguration

import (
	"context"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	descriptionrules "hospital/service/guidance/rpc/internal/rules/description"
)

// Preview 将医生填写的检查说明转换为待确认的结构化规则和提醒。
func (m *Manager) Preview(description string) (descriptionrules.Preview, error) {
	return descriptionrules.Parse(description)
}

// Get 返回一个项目的完整配置视图：Appointment 项目事实、说明规则和先后关系。
func (m *Manager) Get(ctx context.Context, operator authn.Principal, itemID string) (Result, error) {
	if err := requirePermission(operator, contractauthz.PermissionRuleRead); err != nil {
		return Result{}, err
	}
	itemID, err := normalizedUUID(itemID)
	if err != nil {
		return Result{}, err
	}
	project, err := m.projects.ResolveProject(ctx, itemID)
	if err != nil {
		return Result{}, err
	}
	if err := requireScope(operator, project.OwnerDepartmentID); err != nil {
		return Result{}, err
	}
	configuration, err := m.store.GetConfiguration(ctx, itemID)
	if err != nil {
		return Result{}, err
	}
	rules, err := m.store.ListRulesByOwner(ctx, itemID)
	if err != nil {
		return Result{}, err
	}
	return Result{Project: project, Configuration: configuration, Rules: rules}, nil
}
