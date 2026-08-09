package svc

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"

	"hospital/common/authn"
	"hospital/service/identity/rpc/internal/account"
	"hospital/service/identity/rpc/internal/authorization"
	"hospital/service/identity/rpc/internal/config"
	"hospital/service/identity/rpc/internal/login"
	"hospital/service/identity/rpc/internal/repository"
	"hospital/service/identity/rpc/internal/session"
)

type ServiceContext struct {
	Config               config.Config
	AuthorizationManager *authorization.Manager
	AccountManager       *account.Manager
	SessionManager       *session.Manager
	TokenManager         *authn.TokenManager
	identityStore        *repository.MySQLStore
	redisClient          *redis.Client
}

func NewServiceContext(c config.Config) (*ServiceContext, error) {
	store, err := repository.NewMySQLStore(c.MySQL.DataSource)
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
	sessionManager, err := session.NewManager(
		sessionStore, store, tokenManager, time.Duration(c.Token.RefreshTTLSeconds)*time.Second,
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
	return &ServiceContext{
		Config: c, identityStore: store, redisClient: redisClient, TokenManager: tokenManager,
		AuthorizationManager: authorization.NewManager(store),
		AccountManager:       accountManager,
		SessionManager:       sessionManager,
	}, nil
}

func decodePrivateKey(value string) (ed25519.PrivateKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	return ed25519.PrivateKey(decoded), err
}

func decodePublicKey(value string) (ed25519.PublicKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	return ed25519.PublicKey(decoded), err
}

func (s *ServiceContext) Close() error {
	return errors.Join(s.identityStore.Close(), s.redisClient.Close())
}
