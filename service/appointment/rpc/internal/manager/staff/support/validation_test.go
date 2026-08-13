package support

import (
	"errors"
	"strings"
	"testing"

	"hospital/service/appointment/rpc/internal/manager/common"
)

func TestNormalizeProjectDescriptionAllowsNaturalLanguageFormatting(t *testing.T) {
	value := "第一步：空腹\n第二步：携带资料\t原件"
	got, err := NormalizeProjectDescription(value)
	if err != nil {
		t.Fatal(err)
	}
	if got != value {
		t.Fatalf("description changed: got %q want %q", got, value)
	}
}

func TestNormalizeProjectDescriptionRejectsUnsupportedControl(t *testing.T) {
	_, err := NormalizeProjectDescription("有效内容\x00隐藏内容")
	if !errors.Is(err, common.ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
}

func TestNormalizeProjectDescriptionRejectsOversizedValue(t *testing.T) {
	_, err := NormalizeProjectDescription(strings.Repeat("检", maxProjectDescriptionRunes+1))
	if !errors.Is(err, common.ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
}
