package svc

import (
	"fmt"
	"strings"

	"hospital/service/identity/rpc/internal/authentication/provider"
	"hospital/service/identity/rpc/internal/config"
)

// loginProviders 汇总外部登录渠道适配器。
// Provider 只验证或交换外部凭据，不创建账号、不签发 Token，也不写 Refresh Session。
type loginProviders struct {
	weChat provider.WeChatProvider
	phone  provider.PhoneVerificationProvider
}

// buildLoginProviders 根据配置选择登录渠道实现，渠道差异到此为止，不进入 RPC Logic。
func buildLoginProviders(c config.Config) (loginProviders, error) {
	phone, err := phoneVerificationProvider(c)
	if err != nil {
		return loginProviders{}, err
	}
	return loginProviders{
		weChat: provider.NewWeChatClient(c.WeChat.AppID, c.WeChat.AppSecret, c.WeChat.Code2SessionURL, nil),
		phone:  phone,
	}, nil
}

// phoneVerificationProvider 选择手机号验证码实现。
// local 实现仅允许本地和测试环境使用，防止固定验证码误入生产环境。
func phoneVerificationProvider(c config.Config) (provider.PhoneVerificationProvider, error) {
	providerName := strings.ToLower(strings.TrimSpace(c.PhoneLogin.Provider))
	if providerName == "" {
		if c.PhoneLogin.Enabled {
			providerName = "aliyun"
		} else {
			providerName = "disabled"
		}
	}

	switch providerName {
	case "disabled":
		return provider.UnconfiguredPhoneVerificationProvider{}, nil
	case "aliyun":
		value, err := provider.NewAlibabaPNVS(provider.AlibabaPNVSConfig{
			AccessKeyID: c.PhoneLogin.AccessKeyID, AccessKeySecret: c.PhoneLogin.AccessKeySecret,
			RegionID: c.PhoneLogin.RegionID, Endpoint: c.PhoneLogin.Endpoint,
			SignName: c.PhoneLogin.SignName, TemplateCode: c.PhoneLogin.TemplateCode,
			SchemeName: c.PhoneLogin.SchemeName, ValidSeconds: c.PhoneLogin.ValidSeconds,
			IntervalSeconds: c.PhoneLogin.IntervalSeconds, CodeLength: c.PhoneLogin.CodeLength,
		})
		if err != nil {
			return nil, fmt.Errorf("create aliyun phone login provider: %w", err)
		}
		return value, nil
	case "local":
		environment := strings.ToLower(strings.TrimSpace(c.Environment))
		if environment != "local" && environment != "test" {
			return nil, fmt.Errorf("local SMS provider is restricted to local/test; current environment is %q", c.Environment)
		}
		value, err := provider.NewLocalPhoneVerificationProvider(c.PhoneLogin.LocalCode)
		if err != nil {
			return nil, fmt.Errorf("create local phone login provider: %w", err)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported phone login provider %q", providerName)
	}
}
