package projectconfiguration

import (
	"context"
	"errors"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	descriptionrules "hospital/service/guidance/rpc/internal/rules/description"
)

// PatientReminders 返回患者执行检查前需要主动确认的普通提醒。
// 该读模型只暴露第三步“患者提醒”，不混入先后关系的患者提示或准备状态提示。
func (m *Manager) PatientReminders(ctx context.Context, itemID string) ([]descriptionrules.Reminder, error) {
	itemID, err := normalizedUUID(itemID)
	if err != nil {
		return nil, err
	}
	if _, err = m.projects.ResolveProject(ctx, itemID); err != nil {
		return nil, err
	}
	configuration, err := m.store.GetConfiguration(ctx, itemID)
	if errors.Is(err, ErrNotFound) {
		return []descriptionrules.Reminder{}, nil
	}
	if err != nil {
		return nil, err
	}
	return append([]descriptionrules.Reminder(nil), configuration.Reminders...), nil
}

// Preview 将医生填写的检查说明转换为待确认的结构化规则和提醒。
func (m *Manager) Preview(ctx context.Context, description string) (descriptionrules.Preview, error) {
	return m.description.Parse(ctx, description)
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
