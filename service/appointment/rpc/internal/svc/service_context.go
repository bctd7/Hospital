package svc

import (
	"context"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"

	"hospital/common/authn"
	"hospital/common/authn/versionredis"
	"hospital/service/appointment/rpc/internal/catalog"
	"hospital/service/appointment/rpc/internal/config"
	"hospital/service/appointment/rpc/internal/repository/mysqlstore"
	"hospital/service/appointment/rpc/internal/resource"
)

type ServiceContext struct {
	Config                        config.Config
	CatalogManager                *catalog.Manager
	ResourceManager               *resource.Manager
	TokenManager                  *authn.TokenManager
	AuthorizationVersionValidator *authn.AuthorizationVersionValidator
	AppointmentRedis              *redis.Client
	AppointmentRedisPrefix        string
	appointmentStore              *mysqlstore.Store
	authorizationRedisClient      *redis.Client
}

func NewServiceContext(c config.Config) (*ServiceContext, error) {
	store, err := mysqlstore.New(c.MySQL.DataSource)
	if err != nil {
		return nil, err
	}

	authorizationRedisClient := redis.NewClient(&redis.Options{
		Addr: c.AuthorizationRedis.Addr, Password: c.AuthorizationRedis.Password, DB: c.AuthorizationRedis.DB,
	})
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := authorizationRedisClient.Ping(pingCtx).Err(); err != nil {
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("ping appointment authorization redis: %w", err)
	}

	authorizationVersions, err := versionredis.NewStore(authorizationRedisClient, c.AuthorizationRedis.VersionPrefix)
	if err != nil {
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create appointment authorization version store: %w", err)
	}
	authorizationVersionValidator, err := authn.NewAuthorizationVersionValidator(authorizationVersions)
	if err != nil {
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create appointment authorization version validator: %w", err)
	}

	appointmentRedisClient := redis.NewClient(&redis.Options{
		Addr: c.AppointmentRedis.Addr, Password: c.AppointmentRedis.Password, DB: c.AppointmentRedis.DB,
	})
	if err := appointmentRedisClient.Ping(pingCtx).Err(); err != nil {
		appointmentRedisClient.Close()
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("ping appointment business redis: %w", err)
	}

	publicKey, err := decodePublicKey(c.Token.AccessPublicKeyBase64)
	if err != nil {
		appointmentRedisClient.Close()
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("decode appointment access public key: %w", err)
	}
	tokenManager, err := authn.NewTokenManager(authn.TokenConfig{
		Issuer: c.Token.Issuer, Audience: c.Token.Audience,
		VerificationKey: publicKey, TTL: time.Duration(c.Token.AccessTTLSeconds) * time.Second,
	})
	if err != nil {
		appointmentRedisClient.Close()
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create appointment token manager: %w", err)
	}

	catalogManager, err := catalog.NewManager(store)
	if err != nil {
		appointmentRedisClient.Close()
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create examination catalog manager: %w", err)
	}
	resourceCache, err := resource.NewRedisCache(appointmentRedisClient, c.AppointmentRedis.Prefix)
	if err != nil {
		appointmentRedisClient.Close()
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create appointment resource cache: %w", err)
	}
	resourceManager, err := resource.NewManager(store, resourceCache)
	if err != nil {
		appointmentRedisClient.Close()
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create appointment resource manager: %w", err)
	}

	return &ServiceContext{
		Config:                        c,
		CatalogManager:                catalogManager,
		ResourceManager:               resourceManager,
		TokenManager:                  tokenManager,
		AuthorizationVersionValidator: authorizationVersionValidator,
		AppointmentRedis:              appointmentRedisClient,
		AppointmentRedisPrefix:        c.AppointmentRedis.Prefix,
		appointmentStore:              store,
		authorizationRedisClient:      authorizationRedisClient,
	}, nil
}

func (s *ServiceContext) Close() error {
	return errors.Join(
		s.appointmentStore.Close(),
		s.AppointmentRedis.Close(),
		s.authorizationRedisClient.Close(),
	)
}
