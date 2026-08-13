package staff

import (
	"errors"
	"strings"
	"testing"
)

const (
	validationDepartmentID = "00000000-0000-0000-0000-000000000010"
	validationItemID       = "00000000-0000-0000-0000-000000000011"
	validationOperationID  = "00000000-0000-0000-0000-000000000012"
)

func TestNormalizedCreateCommand(t *testing.T) {
	command, err := normalizedCreateProjectCommand(CreateProjectCommand{
		OwnerDepartmentID: "  " + validationDepartmentID + "  ",
		Name:              "  腹部 CT  ",
		Description:       "  检查前禁食。\n可少量饮水。  ",
		OperationID:       validationOperationID,
		RequestID:         " request-1 ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if command.OwnerDepartmentID != validationDepartmentID || command.Name != "腹部 CT" ||
		command.Description != "检查前禁食。\n可少量饮水。" || command.RequestID != "request-1" {
		t.Fatalf("unexpected normalized command: %#v", command)
	}
}

func TestNormalizedCatalogDescriptionAllowsNaturalLanguageFormatting(t *testing.T) {
	value := "第一步：空腹\n第二步：携带资料\t原件"
	got, err := normalizedCatalogDescription(value)
	if err != nil {
		t.Fatal(err)
	}
	if got != value {
		t.Fatalf("description changed: got %q want %q", got, value)
	}
}

func TestNormalizedCatalogDescriptionRejectsUnsupportedControl(t *testing.T) {
	_, err := normalizedCatalogDescription("有效内容\x00隐藏内容")
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
}

func TestNormalizedCatalogDescriptionRejectsOversizedValue(t *testing.T) {
	_, err := normalizedCatalogDescription(strings.Repeat("检", maxCatalogDescriptionRunes+1))
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
}

func TestNormalizedUpdateCommandPreservesAbsentFields(t *testing.T) {
	description := "  无特殊准备  "
	command, err := normalizedUpdateProjectCommand(UpdateProjectCommand{
		ItemID:          validationItemID,
		Description:     &description,
		ExpectedVersion: 2,
		OperationID:     validationOperationID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if command.Name != nil || command.Description == nil || *command.Description != "无特殊准备" {
		t.Fatalf("unexpected normalized command: %#v", command)
	}
}

func TestNormalizedUpdateCommandRequiresAChange(t *testing.T) {
	_, err := normalizedUpdateProjectCommand(UpdateProjectCommand{
		ItemID: validationItemID, ExpectedVersion: 1, OperationID: validationOperationID,
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
}

func TestNormalizedListQueryAppliesPageDefaults(t *testing.T) {
	query, err := normalizedListProjectsQuery(ListProjectsQuery{Status: StatusActive})
	if err != nil {
		t.Fatal(err)
	}
	if query.Page != defaultProjectPage || query.PageSize != defaultProjectPageSize || query.Status != StatusActive {
		t.Fatalf("unexpected normalized query: %#v", query)
	}
}

func TestNormalizedListQueryRejectsInvalidStatusAndPage(t *testing.T) {
	for _, query := range []ListProjectsQuery{
		{Status: "deleted"},
		{Page: -1, PageSize: 20},
		{Page: 1, PageSize: maxProjectPageSize + 1},
	} {
		if _, err := normalizedListProjectsQuery(query); !errors.Is(err, ErrInvalid) {
			t.Fatalf("query %#v error = %v, want ErrInvalid", query, err)
		}
	}
}
