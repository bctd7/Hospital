package projectconfiguration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"hospital/common/authn"
	commonauthz "hospital/common/authz"
	contractauthz "hospital/contracts/authz"
	"hospital/service/guidance/rpc/internal/rules/precedence"
	"hospital/service/guidance/rpc/internal/rules/preparation"
)

// Manager 协调一次完整的项目配置。Appointment 的确认始终最后执行，
// 因而患者不会看到只有项目事实、尚无规则的半成品。
type Manager struct {
	store       Store
	appointment AppointmentParticipant
	projects    ProjectDirectory
}

func NewManager(store Store, appointment AppointmentParticipant, projects ProjectDirectory) (*Manager, error) {
	if store == nil || appointment == nil || projects == nil {
		return nil, errors.New("project configuration dependencies are required")
	}
	return &Manager{store: store, appointment: appointment, projects: projects}, nil
}

func (m *Manager) Preview(description string) (preparation.Preview, error) {
	return preparation.Parse(description)
}

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

func (m *Manager) Configure(ctx context.Context, operator authn.Principal, command Command) (Result, error) {
	if err := requirePermission(operator, contractauthz.PermissionRuleEdit); err != nil {
		return Result{}, err
	}
	command, err := m.normalizeAndResolve(ctx, operator, command)
	if err != nil {
		return Result{}, err
	}
	fingerprint, err := commandFingerprint(command)
	if err != nil {
		return Result{}, err
	}
	now := time.Now().UTC()
	transaction := Transaction{
		TransactionID: uuid.NewString(), OperationID: command.OperationID, Action: command.Action,
		ItemID: command.ItemID, OperatorAccountID: operator.AccountID, State: StateTrying,
		RequestFingerprint: fingerprint, Payload: command, CreatedAt: now, UpdatedAt: now,
	}

	err = m.store.WithinTransaction(ctx, func(tx TxStore) error {
		existing, found, findErr := tx.FindTransactionByOperation(ctx, command.OperationID)
		if findErr != nil {
			return findErr
		}
		if found {
			if existing.OperatorAccountID != operator.AccountID || existing.RequestFingerprint != fingerprint {
				return ErrConflict
			}
			transaction = existing
			return nil
		}
		return tx.CreateTransaction(ctx, transaction)
	})
	if err != nil {
		return Result{}, err
	}
	command = transaction.Payload

	if transaction.State == StateConfirmed {
		return m.Get(ctx, operator, command.ItemID)
	}
	if transaction.State == StateCancelled || transaction.State == StateCancelling {
		return Result{}, ErrConflict
	}
	if transaction.State == StateConfirming {
		return m.finishConfirmation(ctx, operator, transaction)
	}

	if err := m.appointment.Prepare(ctx, transaction.TransactionID, command); err != nil {
		m.recordFailure(ctx, transaction, err)
		return Result{}, err
	}
	if err := m.setState(ctx, transaction.TransactionID, StatePrepared, "", transaction.RetryCount); err != nil {
		_ = m.appointment.Cancel(ctx, transaction.TransactionID)
		return Result{}, err
	}
	transaction.State = StatePrepared

	if err := m.applyGuidanceConfiguration(ctx, operator, &transaction); err != nil {
		_ = m.appointment.Cancel(ctx, transaction.TransactionID)
		_ = m.setState(ctx, transaction.TransactionID, StateCancelled, err.Error(), transaction.RetryCount+1)
		return Result{}, err
	}
	transaction.State = StateConfirming
	return m.finishConfirmation(ctx, operator, transaction)
}

func (m *Manager) finishConfirmation(ctx context.Context, operator authn.Principal, transaction Transaction) (Result, error) {
	project, err := m.appointment.Confirm(ctx, transaction.TransactionID)
	if err == nil {
		if stateErr := m.setState(ctx, transaction.TransactionID, StateConfirmed, "", transaction.RetryCount); stateErr != nil {
			return Result{}, stateErr
		}
		configuration, loadErr := m.store.GetConfiguration(ctx, transaction.ItemID)
		if loadErr != nil {
			return Result{}, loadErr
		}
		rules, loadErr := m.store.ListRulesByOwner(ctx, transaction.ItemID)
		return Result{Project: project, Configuration: configuration, Rules: rules}, loadErr
	}

	// Confirm 的网络结果可能不确定：只有 Appointment 明确接受 Cancel 后才回滚本地规则。
	if cancelErr := m.appointment.Cancel(ctx, transaction.TransactionID); cancelErr != nil {
		m.recordFailure(ctx, transaction, err)
		return Result{}, err
	}
	if compensationErr := m.compensate(ctx, transaction); compensationErr != nil {
		return Result{}, compensationErr
	}
	_ = m.setState(ctx, transaction.TransactionID, StateCancelled, err.Error(), transaction.RetryCount+1)
	return Result{}, err
}

