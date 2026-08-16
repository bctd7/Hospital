package svc

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/zrpc"

	"hospital/common/authn"
	authversion "hospital/common/authz/version"
	"hospital/common/authz/version/redisstore"
	"hospital/service/appointment/rpc/appointmentservice"
	"hospital/service/guidance/rpc/internal/config"
	"hospital/service/guidance/rpc/internal/integration/amap"
	"hospital/service/guidance/rpc/internal/integration/appointmentcatalog"
	"hospital/service/guidance/rpc/internal/repository/mysqlstore"
	"hospital/service/guidance/rpc/internal/routing"
	precedencemanager "hospital/service/guidance/rpc/internal/rules/precedence/manager"
)

type ServiceContext struct {
	Config                        config.Config
	PrecedenceManager             *precedencemanager.Manager
	PlaceFinder                   *routing.PlaceFinder
	RouteCalculator               *routing.Calculator
	TokenManager                  *authn.TokenManager
	AuthorizationVersionValidator *authversion.Validator
	store                         *mysqlstore.Store
	authorizationRedis            *redis.Client
}

func NewServiceContext(c config.Config) (*ServiceContext, error) {
	store, err := mysqlstore.New(c.MySQL.DataSource)
	if err != nil {
		return nil, err
	}
	assembled := false
	var authorizationRedis *redis.Client
	defer func() {
		if assembled {
			return
		}
		if authorizationRedis != nil {
			authorizationRedis.Close()
		}
		store.Close()
	}()

	publicKey, err := base64.StdEncoding.DecodeString(c.Token.AccessPublicKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode guidance access public key: %w", err)
	}
	tokenManager, err := authn.NewTokenManager(authn.TokenConfig{
		Issuer: c.Token.Issuer, Audience: c.Token.Audience,
		VerificationKey: ed25519.PublicKey(publicKey),
		TTL:             time.Duration(c.Token.AccessTTLSeconds) * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("create guidance token verifier: %w", err)
	}
	authorizationRedis = redis.NewClient(&redis.Options{
		Addr: c.AuthorizationRedis.Addr, Password: c.AuthorizationRedis.Password, DB: c.AuthorizationRedis.DB,
	})
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := authorizationRedis.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("ping guidance authorization redis: %w", err)
	}
	versions, err := redisstore.NewStore(authorizationRedis, c.AuthorizationRedis.VersionPrefix)
	if err != nil {
		return nil, fmt.Errorf("create guidance authorization version store: %w", err)
	}
	validator, err := authversion.NewValidator(versions)
	if err != nil {
		return nil, fmt.Errorf("create guidance authorization version validator: %w", err)
	}
	appointmentClient := appointmentservice.NewAppointmentService(zrpc.MustNewClient(c.AppointmentRPC))
	directory, err := appointmentcatalog.New(appointmentClient)
	if err != nil {
		return nil, err
	}
	manager, err := precedencemanager.New(store, directory)
	if err != nil {
		return nil, err
	}
	mapClient := amap.New(amap.Config{
		PlaceSearchEndpoint: c.AMap.PlaceSearchEndpoint, WalkingEndpoint: c.AMap.WalkingEndpoint,
		WebServiceKey: c.AMap.WebServiceKey, Timeout: time.Duration(c.AMap.TimeoutMilliseconds) * time.Millisecond,
	})
	placeFinder, err := routing.NewPlaceFinder(mapClient)
	if err != nil {
		return nil, fmt.Errorf("create guidance place finder: %w", err)
	}
	routeCalculator, err := routing.NewCalculator(mapClient)
	if err != nil {
		return nil, fmt.Errorf("create guidance route calculator: %w", err)
	}
	assembled = true
	return &ServiceContext{
		Config: c, PrecedenceManager: manager, PlaceFinder: placeFinder, RouteCalculator: routeCalculator,
		TokenManager: tokenManager, AuthorizationVersionValidator: validator,
		store: store, authorizationRedis: authorizationRedis,
	}, nil
}

func (s *ServiceContext) Close() error {
	return errors.Join(s.authorizationRedis.Close(), s.store.Close())
}
