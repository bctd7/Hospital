package svc

import (
	"context"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"

	"hospital/common/authn"
	authversion "hospital/common/authz/version"
	"hospital/common/authz/version/redisstore"
	"hospital/service/appointment/rpc/internal/config"
	"hospital/service/appointment/rpc/internal/manager/common"
	patientmanager "hospital/service/appointment/rpc/internal/manager/patient"
	sharedmanager "hospital/service/appointment/rpc/internal/manager/shared"
	staffmanager "hospital/service/appointment/rpc/internal/manager/staff"
	"hospital/service/appointment/rpc/internal/repository/mysqlstore"
)

type ServiceContext struct {
	Config                        config.Config
	StaffManager                  *staffmanager.Manager
	PatientManager                *patientmanager.Manager
	SharedManager                 *sharedmanager.Manager
	TokenManager                  *authn.TokenManager
	AuthorizationVersionValidator *authversion.Validator
	AppointmentRedis              *redis.Client
	AppointmentRedisPrefix        string
	appointmentStore              *mysqlstore.Store
	bookingCache                  common.Cache
	authorizationRedisClient      *redis.Client
	bookingCleanupCancel          context.CancelFunc
	bookingCleanupDone            chan struct{}
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

	authorizationVersions, err := redisstore.NewStore(authorizationRedisClient, c.AuthorizationRedis.VersionPrefix)
	if err != nil {
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create appointment authorization version store: %w", err)
	}
	authorizationVersionValidator, err := authversion.NewValidator(authorizationVersions)
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

	appointmentCache, err := common.NewRedisCache(appointmentRedisClient, c.AppointmentRedis.Prefix)
	if err != nil {
		appointmentRedisClient.Close()
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create appointment query cache: %w", err)
	}
	staffManager, err := staffmanager.NewManager(store, store, store, appointmentCache)
	if err != nil {
		appointmentRedisClient.Close()
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create staff appointment manager: %w", err)
	}
	patientManager, err := patientmanager.NewManager(store, store, appointmentCache)
	if err != nil {
		appointmentRedisClient.Close()
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create patient appointment manager: %w", err)
	}
	sharedManager, err := sharedmanager.NewManager(store, appointmentCache)
	if err != nil {
		appointmentRedisClient.Close()
		authorizationRedisClient.Close()
		store.Close()
		return nil, fmt.Errorf("create shared appointment manager: %w", err)
	}

	serviceContext := &ServiceContext{
		Config:                        c,
		StaffManager:                  staffManager,
		PatientManager:                patientManager,
		SharedManager:                 sharedManager,
		TokenManager:                  tokenManager,
		AuthorizationVersionValidator: authorizationVersionValidator,
		AppointmentRedis:              appointmentRedisClient,
		AppointmentRedisPrefix:        c.AppointmentRedis.Prefix,
		appointmentStore:              store,
		bookingCache:                  appointmentCache,
		authorizationRedisClient:      authorizationRedisClient,
	}
	serviceContext.startBookingCleanup()
	return serviceContext, nil
}

func (s *ServiceContext) Close() error {
	if s.bookingCleanupCancel != nil {
		s.bookingCleanupCancel()
	}
	if s.bookingCleanupDone != nil {
		<-s.bookingCleanupDone
	}
	return errors.Join(
		s.appointmentStore.Close(),
		s.AppointmentRedis.Close(),
		s.authorizationRedisClient.Close(),
	)
}
