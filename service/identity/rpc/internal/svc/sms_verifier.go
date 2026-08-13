package svc

import (
	"fmt"
	"strings"

	"hospital/service/identity/rpc/internal/authentication/sms"
	"hospital/service/identity/rpc/internal/config"
)

// buildSMSVerifier 根据配置选择短信验证码实现。
// local 实现仅允许本地和测试环境使用，防止固定验证码误入生产环境。
func buildSMSVerifier(c config.Config) (sms.Verifier, error) {
	verifierName := strings.ToLower(strings.TrimSpace(c.PhoneLogin.Verifier))
	if verifierName == "" {
		verifierName = "disabled"
	}

	switch verifierName {
	case "disabled":
		return sms.UnconfiguredVerifier{}, nil
	case "aliyun":
		value, err := sms.NewAliyunVerifier(sms.AliyunConfig{
			AccessKeyID: c.PhoneLogin.AccessKeyID, AccessKeySecret: c.PhoneLogin.AccessKeySecret,
			RegionID: c.PhoneLogin.RegionID, Endpoint: c.PhoneLogin.Endpoint,
			SignName: c.PhoneLogin.SignName, TemplateCode: c.PhoneLogin.TemplateCode,
			SchemeName: c.PhoneLogin.SchemeName, ValidSeconds: c.PhoneLogin.ValidSeconds,
			IntervalSeconds: c.PhoneLogin.IntervalSeconds, CodeLength: c.PhoneLogin.CodeLength,
		})
		if err != nil {
			return nil, fmt.Errorf("create Aliyun SMS verifier: %w", err)
		}
		return value, nil
	case "local":
		environment := strings.ToLower(strings.TrimSpace(c.Environment))
		if environment != "local" && environment != "test" {
			return nil, fmt.Errorf("local SMS verifier is restricted to local/test; current environment is %q", c.Environment)
		}
		value, err := sms.NewLocalVerifier(c.PhoneLogin.LocalCode)
		if err != nil {
			return nil, fmt.Errorf("create local SMS verifier: %w", err)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported SMS verifier %q", verifierName)
	}
}
