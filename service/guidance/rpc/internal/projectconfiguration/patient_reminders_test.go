package projectconfiguration

import (
	"context"
	"testing"

	descriptionrules "hospital/service/guidance/rpc/internal/rules/description"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

const reminderTestItemID = "30000000-0000-4000-8000-000000000008"

type patientReminderStoreStub struct {
	configuration Configuration
	err           error
}

func (s patientReminderStoreStub) GetConfiguration(context.Context, string) (Configuration, error) {
	return s.configuration, s.err
}
func (patientReminderStoreStub) ListRulesByOwner(context.Context, string) ([]precedence.Rule, error) {
	return nil, nil
}
func (patientReminderStoreStub) WithinTransaction(context.Context, func(TxStore) error) error {
	panic("not used")
}

type patientReminderProjectDirectoryStub struct{}

func (patientReminderProjectDirectoryStub) ResolveProject(context.Context, string) (Project, error) {
	return Project{ItemID: reminderTestItemID, Status: "active"}, nil
}

func TestPatientRemindersReturnsOnlyConfiguredPatientReminders(t *testing.T) {
	want := "请按检查说明和医嘱提前用药。"
	manager := &Manager{
		store: patientReminderStoreStub{configuration: Configuration{
			ItemID:           reminderTestItemID,
			PreparationRules: []descriptionrules.Rule{{ReadinessHint: "出现明显尿意即可前往检查。"}},
			Reminders:        []descriptionrules.Reminder{{Text: want}},
		}},
		projects: patientReminderProjectDirectoryStub{},
	}

	got, err := manager.PatientReminders(context.Background(), reminderTestItemID)
	if err != nil {
		t.Fatalf("PatientReminders() error = %v", err)
	}
	if len(got) != 1 || got[0].Text != want {
		t.Fatalf("PatientReminders() = %#v, want only configured patient reminder", got)
	}
	got[0].Text = "changed"
	if manager.store.(patientReminderStoreStub).configuration.Reminders[0].Text != want {
		t.Fatal("PatientReminders() returned the store-owned slice")
	}
}

func TestPatientRemindersTreatsMissingConfigurationAsNoReminders(t *testing.T) {
	manager := &Manager{store: patientReminderStoreStub{err: ErrNotFound}, projects: patientReminderProjectDirectoryStub{}}
	got, err := manager.PatientReminders(context.Background(), reminderTestItemID)
	if err != nil {
		t.Fatalf("PatientReminders() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("PatientReminders() = %#v, want empty", got)
	}
}
