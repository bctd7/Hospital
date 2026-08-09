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
	"hospital/service/identity/rpc/internal/authorization"
	"hospital/service/identity/rpc/internal/config"
	"hospital/service/identity/rpc/internal/repository"
	"hospital/service/identity/rpc/internal/session"
)

type ServiceContext struct {
	Config               config.Config
	AuthorizationManager *authorization.Manager
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
	privateKey, err := decodePrivateKey(c.Auth.AccessPrivateKeyBase64)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("decode identity access private key: %w", err)
	}
	publicKey, err := decodePublicKey(c.Auth.AccessPublicKeyBase64)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("decode identity access public key: %w", err)
	}
	tokenManager, err := authn.NewTokenManager(authn.TokenConfig{
		Issuer: c.Auth.Issuer, Audience: c.Auth.Audience,
		SigningKey: privateKey, VerificationKey: publicKey,
		TTL: time.Duration(c.Auth.AccessTTLSeconds) * time.Second,
	})
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("create identity token manager: %w", err)
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr: c.Redis.Addr, Password: c.Redis.Password, DB: c.Redis.DB,
	})
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		redisClient.Close()
		store.Close()
		return nil, fmt.Errorf("ping identity redis: %w", err)
	}
	sessionStore, err := repository.NewRedisSessionStore(redisClient, c.Redis.Prefix)
	if err != nil {
		redisClient.Close()
		store.Close()
		return nil, err
	}
	sessionManager, err := session.NewManager(
		sessionStore, store, tokenManager, time.Duration(c.Auth.RefreshTTLSeconds)*time.Second,
	)
	if err != nil {
		redisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create identity session manager: %w", err)
	}
	return &ServiceContext{
		Config: c, identityStore: store, redisClient: redisClient, TokenManager: tokenManager,
		AuthorizationManager: authorization.NewManager(store),
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
