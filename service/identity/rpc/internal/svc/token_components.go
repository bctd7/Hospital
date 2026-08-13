package svc

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"

	"hospital/common/authn"
	commonauthversion "hospital/common/authz/version"
	"hospital/common/authz/version/redisstore"
	"hospital/service/identity/rpc/internal/config"
	"hospital/service/identity/rpc/internal/repository"
)

// tokenComponents 是 Token、授权版本和 Refresh Session 所需的底层组件。
// 它只保存装配结果，不包含登录渠道选择或账号业务规则。
type tokenComponents struct {
	tokenManager                  *authn.TokenManager
	authorizationVersions         *redisstore.Store
	authorizationVersionValidator *commonauthversion.Validator
	refreshSessions               *repository.RedisSessionStore
	phoneLookupKey                []byte
}

// buildTokenComponents 解码服务端密钥，并组装 JWT、授权版本校验和 Refresh Session Store。
// Session Manager 本身属于业务入口，在 manager_wiring.go 中创建。
func buildTokenComponents(c config.Config, resourceSet serviceResources) (tokenComponents, error) {
	privateKey, err := decodePrivateKey(c.Token.AccessPrivateKeyBase64)
	if err != nil {
		return tokenComponents{}, fmt.Errorf("decode identity access private key: %w", err)
	}
	publicKey, err := decodePublicKey(c.Token.AccessPublicKeyBase64)
	if err != nil {
		return tokenComponents{}, fmt.Errorf("decode identity access public key: %w", err)
	}
	tokenManager, err := authn.NewTokenManager(authn.TokenConfig{
		Issuer:          c.Token.Issuer,
		Audience:        c.Token.Audience,
		SigningKey:      privateKey,
		VerificationKey: publicKey,
		TTL:             time.Duration(c.Token.AccessTTLSeconds) * time.Second,
	})
	if err != nil {
		return tokenComponents{}, fmt.Errorf("create identity token manager: %w", err)
	}

	refreshSessions, err := repository.NewRedisSessionStore(resourceSet.redisClient, c.SessionRedis.Prefix)
	if err != nil {
		return tokenComponents{}, err
	}
	authorizationVersions, err := redisstore.NewStore(resourceSet.redisClient, c.SessionRedis.AuthorizationVersionPrefix)
	if err != nil {
		return tokenComponents{}, fmt.Errorf("create authorization version store: %w", err)
	}
	authorizationVersionValidator, err := commonauthversion.NewValidator(authorizationVersions)
	if err != nil {
		return tokenComponents{}, fmt.Errorf("create authorization version validator: %w", err)
	}
	phoneLookupKey, err := base64.StdEncoding.DecodeString(c.PhoneLookupKeyBase64)
	if err != nil {
		return tokenComponents{}, fmt.Errorf("decode identity phone lookup key: %w", err)
	}

	return tokenComponents{
		tokenManager:                  tokenManager,
		authorizationVersions:         authorizationVersions,
		authorizationVersionValidator: authorizationVersionValidator,
		refreshSessions:               refreshSessions,
		phoneLookupKey:                phoneLookupKey,
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
