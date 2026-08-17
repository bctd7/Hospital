package projectconfiguration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"hospital/common/authn"
	commonauthz "hospital/common/authz"
	descriptionrules "hospital/service/guidance/rpc/internal/rules/description"
	"hospital/service/guidance/rpc/internal/rules/precedence"

	"github.com/google/uuid"
)

// normalizeAndResolve 完成配置写入前的全部边界校验，并把先后规则中的项目快照补齐。
func (m *Manager) normalizeAndResolve(ctx context.Context, operator authn.Principal, command Command) (Command, error) {
	command.Action = strings.ToLower(strings.TrimSpace(command.Action))
	if command.Action != ActionCreate && command.Action != ActionUpdate {
		return Command{}, ErrInvalid
	}
	operationID, err := normalizedUUID(command.OperationID)
	if err != nil {
		return Command{}, err
	}
	command.OperationID = operationID
	if strings.TrimSpace(command.RequestID) != "" {
		if command.RequestID, err = normalizedUUID(command.RequestID); err != nil {
			return Command{}, err
		}
	}
	if command.Action == ActionCreate && strings.TrimSpace(command.ItemID) == "" {
		// 同一个 operation_id 必须在重试时得到同一个项目 ID。
		command.ItemID = uuid.NewSHA1(uuid.NameSpaceOID, []byte("guidance-item:"+command.OperationID)).String()
	} else if command.ItemID, err = normalizedUUID(command.ItemID); err != nil {
		return Command{}, err
	}
	if command.OwnerDepartmentID, err = normalizedUUID(command.OwnerDepartmentID); err != nil {
		return Command{}, err
	}
	command.ItemName, err = normalizedText(command.ItemName, 128, true)
	if err != nil || command.EstimatedDurationMinutes < 5 || command.EstimatedDurationMinutes > 480 || command.EstimatedDurationMinutes%5 != 0 {
		return Command{}, ErrInvalid
	}
	command.Description, err = normalizedText(command.Description, 4000, false)
	if err != nil || descriptionrules.Validate(command.PreparationRules, command.Reminders) != nil {
		return Command{}, ErrInvalid
	}
	if command.Action == ActionCreate {
		if command.ExpectedItemVersion != 0 || command.ExpectedConfigurationVersion != 0 {
			return Command{}, ErrInvalid
		}
	} else {
		project, resolveErr := m.projects.ResolveProject(ctx, command.ItemID)
		if resolveErr != nil {
			return Command{}, resolveErr
		}
		if project.OwnerDepartmentID != command.OwnerDepartmentID || project.Version != command.ExpectedItemVersion {
			return Command{}, ErrVersionConflict
		}
	}
	if err := requireScope(operator, command.OwnerDepartmentID); err != nil {
		return Command{}, err
	}
	command.PrecedenceRules, err = m.resolveRules(ctx, operator, command)
	if err != nil {
		return Command{}, err
	}
	return command, nil
}

func (m *Manager) resolveRules(ctx context.Context, operator authn.Principal, command Command) ([]precedence.Rule, error) {
	resolved := make([]precedence.Rule, 0, len(command.PrecedenceRules))
	seen := make(map[string]struct{}, len(command.PrecedenceRules))
	current := Project{ItemID: command.ItemID, OwnerDepartmentID: command.OwnerDepartmentID, Name: command.ItemName}
	for _, input := range command.PrecedenceRules {
		predecessorID, err := normalizedUUID(input.PredecessorItemID)
		if err != nil {
			return nil, err
		}
		successorID, err := normalizedUUID(input.SuccessorItemID)
		if err != nil || predecessorID == successorID || (predecessorID != command.ItemID && successorID != command.ItemID) {
			return nil, ErrInvalid
		}
		key := predecessorID + ">" + successorID
		if _, exists := seen[key]; exists {
			return nil, ErrConflict
		}
		seen[key] = struct{}{}
		predecessor, err := m.resolveRuleProject(ctx, current, predecessorID)
		if err != nil {
			return nil, err
		}
		successor, err := m.resolveRuleProject(ctx, current, successorID)
		if err != nil {
			return nil, err
		}
		staffReason, err := normalizedText(input.StaffReason, 512, true)
		if err != nil {
			return nil, err
		}
		patientMessage, err := normalizedText(input.PatientMessage, 512, false)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, precedence.Rule{
			RuleID: uuid.NewString(), OwnerItemID: command.ItemID,
			PredecessorItemID: predecessor.ItemID, PredecessorDepartmentID: predecessor.OwnerDepartmentID, PredecessorItemName: predecessor.Name,
			SuccessorItemID: successor.ItemID, SuccessorDepartmentID: successor.OwnerDepartmentID, SuccessorItemName: successor.Name,
			StaffReason: staffReason, PatientMessage: patientMessage, CreatedBy: operator.AccountID,
			CreateOperationID: uuid.NewString(), Version: 1,
		})
	}
	return resolved, nil
}

