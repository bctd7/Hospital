package svc

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"

	"hospital/common/authn"
	"hospital/common/authn/versionredis"
	contractevents "hospital/contracts/events"
	"hospital/service/identity/rpc/internal/account"
	"hospital/service/identity/rpc/internal/authorization"
	"hospital/service/identity/rpc/internal/authorizationprojection"
	"hospital/service/identity/rpc/internal/config"
	"hospital/service/identity/rpc/internal/identityadmin"
	"hospital/service/identity/rpc/internal/login"
	"hospital/service/identity/rpc/internal/messaging/kafkaconsumer"
	"hospital/service/identity/rpc/internal/messaging/kafkaproducer"
	"hospital/service/identity/rpc/internal/organization"
	"hospital/service/identity/rpc/internal/outbox"
	"hospital/service/identity/rpc/internal/repository"
	"hospital/service/identity/rpc/internal/repository/mysqlstore"
	"hospital/service/identity/rpc/internal/session"
)

type ServiceContext struct {
	Config                        config.Config
	AuthorizationManager          *authorization.Manager
	AccountManager                *account.Manager
	PhoneLoginManager             *account.PhoneLoginManager
	SessionManager                *session.Manager
	TokenManager                  *authn.TokenManager
	AuthorizationVersionValidator *authn.AuthorizationVersionValidator
	identityStore                 *mysqlstore.Store
	redisClient                   *redis.Client
	kafkaProducer                 *kafkaproducer.Producer
	OutboxPublisher               *outbox.Publisher
	AuthorizationVersionConsumer  *kafkaconsumer.Consumer
	AuthorizationVersionProjector *authorizationprojection.Projector
	OrganizationManager           *organization.Manager
	IdentityAdminManager          *identityadmin.Manager
}

