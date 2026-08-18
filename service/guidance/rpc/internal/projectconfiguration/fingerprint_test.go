package projectconfiguration

import (
	"testing"

	"hospital/service/guidance/rpc/internal/rules/precedence"
)

func TestCommandFingerprintDoesNotMutateResolvedRules(t *testing.T) {
	rule := precedence.Rule{RuleID: "rule-id", CreateOperationID: "operation-id", CreatedBy: "account-id", Version: 1}
	command := Command{PrecedenceRules: []precedence.Rule{rule}}
	if _, err := commandFingerprint(command); err != nil {
		t.Fatalf("commandFingerprint() error = %v", err)
	}
	got := command.PrecedenceRules[0]
	if got.RuleID != rule.RuleID || got.CreateOperationID != rule.CreateOperationID || got.CreatedBy != rule.CreatedBy || got.Version != rule.Version {
		t.Fatalf("fingerprint mutated resolved rule: %#v", got)
	}
}
