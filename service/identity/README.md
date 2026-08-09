# Identity Service

Identity Service 是账号身份、工作人员科室、角色、权限和登录会话的数据拥有者。普通业务服务使用 `common/authn` 和 `common/authz` 本地校验 Access Token；只有登录会话和敏感授权变更需要调用 Identity。

## 当前能力

- Ed25519 JWT Access Token 签发与 gRPC 本地验证；
- Redis Refresh Session 创建、刷新轮换和注销；
- 刷新时从 MySQL 重新读取账号最新状态和权限；
- 旧 Refresh Token 重放检测和 Session 撤销；
- 查询授权上下文，以及首期角色、科室和账号状态管理；
- 权限变更幂等、审计和 Outbox；
- 阿里云 PNVS 手机号验证码发送、校验、自动注册和登录；
- 微信登录兼容接口，以及微信手机号供应商接口预留。

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

启用真实手机号登录前，在阿里云号码认证控制台开通“短信认证服务”，从当前可用列表复制系统
签名和登录/注册模板 Code，填写 `.env` 中的 PNVS 配置，并把
`service/identity/rpc/etc/identity-rpc.yaml` 的 `PhoneLogin.Enabled` 改为 `true`。AccessKey 应来自
最小权限 RAM 用户，不使用主账号 AccessKey。

随后启动 `app-api`，对外提供：

- `POST /api/v1/auth/token/refresh`
- `POST /api/v1/auth/token/revoke`

完整登录与会话设计见 `docs/identity-authentication.md`。

## 尚未开放

- 多设备会话列表与一键退出全部设备；
- 普通 Access Token 的即时撤销；
- 多级权限审批、临时授权和多科室任职。

## 手机号注册、登录与医生开通

当前主登录方案使用阿里云 PNVS 短信认证：

- `POST /api/v1/auth/phone/code` 使用平台预置签名和模板发送动态验证码；
- `POST /api/v1/auth/phone/login` 校验验证码，同一手机号复用账号，首次登录自动创建患者账号；
- 手机号绑定保存为 `verified/sms`，数据库只保存 HMAC 指纹和脱敏号码；
- `GET /api/v1/auth/me` 返回当前 Token 中的账号类型、角色、科室和权限；
- 超级管理员可以按完整手机号精确查找账号，并开通为指定科室医生；
- `POST /api/v1/auth/wechat/login` 暂时保留为兼容接口，新小程序入口不再调用。

手机号是唯一登录标识，但内部 `account_id` 仍是不可变数据库主键。完整规则见
`plan/miniapp/07-phone-primary-authentication.md`。
