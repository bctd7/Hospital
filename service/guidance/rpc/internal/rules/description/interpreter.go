package description

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Interpreter 只把自然语言转换为候选封闭结构，不拥有保存规则的权限。
type Interpreter interface {
	Interpret(ctx context.Context, description string) (Preview, error)
}

// Parser 优先使用外部模型理解自然语言，模型不可用或输出非法时降级到确定性解析。
type Parser struct {
	interpreter Interpreter
}

func NewParser(interpreter Interpreter) *Parser {
	return &Parser{interpreter: interpreter}
}

func (p *Parser) Parse(ctx context.Context, value string) (Preview, error) {
	fallback, err := Parse(value)
	if err != nil || p == nil || p.interpreter == nil || strings.TrimSpace(value) == "" {
		return fallback, err
	}
	preview, interpretErr := p.interpreter.Interpret(ctx, value)
	if interpretErr == nil {
		preview.Description = strings.TrimSpace(value)
		preview.ParserMode = "llm"
		for index := range preview.Rules {
			if preview.Rules[index].Source == "" {
				preview.Rules[index].Source = "model"
			}
		}
		if Validate(preview.Rules, preview.Reminders) == nil {
			return preview, nil
		}
	}
	fallback.Warning = "模型解析暂不可用或结果未通过校验，当前展示医院默认解析结果，请人工确认。"
	return fallback, nil
}

type LLMConfig struct {
	Endpoint string
	APIKey   string
	Model    string
	Timeout  time.Duration
}

type llmInterpreter struct {
	endpoint string
	apiKey   string
	model    string
	client   *http.Client
}

// NewLLMInterpreter 创建兼容 OpenAI Chat Completions JSON 响应格式的解释器。
// 未配置 endpoint 或 model 时返回 nil，由 Parser 使用确定性解析。
func NewLLMInterpreter(config LLMConfig) Interpreter {
	endpoint, model := strings.TrimSpace(config.Endpoint), strings.TrimSpace(config.Model)
	if endpoint == "" || model == "" {
		return nil
	}
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &llmInterpreter{endpoint: endpoint, apiKey: strings.TrimSpace(config.APIKey), model: model, client: &http.Client{Timeout: timeout}}
}

func (i *llmInterpreter) Interpret(ctx context.Context, description string) (Preview, error) {
	payload := map[string]any{
		"model":       i.model,
		"temperature": 0,
		"messages": []map[string]string{
			{"role": "system", "content": interpretationPrompt},
			{"role": "user", "content": description},
		},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return Preview{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, i.endpoint, bytes.NewReader(encoded))
	if err != nil {
		return Preview{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	if i.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+i.apiKey)
	}
	response, err := i.client.Do(request)
	if err != nil {
		return Preview{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return Preview{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Preview{}, fmt.Errorf("llm returned status %d", response.StatusCode)
	}
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || len(envelope.Choices) == 0 {
		return Preview{}, fmt.Errorf("invalid llm response")
	}
	content := strings.TrimSpace(envelope.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	var preview Preview
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &preview); err != nil {
		return Preview{}, fmt.Errorf("decode llm preparation rules: %w", err)
	}
	return preview, nil
}

const interpretationPrompt = `你是医院检查准备说明的结构化解析器。只返回一个 JSON 对象，不要解释。
允许的 rules[].rule_type 只有 fasting、no_water、drink_water。
允许的 start_mode 只有 advance_range、previous_day_time。
原文明确给出“提前8至12小时”等相对时长时，必须使用 advance_range 和分钟数，不得改成固定钟点。
只有原文没有给出时长且明确要求空腹、禁食或禁水时，才可使用 previous_day_time="20:00"。
空腹同时生成 fasting 和 no_water。提前用药、泻药、怀孕、月经等内容只能放入 reminders。
同一检查项目不得同时生成 no_water 和 drink_water。
无法可靠归类的原文片段放入 unresolved_fragments。
空腹示例结构：{"rules":[{"rule_type":"fasting","start_mode":"advance_range","min_advance_minutes":480,"recommended_advance_minutes":480,"max_advance_minutes":720,"previous_day_time":"","readiness_hint":"","source":"model"},{"rule_type":"no_water","start_mode":"advance_range","min_advance_minutes":480,"recommended_advance_minutes":480,"max_advance_minutes":720,"previous_day_time":"","readiness_hint":"","source":"model"}],"reminders":[{"text":"...","advance_minutes":0}],"unresolved_fragments":[]}`
