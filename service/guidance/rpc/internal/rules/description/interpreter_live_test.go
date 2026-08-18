package description

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLiveLLMInterpreter 可选验证当前配置模型与封闭规则契约。
// 常规测试没有配置 Key 时自动跳过，不把第三方模型作为构建前提。
func TestLiveLLMInterpreter(t *testing.T) {
	endpoint := strings.TrimSpace(os.Getenv("GUIDANCE_LLM_ENDPOINT"))
	key := strings.TrimSpace(os.Getenv("GUIDANCE_LLM_API_KEY"))
	model := strings.TrimSpace(os.Getenv("GUIDANCE_LLM_MODEL"))
	if endpoint == "" || key == "" || model == "" {
		t.Skip("guidance LLM is not configured")
	}
	parser := NewParser(NewLLMInterpreter(LLMConfig{Endpoint: endpoint, APIKey: key, Model: model, Timeout: 20 * time.Second}))
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	preview, err := parser.Parse(ctx, "检查前需空腹8～12小时；怀孕或可能怀孕请提前告知工作人员。")
	if err != nil {
		t.Fatal(err)
	}
	if preview.Rules == nil || preview.Reminders == nil || preview.UnresolvedFragments == nil {
		t.Fatalf("live preview contains null collections: %#v", preview)
	}
	if len(preview.Rules) != 2 || preview.Rules[0].MinAdvanceMinutes != 480 {
		t.Fatalf("unexpected live preparation rules: %#v", preview)
	}
}