func (m *Manager) applyGuidanceConfiguration(ctx context.Context, operator authn.Principal, transaction *Transaction) error {
	command := transaction.Payload
	return m.store.WithinTransaction(ctx, func(tx TxStore) error {
		if err := tx.LockGraph(ctx); err != nil {
			return err
		}
		currentConfiguration, found, err := tx.GetConfigurationForUpdate(ctx, command.ItemID)
		if err != nil {
			return err
		}
		if command.Action == ActionCreate {
			if found || command.ExpectedConfigurationVersion != 0 {
				return ErrVersionConflict
			}
		} else if !found || currentConfiguration.Version != command.ExpectedConfigurationVersion {
			return ErrVersionConflict
		}
		beforeRules, err := tx.ListRulesByOwnerForUpdate(ctx, command.ItemID)
		if err != nil {
			return err
		}
		var before *Configuration
		if found {
			copy := currentConfiguration
			before = &copy
		}
		if err := tx.SetTransactionBefore(ctx, transaction.TransactionID, before, beforeRules, time.Now().UTC()); err != nil {
			return err
		}
		transaction.BeforeConfiguration = before
		transaction.BeforePrecedenceRules = beforeRules

		allRules, err := tx.ListAllRules(ctx)
		if err != nil {
			return err
		}
		remaining := make([]precedence.Rule, 0, len(allRules)+len(command.PrecedenceRules))
		for _, rule := range allRules {
			if rule.OwnerItemID != command.ItemID {
				remaining = append(remaining, rule)
			}
		}
		remaining = append(remaining, command.PrecedenceRules...)
		if hasCycle(remaining) {
			return ErrCycle
		}
		if err := tx.DeleteRulesByOwner(ctx, command.ItemID); err != nil {
			return err
		}
		for _, rule := range command.PrecedenceRules {
			if err := tx.InsertRule(ctx, rule); err != nil {
				return err
			}
		}
		now := time.Now().UTC()
		version := int64(1)
		createdAt := now
		if found {
			version = currentConfiguration.Version + 1
			createdAt = currentConfiguration.CreatedAt
		}
		configuration := Configuration{
			ItemID: command.ItemID, Description: command.Description,
			PreparationRules: command.PreparationRules, Reminders: command.Reminders,
			Version: version, UpdatedBy: operator.AccountID, CreatedAt: createdAt, UpdatedAt: now,
		}
		if err := tx.UpsertConfiguration(ctx, configuration); err != nil {
			return err
		}
		return tx.UpdateTransaction(ctx, transaction.TransactionID, StateConfirming, "", transaction.RetryCount, now)
	})
}

func (m *Manager) compensate(ctx context.Context, transaction Transaction) error {
	return m.store.WithinTransaction(ctx, func(tx TxStore) error {
		current, found, err := tx.GetTransactionForUpdate(ctx, transaction.TransactionID)
		if err != nil || !found {
			return err
		}
		if current.State == StateCancelled {
			return nil
		}
		if err := tx.LockGraph(ctx); err != nil {
			return err
		}
		if err := tx.DeleteRulesByOwner(ctx, transaction.ItemID); err != nil {
			return err
		}
		for _, rule := range current.BeforePrecedenceRules {
			if err := tx.InsertRule(ctx, rule); err != nil {
				return err
			}
		}
		if current.BeforeConfiguration == nil {
			if err := tx.DeleteConfiguration(ctx, transaction.ItemID); err != nil {
				return err
			}
		} else if err := tx.UpsertConfiguration(ctx, *current.BeforeConfiguration); err != nil {
			return err
		}
		return tx.UpdateTransaction(ctx, transaction.TransactionID, StateCancelled, current.LastError, current.RetryCount+1, time.Now().UTC())
	})
}

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
	if err != nil || preparation.Validate(command.PreparationRules, command.Reminders) != nil {
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

func (m *Manager) setState(ctx context.Context, transactionID, state, lastError string, retryCount int) error {
	return m.store.WithinTransaction(ctx, func(tx TxStore) error {
		return tx.UpdateTransaction(ctx, transactionID, state, lastError, retryCount, time.Now().UTC())
	})
}

func (m *Manager) recordFailure(ctx context.Context, transaction Transaction, cause error) {
	_ = m.setState(ctx, transaction.TransactionID, transaction.State, cause.Error(), transaction.RetryCount+1)
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
