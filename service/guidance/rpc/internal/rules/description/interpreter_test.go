package description

import (
	"context"
	"errors"
	"testing"
)

type interpreterStub struct {
	preview Preview
	err     error
}

func (s interpreterStub) Interpret(context.Context, string) (Preview, error) {
	return s.preview, s.err
}

func TestParserAcceptsValidatedModelCandidate(t *testing.T) {
	parser := NewParser(interpreterStub{preview: Preview{Rules: []Rule{{
		RuleType: RuleTypeFasting, StartMode: StartModeAdvanceRange,
		MinAdvanceMinutes: 480, RecommendedAdvanceMinutes: 480, MaxAdvanceMinutes: 720,
	}}}})
	preview, err := parser.Parse(context.Background(), "空腹8至12小时")
	if err != nil || preview.ParserMode != "llm" || preview.Rules[0].Source != "model" {
		t.Fatalf("validated model candidate was not used: %#v, %v", preview, err)
	}
}

func TestParserFallsBackWhenModelCandidateIsInvalid(t *testing.T) {
	parser := NewParser(interpreterStub{preview: Preview{Rules: []Rule{{RuleType: "invented"}}}})
	preview, err := parser.Parse(context.Background(), "检查前空腹")
	if err != nil || preview.ParserMode != "deterministic" || preview.Warning == "" || len(preview.Rules) != 2 {
		t.Fatalf("invalid model candidate must fall back safely: %#v, %v", preview, err)
	}
}

func TestParserFallsBackWhenModelIsUnavailable(t *testing.T) {
	parser := NewParser(interpreterStub{err: errors.New("unavailable")})
	preview, err := parser.Parse(context.Background(), "检查前空腹")
	if err != nil || preview.ParserMode != "deterministic" || preview.Warning == "" {
		t.Fatalf("unavailable model must fall back safely: %#v, %v", preview, err)
	}
}
