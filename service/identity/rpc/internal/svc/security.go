package svc

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"

	"hospital/common/authn"
	commonauthversion "hospital/common/authz/version"
	"hospital/common/authz/version/redisstore"
	"hospital/service/identity/rpc/internal/config"
	"hospital/service/identity/rpc/internal/repository"
	"hospital/service/identity/rpc/internal/repository/mysqlstore"
	"hospital/service/identity/rpc/internal/session"
)

type securityRuntime struct {
	security       Security
	redis          *redis.Client
	versions       *redisstore.Store
	sessions       *session.Manager
	phoneLookupKey []byte
}

func buildSecurity(c config.Config, store *mysqlstore.Store) (securityRuntime, error) {
	privateKey, err := decodePrivateKey(c.Token.AccessPrivateKeyBase64)
	if err != nil {
		return securityRuntime{}, fmt.Errorf("decode identity access private key: %w", err)
	}
	publicKey, err := decodePublicKey(c.Token.AccessPublicKeyBase64)
	if err != nil {
		return securityRuntime{}, fmt.Errorf("decode identity access public key: %w", err)
	}
	tokenManager, err := authn.NewTokenManager(authn.TokenConfig{
		Issuer: c.Token.Issuer, Audience: c.Token.Audience,
		SigningKey: privateKey, VerificationKey: publicKey,
		TTL: time.Duration(c.Token.AccessTTLSeconds) * time.Second,
	})
	if err != nil {
		return securityRuntime{}, fmt.Errorf("create identity token manager: %w", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: c.SessionRedis.Addr, Password: c.SessionRedis.Password, DB: c.SessionRedis.DB,
	})
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		redisClient.Close()
		return securityRuntime{}, fmt.Errorf("ping identity redis: %w", err)
	}
	sessionStore, err := repository.NewRedisSessionStore(redisClient, c.SessionRedis.Prefix)
	if err != nil {
		redisClient.Close()
		return securityRuntime{}, err
	}
	versions, err := redisstore.NewStore(redisClient, c.SessionRedis.AuthorizationVersionPrefix)
	if err != nil {
		redisClient.Close()
		return securityRuntime{}, fmt.Errorf("create authorization version store: %w", err)
	}
	validator, err := commonauthversion.NewValidator(versions)
	if err != nil {
		redisClient.Close()
		return securityRuntime{}, fmt.Errorf("create authorization version validator: %w", err)
	}
	sessions, err := session.NewManager(
		sessionStore, store, tokenManager, versions,
		time.Duration(c.Token.RefreshTTLSeconds)*time.Second,
	)
	if err != nil {
		redisClient.Close()
		return securityRuntime{}, fmt.Errorf("create identity session manager: %w", err)
	}
	phoneLookupKey, err := base64.StdEncoding.DecodeString(c.PhoneLookupKeyBase64)
	if err != nil {
		redisClient.Close()
		return securityRuntime{}, fmt.Errorf("decode identity phone lookup key: %w", err)
	}
	return securityRuntime{
		security: Security{Token: tokenManager, AuthorizationVersion: validator},
		redis:    redisClient, versions: versions, sessions: sessions,
		phoneLookupKey: phoneLookupKey,
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
