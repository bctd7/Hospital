package svc

import (
	"fmt"
	"strings"

	accountmanager "hospital/service/identity/rpc/internal/account/manager"
	"hospital/service/identity/rpc/internal/authentication"
	"hospital/service/identity/rpc/internal/authentication/provider"
	authorizationmanager "hospital/service/identity/rpc/internal/authorization/manager"
	"hospital/service/identity/rpc/internal/config"
	organizationmanager "hospital/service/identity/rpc/internal/organization/manager"
	"hospital/service/identity/rpc/internal/repository/mysqlstore"
	"hospital/service/identity/rpc/internal/session"
)

func buildManagers(c config.Config, store *mysqlstore.Store, sessions *session.Manager, phoneLookupKey []byte) (Managers, error) {
	phoneProvider, err := phoneVerificationProvider(c)
	if err != nil {
		return Managers{}, err
	}
	authenticationManager, err := authentication.NewManager(
		store,
		provider.NewWeChatClient(c.WeChat.AppID, c.WeChat.AppSecret, c.WeChat.Code2SessionURL, nil),
		sessions,
		c.WeChat.AppID,
		phoneLookupKey,
	)
	if err != nil {
		return Managers{}, fmt.Errorf("create identity authentication manager: %w", err)
	}
	phoneLoginManager, err := authentication.NewPhoneLoginManager(
		store, phoneProvider, sessions, phoneLookupKey, c.PhoneLogin.IntervalSeconds,
	)
	if err != nil {
		return Managers{}, fmt.Errorf("create phone login manager: %w", err)
	}
	authorizationManager, err := authorizationmanager.NewManager(store)
	if err != nil {
		return Managers{}, fmt.Errorf("create identity authorization manager: %w", err)
	}
	accountManager, err := accountmanager.NewManager(store, phoneLookupKey)
	if err != nil {
		return Managers{}, fmt.Errorf("create identity account manager: %w", err)
	}
	return Managers{
		Authentication: authenticationManager, PhoneLogin: phoneLoginManager,
		Session: sessions, Authorization: authorizationManager, Account: accountManager,
		OrganizationUnit:      organizationmanager.NewUnitManager(store),
		OrganizationDirectory: organizationmanager.NewDirectoryManager(store),
	}, nil
}

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