func NewServiceContext(c config.Config) (*ServiceContext, error) {
	phoneProvider, err := phoneVerificationProvider(c)
	if err != nil {
		return nil, err
	}
	store, err := mysqlstore.New(c.MySQL.DataSource)
	if err != nil {
		return nil, err
	}
	privateKey, err := decodePrivateKey(c.Token.AccessPrivateKeyBase64)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("decode identity access private key: %w", err)
	}
	publicKey, err := decodePublicKey(c.Token.AccessPublicKeyBase64)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("decode identity access public key: %w", err)
	}
	tokenManager, err := authn.NewTokenManager(authn.TokenConfig{
		Issuer: c.Token.Issuer, Audience: c.Token.Audience,
		SigningKey: privateKey, VerificationKey: publicKey,
		TTL: time.Duration(c.Token.AccessTTLSeconds) * time.Second,
	})
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("create identity token manager: %w", err)
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr: c.SessionRedis.Addr, Password: c.SessionRedis.Password, DB: c.SessionRedis.DB,
	})
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		redisClient.Close()
		store.Close()
		return nil, fmt.Errorf("ping identity redis: %w", err)
	}
	sessionStore, err := repository.NewRedisSessionStore(redisClient, c.SessionRedis.Prefix)
	if err != nil {
		redisClient.Close()
		store.Close()
		return nil, err
	}
	authorizationVersions, err := versionredis.NewStore(redisClient, c.SessionRedis.AuthorizationVersionPrefix)
	if err != nil {
		redisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create authorization version store: %w", err)
	}
	authorizationVersionValidator, err := authn.NewAuthorizationVersionValidator(authorizationVersions)
	if err != nil {
		redisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create authorization version validator: %w", err)
	}
	sessionManager, err := session.NewManager(
		sessionStore, store, tokenManager, authorizationVersions,
		time.Duration(c.Token.RefreshTTLSeconds)*time.Second,
	)
	if err != nil {
		redisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create identity session manager: %w", err)
	}
	phoneLookupKey, err := base64.StdEncoding.DecodeString(c.PhoneLookupKeyBase64)
	if err != nil {
		redisClient.Close()
		store.Close()
		return nil, fmt.Errorf("decode identity phone lookup key: %w", err)
	}
	accountManager, err := account.NewManager(
		store,
		login.NewWeChatClient(c.WeChat.AppID, c.WeChat.AppSecret, c.WeChat.Code2SessionURL, nil),
		sessionManager,
		c.WeChat.AppID,
		phoneLookupKey,
	)
	if err != nil {
		redisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create identity account manager: %w", err)
	}

	organizationManager := organization.NewManager(store)

	phoneLoginManager, err := account.NewPhoneLoginManager(
		store, phoneProvider, sessionManager, phoneLookupKey, c.PhoneLogin.IntervalSeconds,
	)
	if err != nil {
		redisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create phone login manager: %w", err)
	}
	authorizationManager, err := authorization.NewManager(store)
	if err != nil {
		redisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create identity authorization manager: %w", err)
	}
	identityAdminManager, err := identityadmin.NewManager(store, phoneLookupKey)
	if err != nil {
		redisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create identity admin manager: %w", err)
	}

	var kafkaProducer *kafkaproducer.Producer
	var kafkaConsumer *kafkaconsumer.Consumer
	var outboxPublisher *outbox.Publisher
	var authorizationVersionProjector *authorizationprojection.Projector
	if c.Kafka.Enabled {
		kafkaCtx, kafkaCancel := context.WithTimeout(context.Background(), 5*time.Second)
		kafkaProducer, err = kafkaproducer.New(
			kafkaCtx,
			splitCommaSeparated(c.Kafka.Brokers),
			c.Kafka.ClientID,
		)
		kafkaCancel()
		if err != nil {
			redisClient.Close()
			store.Close()
			return nil, fmt.Errorf("create identity kafka producer: %w", err)
		}

		outboxPublisher, err = outbox.NewPublisher(
			store,
			kafkaProducer,
			c.Kafka.AuthorizationChangedTopic,
			contractevents.EventTypeIdentityAuthorizationChangedV1,
			c.Kafka.ClientID,
			c.Kafka.BatchSize,
		)
		if err != nil {
			kafkaProducer.Close()
			redisClient.Close()
			store.Close()
			return nil, fmt.Errorf("create identity outbox publisher: %w", err)
		}

		authorizationVersionProjector, err = authorizationprojection.NewProjector(authorizationVersions)
		if err != nil {
			kafkaProducer.Close()
			redisClient.Close()
			store.Close()
			return nil, fmt.Errorf("create authorization version projector: %w", err)
		}

		consumerCtx, consumerCancel := context.WithTimeout(context.Background(), 5*time.Second)
		kafkaConsumer, err = kafkaconsumer.New(
			consumerCtx,
			splitCommaSeparated(c.Kafka.Brokers),
			c.Kafka.ClientID,
			c.Kafka.AuthorizationChangedTopic,
			c.Kafka.AuthorizationVersionConsumerGroup,
		)
		consumerCancel()
		if err != nil {
			kafkaProducer.Close()
			redisClient.Close()
			store.Close()
			return nil, fmt.Errorf("create identity kafka consumer: %w", err)
		}
	}

	return &ServiceContext{
		Config: c, identityStore: store, redisClient: redisClient,
		kafkaProducer:                 kafkaProducer,
		TokenManager:                  tokenManager,
		AuthorizationVersionValidator: authorizationVersionValidator,
		AuthorizationManager:          authorizationManager,
		AccountManager:                accountManager,
		PhoneLoginManager:             phoneLoginManager,
		SessionManager:                sessionManager,
		OrganizationManager:           organizationManager,
		IdentityAdminManager:          identityAdminManager,
		OutboxPublisher:               outboxPublisher,
		AuthorizationVersionConsumer:  kafkaConsumer,
		AuthorizationVersionProjector: authorizationVersionProjector,
	}, nil
}

func phoneVerificationProvider(c config.Config) (login.PhoneVerificationProvider, error) {
	provider := strings.ToLower(strings.TrimSpace(c.PhoneLogin.Provider))
	if provider == "" {
		if c.PhoneLogin.Enabled {
			provider = "aliyun"
		} else {
			provider = "disabled"
		}
	}

	switch provider {
	case "disabled":
		return login.UnconfiguredPhoneVerificationProvider{}, nil
	case "aliyun":
		value, err := login.NewAlibabaPNVS(login.AlibabaPNVSConfig{
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
		value, err := login.NewLocalPhoneVerificationProvider(c.PhoneLogin.LocalCode)
		if err != nil {
			return nil, fmt.Errorf("create local phone login provider: %w", err)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported phone login provider %q", provider)
	}
}

func decodePrivateKey(value string) (ed25519.PrivateKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	return ed25519.PrivateKey(decoded), err
}

func decodePublicKey(value string) (ed25519.PublicKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	return ed25519.PublicKey(decoded), err
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func (s *ServiceContext) Close() error {
	if s.AuthorizationVersionConsumer != nil {
		s.AuthorizationVersionConsumer.Close()
	}
	if s.kafkaProducer != nil {
		s.kafkaProducer.Close()
	}
	return errors.Join(s.identityStore.Close(), s.redisClient.Close())
}
