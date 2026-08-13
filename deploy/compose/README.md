# 本地基础设施

本目录提供本地开发环境，不代表生产级高可用部署。

## 组件

| 组件 | 镜像 | 主机地址 | 用途 |
|---|---|---|---|
| MySQL | `mysql:8.4.11` | `127.0.0.1:3306` | 业务事实库；首期包含独立 `hospital_identity` 数据库和账号 |
| Redis | `redis:7.4.10-alpine` | `127.0.0.1:6379` | Refresh Session 和授权版本同步状态 |
| Kafka | `apache/kafka:4.2.0` | `127.0.0.1:9092` | 传递 Identity Outbox 授权版本事件 |

Kafka 使用单节点 KRaft combined mode，仅用于本地开发、集成测试和功能验证。容器内客户端使用 `kafka:29092`，运行在 Windows 主机上的 Go 服务使用 `127.0.0.1:9092`。

## 启动

```powershell
Copy-Item .env.example .env

docker compose `
  --env-file .env `
  -f deploy/compose/docker-compose.yml `
  --profile messaging `
  up -d
```

推荐直接从仓库根目录执行：

```powershell
.\scripts\db-bootstrap-local.ps1
```

该脚本会启动 MySQL、幂等创建当前服务数据库和账号，并只执行尚未运行的迁移，不会删除已有数据。
MySQL 初始化脚本仍只会在数据卷第一次创建时运行，表结构不再依赖初始化目录中的 SQL 挂载。
旧数据库第一次接入版本管理时，按 `scripts/README.md` 完成一次基线登记。

`.env.example` 默认设置 `IDENTITY_KAFKA_ENABLED=true`，因此标准 Identity 开发环境同时启动 Kafka。只有明确
关闭 Kafka 授权版本同步、只调试不涉及授权变化的局部功能时，才使用精简启动：

```powershell
docker compose `
  --env-file .env `
  -f deploy/compose/docker-compose.yml `
  up -d mysql redis
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
