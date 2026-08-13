package svc

import (
	"hospital/common/authn"
	commonauthversion "hospital/common/authz/version"
	accountmanager "hospital/service/identity/rpc/internal/account/manager"
	authenticationmanager "hospital/service/identity/rpc/internal/authentication/manager"
	authorizationversion "hospital/service/identity/rpc/internal/authorization/version"
	"hospital/service/identity/rpc/internal/config"
	"hospital/service/identity/rpc/internal/messaging/kafka"
	"hospital/service/identity/rpc/internal/messaging/outbox"
	organizationmanager "hospital/service/identity/rpc/internal/organization/manager"
	"hospital/service/identity/rpc/internal/session"
)

// Managers 是 RPC Logic 可以调用的业务入口集合。
// ServiceContext 只保存这些入口，不在这里实现账号、授权或会话规则。
type Managers struct {
	Authentication        *authenticationmanager.Manager
	Session               *session.Manager
	Account               *accountmanager.Manager
	OrganizationUnit      *organizationmanager.UnitManager
	OrganizationDirectory *organizationmanager.DirectoryManager
}

// Security 是拦截器所需的安全组件，不向 Logic 暴露 Redis 等实现细节。
type Security struct {
	Token                *authn.TokenManager
	AuthorizationVersion *commonauthversion.Validator
}

// Workers 是可选的后台消息任务；Kafka 未启用时对应字段为 nil。
type Workers struct {
	OutboxPublisher              *outbox.Publisher
	AuthorizationVersionConsumer *authorizationversion.Consumer
}

// ServiceContext 是 Identity 的唯一依赖装配入口，由所有生成的 Logic 共享。
// 各类组件分别在 resources.go、token_components.go、sms_verifier.go、
// manager_wiring.go 和 messaging.go 中创建，这里只定义总装配和关闭顺序。
type ServiceContext struct {
	Config      config.Config
	Managers    Managers
	Security    Security
	Workers     Workers
	resources   serviceResources
	kafkaWriter *kafka.Writer
	kafkaReader *kafka.Reader
}

// NewServiceContext 按“基础资源 → Token 组件 → 短信校验器 → 业务 Manager → 消息任务”
// 的顺序完成装配。任一步失败都会关闭此前已经创建的基础资源。
func NewServiceContext(c config.Config) (*ServiceContext, error) {
	resourceSet, err := openResources(c)
	if err != nil {
		return nil, err
	}
	assembled := false
	defer func() {
		if !assembled {
			resourceSet.Close()
		}
	}()

	tokens, err := buildTokenComponents(c, resourceSet)
	if err != nil {
		return nil, err
	}
	verifier, err := buildSMSVerifier(c)
	if err != nil {
		return nil, err
	}
	managers, err := wireManagers(c, resourceSet, tokens, verifier)
	if err != nil {
		return nil, err
	}
	messagingRuntime, err := buildMessaging(c, resourceSet.identityStore, tokens.authorizationVersions)
	if err != nil {
		return nil, err
	}
	assembled = true

	return &ServiceContext{
		Config:   c,
		Managers: managers,
		Security: Security{
			Token:                tokens.tokenManager,
			AuthorizationVersion: tokens.authorizationVersionValidator,
		},
		Workers:     messagingRuntime.workers,
		resources:   resourceSet,
		kafkaWriter: messagingRuntime.writer,
		kafkaReader: messagingRuntime.reader,
	}, nil
}

// Close 先停止 Kafka 传输，再关闭 Redis 和 MySQL。
// 后台 Worker 必须由 main 在调用 Close 前先取消并等待退出。
func (s *ServiceContext) Close() error {
	if s.kafkaReader != nil {
		s.kafkaReader.Close()
	}
	if s.kafkaWriter != nil {
		s.kafkaWriter.Close()
	}
	return s.resources.Close()
}
