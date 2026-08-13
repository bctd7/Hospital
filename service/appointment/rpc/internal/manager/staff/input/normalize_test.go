package input

import (
	"errors"
	"testing"

	"hospital/service/appointment/rpc/internal/manager/common"
)

const (
	validationDepartmentID = "00000000-0000-0000-0000-000000000010"
	validationItemID       = "00000000-0000-0000-0000-000000000011"
	validationOperationID  = "00000000-0000-0000-0000-000000000012"
)

func TestNormalizeCreateProject(t *testing.T) {
	value, err := NormalizeCreateProject(CreateProject{
		OwnerDepartmentID: "  " + validationDepartmentID + "  ",
		Name:              "  腹部 CT  ",
		Description:       "  检查前禁食。\r\n可少量饮水。  ",
		OperationID:       validationOperationID,
		RequestID:         " request-1 ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if value.OwnerDepartmentID != validationDepartmentID || value.Name != "腹部 CT" ||
		value.Description != "检查前禁食。\r\n可少量饮水。" || value.RequestID != "request-1" {
		t.Fatalf("unexpected normalized input: %#v", value)
	}
}

func TestNormalizeUpdateProjectPreservesAbsentFields(t *testing.T) {
	description := "  无特殊准备  "
	value, err := NormalizeUpdateProject(UpdateProject{
		ItemID:          validationItemID,
		Description:     &description,
		ExpectedVersion: 2,
		OperationID:     validationOperationID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if value.Name != nil || value.Description == nil || *value.Description != "无特殊准备" {
		t.Fatalf("unexpected normalized input: %#v", value)
	}
}

func TestNormalizeUpdateProjectRequiresAChange(t *testing.T) {
	_, err := NormalizeUpdateProject(UpdateProject{
		ItemID: validationItemID, ExpectedVersion: 1, OperationID: validationOperationID,
	})
	if !errors.Is(err, common.ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
}
