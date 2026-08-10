package login

import (
	"context"
	"errors"
	"strings"
	"testing"

	dypns "github.com/alibabacloud-go/dypnsapi-20170525/v3/client"
	"github.com/alibabacloud-go/tea/dara"
)

func TestAlibabaPNVSSendsProviderGeneratedCodeWithoutReturningIt(t *testing.T) {
	client := &fakeAlibabaPNVSClient{}
	provider := newAlibabaPNVS(client, AlibabaPNVSConfig{
		SignName: "system-sign", TemplateCode: "100001", SchemeName: "hospital-login",
		ValidSeconds: 300, IntervalSeconds: 60, CodeLength: 6,
	})

	if err := provider.SendLoginCode(context.Background(), "+8613800138000"); err != nil {
		t.Fatal(err)
	}
	request := client.sendRequest
	if request == nil || dara.StringValue(request.PhoneNumber) != "13800138000" {
		t.Fatalf("unexpected phone request: %#v", request)
	}
	if dara.BoolValue(request.ReturnVerifyCode) {
		t.Fatal("verification code must not be returned to Hospital")
	}
	if !strings.Contains(dara.StringValue(request.TemplateParam), "##code##") {
		t.Fatalf("expected provider-generated code placeholder: %q", dara.StringValue(request.TemplateParam))
	}
}

func TestAlibabaPNVSRequiresPassVerificationResult(t *testing.T) {
	client := &fakeAlibabaPNVSClient{verifyResult: "UNKNOWN"}
	provider := newAlibabaPNVS(client, AlibabaPNVSConfig{})

	err := provider.VerifyLoginCode(context.Background(), "+8613800138000", "123456")
	if !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("expected invalid credential, got %v", err)
	}
}

type fakeAlibabaPNVSClient struct {
	sendRequest  *dypns.SendSmsVerifyCodeRequest
	sendError    error
	verifyResult string
}

func (c *fakeAlibabaPNVSClient) SendSmsVerifyCodeWithContext(_ context.Context, request *dypns.SendSmsVerifyCodeRequest, _ *dara.RuntimeOptions) (*dypns.SendSmsVerifyCodeResponse, error) {
	c.sendRequest = request
	if c.sendError != nil {
		return nil, c.sendError
	}
	return &dypns.SendSmsVerifyCodeResponse{Body: &dypns.SendSmsVerifyCodeResponseBody{
		Code: dara.String("OK"), Success: dara.Bool(true),
	}}, nil
}

func TestAlibabaPNVSPreservesOnlyProviderErrorCode(t *testing.T) {
	client := &fakeAlibabaPNVSClient{sendError: dara.NewSDKError(map[string]any{
		"code":    "InvalidAccessKeyId.NotFound",
		"message": "sensitive provider detail",
	})}
	provider := newAlibabaPNVS(client, AlibabaPNVSConfig{})

	err := provider.SendLoginCode(context.Background(), "+8613800138000")
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expected unavailable provider, got %v", err)
	}
	if !strings.Contains(err.Error(), "InvalidAccessKeyId.NotFound") {
		t.Fatalf("provider code was not preserved: %v", err)
	}
	if strings.Contains(err.Error(), "sensitive provider detail") {
		t.Fatalf("provider detail must not enter logs: %v", err)
	}
}

func (c *fakeAlibabaPNVSClient) CheckSmsVerifyCodeWithContext(_ context.Context, _ *dypns.CheckSmsVerifyCodeRequest, _ *dara.RuntimeOptions) (*dypns.CheckSmsVerifyCodeResponse, error) {
	return &dypns.CheckSmsVerifyCodeResponse{Body: &dypns.CheckSmsVerifyCodeResponseBody{
		Code: dara.String("OK"), Success: dara.Bool(true),
		Model: &dypns.CheckSmsVerifyCodeResponseBodyModel{VerifyResult: dara.String(c.verifyResult)},
	}}, nil
}
