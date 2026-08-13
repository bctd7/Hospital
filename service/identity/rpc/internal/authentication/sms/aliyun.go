package sms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dypns "github.com/alibabacloud-go/dypnsapi-20170525/v3/client"
	"github.com/alibabacloud-go/tea/dara"
)

const aliyunPNVSEndpoint = "dypnsapi.aliyuncs.com"

type AliyunConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	RegionID        string
	Endpoint        string
	SignName        string
	TemplateCode    string
	SchemeName      string
	ValidSeconds    int64
	IntervalSeconds int64
	CodeLength      int64
}

type aliyunClient interface {
	SendSmsVerifyCodeWithContext(context.Context, *dypns.SendSmsVerifyCodeRequest, *dara.RuntimeOptions) (*dypns.SendSmsVerifyCodeResponse, error)
	CheckSmsVerifyCodeWithContext(context.Context, *dypns.CheckSmsVerifyCodeRequest, *dara.RuntimeOptions) (*dypns.CheckSmsVerifyCodeResponse, error)
}

// AliyunVerifier 使用阿里云号码认证服务生成、发送并校验验证码。
// ReturnVerifyCode 始终为 false，Identity 不接收生产验证码原文。
type AliyunVerifier struct {
	client          aliyunClient
	signName        string
	templateCode    string
	schemeName      string
	validSeconds    int64
	intervalSeconds int64
	codeLength      int64
}

func NewAliyunVerifier(config AliyunConfig) (*AliyunVerifier, error) {
	config.AccessKeyID = strings.TrimSpace(config.AccessKeyID)
	config.AccessKeySecret = strings.TrimSpace(config.AccessKeySecret)
	config.SignName = strings.TrimSpace(config.SignName)
	config.TemplateCode = strings.TrimSpace(config.TemplateCode)
	if config.AccessKeyID == "" || config.AccessKeySecret == "" || config.SignName == "" || config.TemplateCode == "" {
		return nil, ErrNotConfigured
	}
	if config.RegionID == "" {
		config.RegionID = "cn-shanghai"
	}
	if config.Endpoint == "" {
		config.Endpoint = aliyunPNVSEndpoint
	}
	if config.ValidSeconds <= 0 {
		config.ValidSeconds = 300
	}
	if config.IntervalSeconds <= 0 {
		config.IntervalSeconds = 60
	}
	if config.CodeLength < 4 || config.CodeLength > 8 {
		config.CodeLength = 6
	}

	client, err := dypns.NewClient(&openapi.Config{
		AccessKeyId:     dara.String(config.AccessKeyID),
		AccessKeySecret: dara.String(config.AccessKeySecret),
		RegionId:        dara.String(config.RegionID),
		Endpoint:        dara.String(config.Endpoint),
	})
	if err != nil {
		return nil, fmt.Errorf("create Alibaba Cloud PNVS client: %w", err)
	}
	return newAliyunVerifier(client, config), nil
}

func newAliyunVerifier(client aliyunClient, config AliyunConfig) *AliyunVerifier {
	return &AliyunVerifier{
		client: client, signName: config.SignName, templateCode: config.TemplateCode,
		schemeName: config.SchemeName, validSeconds: config.ValidSeconds,
		intervalSeconds: config.IntervalSeconds, codeLength: config.CodeLength,
	}
}

func (p *AliyunVerifier) SendLoginCode(ctx context.Context, phoneNumber string) error {
	minutes := (p.validSeconds + 59) / 60
	templateParam, err := json.Marshal(map[string]string{"code": "##code##", "min": fmt.Sprint(minutes)})
	if err != nil {
		return fmt.Errorf("build PNVS template parameters: %w", err)
	}
	request := (&dypns.SendSmsVerifyCodeRequest{}).
		SetCountryCode("86").
		SetPhoneNumber(mainlandDigits(phoneNumber)).
		SetSignName(p.signName).
		SetTemplateCode(p.templateCode).
		SetTemplateParam(string(templateParam)).
		SetInterval(p.intervalSeconds).
		SetValidTime(p.validSeconds).
		SetCodeType(1).
		SetCodeLength(p.codeLength).
		SetDuplicatePolicy(1).
		SetReturnVerifyCode(false).
		SetAutoRetry(1)
	if p.schemeName != "" {
		request.SetSchemeName(p.schemeName)
	}
	response, err := p.client.SendSmsVerifyCodeWithContext(ctx, request, &dara.RuntimeOptions{})
	if err != nil {
		if isPNVSRateLimitCode(pnvsErrorCode(err)) {
			return ErrRateLimited
		}
		return pnvsUnavailable("send verification code", err)
	}
	if response == nil || response.Body == nil || dara.StringValue(response.Body.Code) != "OK" || !dara.BoolValue(response.Body.Success) {
		if response != nil && response.Body != nil && isPNVSRateLimitCode(dara.StringValue(response.Body.Code)) {
			return ErrRateLimited
		}
		if response != nil && response.Body != nil {
			return pnvsUnavailableCode("send verification code", dara.StringValue(response.Body.Code))
		}
		return pnvsUnavailable("send verification code", nil)
	}
	return nil
}

func (p *AliyunVerifier) VerifyLoginCode(ctx context.Context, phoneNumber, code string) error {
	request := (&dypns.CheckSmsVerifyCodeRequest{}).
		SetCountryCode("86").
		SetPhoneNumber(mainlandDigits(phoneNumber)).
		SetVerifyCode(code).
		SetCaseAuthPolicy(2)
	if p.schemeName != "" {
		request.SetSchemeName(p.schemeName)
	}
	response, err := p.client.CheckSmsVerifyCodeWithContext(ctx, request, &dara.RuntimeOptions{})
	if err != nil {
		return pnvsUnavailable("check verification code", err)
	}
	if response == nil || response.Body == nil || dara.StringValue(response.Body.Code) != "OK" || !dara.BoolValue(response.Body.Success) {
		if response != nil && response.Body != nil {
			return pnvsUnavailableCode("check verification code", dara.StringValue(response.Body.Code))
		}
		return pnvsUnavailable("check verification code", nil)
	}
	if response.Body.Model == nil || dara.StringValue(response.Body.Model.VerifyResult) != "PASS" {
		return ErrInvalidCode
	}
	return nil
}

func mainlandDigits(normalized string) string {
	return strings.TrimPrefix(normalized, "+86")
}

func isPNVSRateLimitCode(code string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	return normalized == "FREQUENCY_FAIL" ||
		normalized == "BIZ.FREQUENCY" ||
		normalized == "BUSINESS_LIMIT_CONTROL" ||
		normalized == "ISV.BUSINESS_LIMIT_CONTROL"
}

// pnvsUnavailable 只保留短信平台返回的机器可读错误码。
// 客户端仍收到统一的不可用错误；服务端日志不会记录手机号、凭据或响应正文。
func pnvsUnavailable(operation string, cause error) error {
	if code := pnvsErrorCode(cause); code != "" {
		return pnvsUnavailableCode(operation, code)
	}
	return fmt.Errorf("%w: PNVS %s failed", ErrUnavailable, operation)
}

func pnvsErrorCode(cause error) string {
	var providerError dara.BaseError
	if errors.As(cause, &providerError) {
		return strings.TrimSpace(dara.StringValue(providerError.GetCode()))
	}
	return ""
}

func pnvsUnavailableCode(operation, code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return pnvsUnavailable(operation, nil)
	}
	return fmt.Errorf("%w: PNVS %s failed (code=%s)", ErrUnavailable, operation, code)
}
