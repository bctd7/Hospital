package svc

import (
	"context"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"

	"hospital/common/authn"
	"hospital/common/authn/versionredis"
	"hospital/service/appointment/rpc/internal/config"
	"hospital/service/appointment/rpc/internal/manager"
	"hospital/service/appointment/rpc/internal/repository/mysqlstore"
)

type ServiceContext struct {
	Config                        config.Config
	StaffManager                  *manager.StaffManager
	PatientManager                *manager.PatientManager
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

	appointmentCache, err := manager.NewRedisCache(appointmentRedisClient, c.AppointmentRedis.Prefix)
	if err != nil {
		appointmentRedisClient.Close()
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create appointment query cache: %w", err)
	}
	staffManager, err := manager.NewStaffManager(store, store, appointmentCache)
	if err != nil {
		appointmentRedisClient.Close()
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create staff appointment manager: %w", err)
	}
	patientManager, err := manager.NewPatientManager(store, appointmentCache)
	if err != nil {
		appointmentRedisClient.Close()
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create patient appointment manager: %w", err)
	}

	return &ServiceContext{
		Config:                        c,
		StaffManager:                  staffManager,
		PatientManager:                patientManager,
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