func (m *Manager) resolveRuleProject(ctx context.Context, current Project, itemID string) (Project, error) {
	if itemID == current.ItemID {
		return current, nil
	}
	return m.projects.ResolveProject(ctx, itemID)
}

func requirePermission(operator authn.Principal, permission string) error {
	if err := commonauthz.RequirePermission(operator, permission); err != nil {
		return ErrForbidden
	}
	if _, err := uuid.Parse(strings.TrimSpace(operator.AccountID)); err != nil {
		return ErrForbidden
	}
	return nil
}

func requireScope(operator authn.Principal, departmentID string) error {
	if operator.HasRole(authn.RoleSuperAdmin) {
		return nil
	}
	if !operator.HasRole(authn.RoleDepartmentDoctor) || strings.TrimSpace(operator.DepartmentID) != departmentID {
		return ErrForbidden
	}
	return nil
}

func normalizedUUID(value string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", ErrInvalid
	}
	return parsed.String(), nil
}

func normalizedText(value string, maxRunes int, required bool) (string, error) {
	value = strings.TrimSpace(value)
	if (required && value == "") || !utf8.ValidString(value) || utf8.RuneCountInString(value) > maxRunes {
		return "", ErrInvalid
	}
	return value, nil
}

// commandFingerprint 生成与 request_id 无关的业务指纹，用于保证 operation_id 幂等。
func commandFingerprint(command Command) (string, error) {
	copy := command
	copy.RequestID = ""
	copy.OperationID = ""
	copy.PrecedenceRules = append([]precedence.Rule(nil), command.PrecedenceRules...)
	for index := range copy.PrecedenceRules {
		copy.PrecedenceRules[index].RuleID = ""
		copy.PrecedenceRules[index].CreateOperationID = ""
		copy.PrecedenceRules[index].CreatedBy = ""
		copy.PrecedenceRules[index].Version = 0
		copy.PrecedenceRules[index].CreatedAt = time.Time{}
		copy.PrecedenceRules[index].UpdatedAt = time.Time{}
	}
	encoded, err := json.Marshal(copy)
	if err != nil {
		return "", fmt.Errorf("encode configuration fingerprint: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

// hasCycle 使用拓扑遍历判断完整直接规则图是否存在循环。
func hasCycle(rules []precedence.Rule) bool {
	graph := make(map[string][]string)
	indegree := make(map[string]int)
	for _, rule := range rules {
		graph[rule.PredecessorItemID] = append(graph[rule.PredecessorItemID], rule.SuccessorItemID)
		if _, exists := indegree[rule.PredecessorItemID]; !exists {
			indegree[rule.PredecessorItemID] = 0
		}
		indegree[rule.SuccessorItemID]++
	}
	queue := make([]string, 0, len(indegree))
	for itemID, degree := range indegree {
		if degree == 0 {
			queue = append(queue, itemID)
		}
	}
	visited := 0
	for len(queue) > 0 {
		itemID := queue[0]
		queue = queue[1:]
		visited++
		for _, next := range graph[itemID] {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	return visited != len(indegree)
}
