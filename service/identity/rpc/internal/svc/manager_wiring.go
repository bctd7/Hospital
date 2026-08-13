package svc

import (
	"fmt"
	"time"

	accountmanager "hospital/service/identity/rpc/internal/account/manager"
	authenticationmanager "hospital/service/identity/rpc/internal/authentication/manager"
	"hospital/service/identity/rpc/internal/config"
	organizationmanager "hospital/service/identity/rpc/internal/organization/manager"
	"hospital/service/identity/rpc/internal/session"
)

// wireManagers 只负责把已经创建好的资源、密钥组件和登录 Provider 注入业务 Manager。
// 这里不读取外部系统、不启动后台任务，也不承载任何 RPC 业务规则。
func wireManagers(
	c config.Config,
	resourceSet serviceResources,
	tokens tokenComponents,
	providers loginProviders,
) (Managers, error) {
	sessionManager, err := session.NewManager(
		tokens.refreshSessions,
		resourceSet.identityStore,
		tokens.tokenManager,
		tokens.authorizationVersions,
		time.Duration(c.Token.RefreshTTLSeconds)*time.Second,
	)
	if err != nil {
		return Managers{}, fmt.Errorf("create identity session manager: %w", err)
	}
	authenticationManager, err := authenticationmanager.NewManager(
		resourceSet.identityStore,
		providers.weChat,
		sessionManager,
		c.WeChat.AppID,
		tokens.phoneLookupKey,
	)
	if err != nil {
		return Managers{}, fmt.Errorf("create identity authentication manager: %w", err)
	}
	phoneLoginManager, err := authenticationmanager.NewPhoneLoginManager(
		resourceSet.identityStore,
		providers.phone,
		sessionManager,
		tokens.phoneLookupKey,
		c.PhoneLogin.IntervalSeconds,
	)
	if err != nil {
		return Managers{}, fmt.Errorf("create phone login manager: %w", err)
	}
	accountManager, err := accountmanager.NewManager(resourceSet.identityStore, tokens.phoneLookupKey)
	if err != nil {
		return Managers{}, fmt.Errorf("create identity account manager: %w", err)
	}

	return Managers{
		Authentication:        authenticationManager,
		PhoneLogin:            phoneLoginManager,
		Session:               sessionManager,
		Account:               accountManager,
		OrganizationUnit:      organizationmanager.NewUnitManager(resourceSet.identityStore),
		OrganizationDirectory: organizationmanager.NewDirectoryManager(resourceSet.identityStore),
	}, nil
}
