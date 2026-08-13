package svc

import (
	"context"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"

	"hospital/service/identity/rpc/internal/config"
	"hospital/service/identity/rpc/internal/repository/mysqlstore"
)

// serviceResources 保存 Identity 进程独占且需要显式关闭的基础资源。
// 业务 Manager 不直接创建连接，只接收这里已经初始化完成的 Store 或 Client。
type serviceResources struct {
	identityStore *mysqlstore.Store
	redisClient   *redis.Client
}

// openResources 按 MySQL、Redis 的顺序建立连接。
// Redis 在启动阶段执行一次 Ping，避免服务启动成功后才暴露配置或网络错误。
func openResources(c config.Config) (serviceResources, error) {
	identityStore, err := mysqlstore.New(c.MySQL.DataSource)
	if err != nil {
		return serviceResources{}, err
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     c.SessionRedis.Addr,
		Password: c.SessionRedis.Password,
		DB:       c.SessionRedis.DB,
	})
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		redisClient.Close()
		identityStore.Close()
		return serviceResources{}, fmt.Errorf("ping identity redis: %w", err)
	}

	return serviceResources{identityStore: identityStore, redisClient: redisClient}, nil
}

// Close 按照与创建相反的顺序关闭资源，并保留所有关闭错误。
func (r serviceResources) Close() error {
	var redisErr error
	if r.redisClient != nil {
		redisErr = r.redisClient.Close()
	}
	var mysqlErr error
	if r.identityStore != nil {
		mysqlErr = r.identityStore.Close()
	}
	return errors.Join(redisErr, mysqlErr)
}
