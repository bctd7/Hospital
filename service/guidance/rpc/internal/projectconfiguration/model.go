// Package projectconfiguration 编排 Appointment 项目事实和 Guidance 规则的一次性完整配置。
package projectconfiguration

import (
	"time"

	descriptionrules "hospital/service/guidance/rpc/internal/rules/description"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

const (
	ActionCreate = "create"
	ActionUpdate = "update"

	StateTrying     = "trying"
	StatePrepared   = "prepared"
	StateConfirming = "confirming"
	StateConfirmed  = "confirmed"
	StateCancelling = "cancelling"
	StateCancelled  = "cancelled"
)

type Configuration struct {
	ItemID           string                      `json:"item_id"`
	Description      string                      `json:"description"`
	PreparationRules []descriptionrules.Rule     `json:"preparation_rules"`
	Reminders        []descriptionrules.Reminder `json:"reminders"`
	Version          int64                       `json:"version"`
	UpdatedBy        string                      `json:"updated_by"`
	CreatedAt        time.Time                   `json:"created_at"`
	UpdatedAt        time.Time                   `json:"updated_at"`
}

type Project struct {
	ItemID                   string
	OwnerDepartmentID        string
	Name                     string
	Status                   string
	Version                  int64
	EstimatedDurationMinutes int32
}

type Command struct {
	Action                       string                      `json:"action"`
	ItemID                       string                      `json:"item_id"`
	OwnerDepartmentID            string                      `json:"owner_department_id"`
	ItemName                     string                      `json:"item_name"`
	EstimatedDurationMinutes     int32                       `json:"estimated_duration_minutes"`
	ExpectedItemVersion          int64                       `json:"expected_item_version"`
	Description                  string                      `json:"description"`
	PrecedenceRules              []precedence.Rule           `json:"precedence_rules"`
	PreparationRules             []descriptionrules.Rule     `json:"preparation_rules"`
	Reminders                    []descriptionrules.Reminder `json:"reminders"`
	ExpectedConfigurationVersion int64                       `json:"expected_configuration_version"`
	OperationID                  string                      `json:"operation_id"`
	RequestID                    string                      `json:"request_id"`
}

type Transaction struct {
	TransactionID         string
	OperationID           string
	Action                string
	ItemID                string
	OperatorAccountID     string
	State                 string
	RequestFingerprint    string
	Payload               Command
	BeforeConfiguration   *Configuration
	BeforePrecedenceRules []precedence.Rule
	LastError             string
	RetryCount            int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type Result struct {
	Project       Project
	Configuration Configuration
	Rules         []precedence.Rule
}
