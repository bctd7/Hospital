package svc

import (
	"errors"

	redis "github.com/redis/go-redis/v9"

	"hospital/common/authn"
	commonauthversion "hospital/common/authz/version"
	accountmanager "hospital/service/identity/rpc/internal/account/manager"
	"hospital/service/identity/rpc/internal/authentication"
	authorizationmanager "hospital/service/identity/rpc/internal/authorization/manager"
	identityauthversion "hospital/service/identity/rpc/internal/authorization/version"
	"hospital/service/identity/rpc/internal/config"
	"hospital/service/identity/rpc/internal/messaging/kafka"
	"hospital/service/identity/rpc/internal/messaging/outbox"
	organizationmanager "hospital/service/identity/rpc/internal/organization/manager"
	"hospital/service/identity/rpc/internal/repository/mysqlstore"
	"hospital/service/identity/rpc/internal/session"
)

// Managers groups business entry points exposed to RPC Logic. ServiceContext
// owns their lifetime but contains no account, authorization, or session rules.
type Managers struct {
	Authentication        *authentication.Manager
	PhoneLogin            *authentication.PhoneLoginManager
	Session               *session.Manager
	Authorization         *authorizationmanager.Manager
	Account               *accountmanager.Manager
	OrganizationUnit      *organizationmanager.UnitManager
	OrganizationDirectory *organizationmanager.DirectoryManager
}

// Security groups token verification and runtime authorization-version checks.
type Security struct {
	Token                *authn.TokenManager
	AuthorizationVersion *commonauthversion.Validator
}

// Workers groups optional background message processors.
type Workers struct {
	OutboxPublisher              *outbox.Publisher
	AuthorizationVersionConsumer *identityauthversion.Consumer
}

// ServiceContext is the Identity composition root shared by generated Logic.
// Construction details are split into security.go, managers.go, and messaging.go.
type ServiceContext struct {
	Config        config.Config
	Managers      Managers
	Security      Security
	Workers       Workers
	identityStore *mysqlstore.Store
	redisClient   *redis.Client
	kafkaWriter   *kafka.Writer
	kafkaReader   *kafka.Reader
}

func NewServiceContext(c config.Config) (*ServiceContext, error) {
	store, err := mysqlstore.New(c.MySQL.DataSource)
	if err != nil {
		return nil, err
	}

	securityRuntime, err := buildSecurity(c, store)
	if err != nil {
		store.Close()
		return nil, err
	}
	managers, err := buildManagers(c, store, securityRuntime.sessions, securityRuntime.phoneLookupKey)
	if err != nil {
		securityRuntime.redis.Close()
		store.Close()
		return nil, err
	}
	messagingRuntime, err := buildMessaging(c, store, securityRuntime.versions)
	if err != nil {
		securityRuntime.redis.Close()
		store.Close()
		return nil, err
	}

	return &ServiceContext{
		Config: c, Managers: managers, Security: securityRuntime.security,
		Workers: messagingRuntime.workers, identityStore: store,
		redisClient: securityRuntime.redis, kafkaWriter: messagingRuntime.writer,
		kafkaReader: messagingRuntime.reader,
	}, nil
}

func (s *ServiceContext) Close() error {
	if s.kafkaReader != nil {
		s.kafkaReader.Close()
	}
	if s.kafkaWriter != nil {
		s.kafkaWriter.Close()
	}
	return errors.Join(s.identityStore.Close(), s.redisClient.Close())
}
