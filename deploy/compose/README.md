# Local Infrastructure

本目录提供本地开发环境，不代表生产级高可用部署。

## 组件

| 组件 | 镜像 | 主机地址 | 用途 |
|---|---|---|---|
| MySQL | `mysql:8.4.11` | `127.0.0.1:3306` | 业务事实库；首期包含独立 `hospital_identity` 数据库和账号 |
| Redis | `redis:7.4.10-alpine` | `127.0.0.1:6379` | Refresh Session 和授权版本投影 |
| Kafka | `apache/kafka:4.2.0` | `127.0.0.1:9092` | 可选的异步事件系统 |

Kafka 使用单节点 KRaft combined mode，仅用于本地开发、集成测试和功能验证。容器内客户端使用 `kafka:29092`，运行在 Windows 主机上的 Go 服务使用 `127.0.0.1:9092`。

## 启动

```powershell
Copy-Item .env.example .env

docker compose `
  --env-file .env `
  -f deploy/compose/docker-compose.yml `
  up -d mysql redis
```

MySQL 初始化脚本只会在数据卷第一次创建时运行。已有 `mysql-data` 卷不会自动重放 Identity 初始迁移；后续数据库升级必须使用正式迁移工具，不通过删除数据卷模拟升级。

需要 Kafka 时：

```powershell
docker compose `
  --env-file .env `
  -f deploy/compose/docker-compose.yml `
  --profile messaging `
  up -d
```

## 停止

```powershell
docker compose `
  --env-file .env `
  -f deploy/compose/docker-compose.yml `
  --profile messaging `
  down
```

`down` 不会删除命名卷。不要随意使用 `down -v`，它会删除本地数据库、Redis 和 Kafka 数据。

## 安全边界

- 所有端口只绑定 `127.0.0.1`；
- 默认密码只用于本地开发；
- 正式部署不能使用此单节点 Kafka 配置；
- 正式环境应使用独立 Secret、备份、TLS、监控和访问控制。
