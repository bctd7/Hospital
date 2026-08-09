# Identity Service

Identity Service 是账号身份、工作人员科室、角色、权限和登录会话的数据拥有者。普通业务服务使用 `common/authn` 和 `common/authz` 本地校验 Access Token；只有登录会话和敏感授权变更需要调用 Identity。

## 当前能力

- Ed25519 JWT Access Token 签发与 gRPC 本地验证；
- Redis Refresh Session 创建、刷新轮换和注销；
- 刷新时从 MySQL 重新读取账号最新状态和权限；
- 旧 Refresh Token 重放检测和 Session 撤销；
- 查询授权上下文，以及首期角色、科室和账号状态管理；
- 权限变更幂等、审计和 Outbox；
- 微信登录、微信手机号和短信验证码供应商接口预留。

Refresh Token 原文不会写入 Redis、数据库或日志。Redis 仅保存 SHA-256 哈希，默认绝对有效期为 30 天；Access Token 默认有效期为 15 分钟。

## 本地运行

根据根目录 `.env.example` 创建 `.env`，启动 MySQL 和 Redis：

```powershell
docker compose `
  --env-file .env `
  -f deploy/compose/docker-compose.yml `
  up -d mysql redis
```

生成本地 Ed25519 密钥，并将结果写入本地 `.env`：

```powershell
go run ./tools/identity-keygen
```

将 `.env` 中的 MySQL、Redis 和密钥变量注入当前终端后启动 RPC：

```powershell
go run ./service/identity/rpc -f service/identity/rpc/etc/identity-rpc.yaml
```

随后启动 `app-api`，对外提供：

- `POST /api/v1/auth/token/refresh`
- `POST /api/v1/auth/token/revoke`

完整登录与会话设计见 `docs/identity-authentication.md`。

## 尚未开放

- 微信和短信真实登录；
- Refresh Token 首次签发的公共入口——未来登录供应商验证成功后调用 `SessionManager.Start`；
- 多设备会话列表与一键退出全部设备；
- 普通 Access Token 的即时撤销；
- 多级权限审批、临时授权和多科室任职。
