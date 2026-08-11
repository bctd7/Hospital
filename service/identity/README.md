# Identity Service

Identity Service 是账号、登录会话、角色权限、医院组织和医生档案的数据拥有者。首期 Identity 开发已经结束，
后续业务服务通过 Access Token、Identity RPC 和授权事件使用这些能力，不得直接访问 Identity 数据表。

## 已实现能力

### 认证与会话

- 阿里云 PNVS 手机验证码发送与校验；
- 首次手机号登录自动创建患者账号；
- Ed25519 JWT Access Token；
- Redis Refresh Session、Token 轮换、重放检测和注销；
- Redis 授权版本校验，账号、角色或科室权限变化后旧 Token 失效；
- 受限的本地固定验证码 Provider，仅允许 `local/test` 环境。

### 组织与公共目录

- 单医院下的院区、科室层级；
- 医院/院区上下文、按院区查询科室、按科室查询医生；
- 管理员创建、查询、修改、停用和恢复院区/科室；
- 有效子节点和有效医生保护，不做物理级联删除。

### 账号与医生管理

- 管理员账号分页、详情和完整手机号精确搜索；
- 账号停用与恢复；
- 患者账号开通医生、编辑医生资料、调岗和撤销医生身份；
- 普通用户维护自己的展示昵称；
- `management_version` 与组织 `version` 乐观锁；
- `operation_id` 写操作幂等；
- 授权审计、敏感访问日志保护和 Outbox 事件。

## 调用链

```text
Miniapp
  -> app-api Handler
  -> app-api Logic
  -> Identity gRPC Client
  -> Identity Unary Auth Interceptor
  -> Identity Logic
  -> IdentityAdmin / Organization / Authorization Manager
  -> Store / Transaction Store
  -> MySQL + Redis
```

职责边界：

- Handler：解析 HTTP 或接收 RPC，不实现业务规则；
- Logic：把协议对象转换为领域命令，取得当前 Principal；
- Manager：权限、状态机、层级、幂等、乐观锁和事务边界；
- Store：持久化接口，相当于 Java 项目中的 Mapper/Repository 抽象；
- `mysqlstore`：Store 的 MySQL 实现和具体 SQL；
- Interceptor：在进入需要认证的 RPC Logic 前校验 JWT 与授权版本，并把 Principal 写入 Context。

`AuthorizationManager` 只负责读取授权上下文。管理员写操作统一进入 `IdentityAdminManager`，不存在第二套旧写链路。

## 目录

```text
service/identity/rpc/
├── identity.go                    # RPC 启动、服务注册和鉴权拦截器
├── etc/                           # 本地配置
├── identityservice/               # 生成的 RPC Client 包装
└── internal/
    ├── account/                   # 登录账号与手机号绑定
    ├── authorization/             # 授权上下文读取
    ├── identityadmin/             # 账号和医生管理领域
    ├── organization/              # 组织管理与公共目录领域
    ├── logic/                     # RPC 用例适配
    ├── repository/mysqlstore/     # MySQL Store 与事务实现
    ├── server/                    # gRPC Server 方法
    ├── session/                   # Access/Refresh 会话
    └── svc/                       # 依赖装配
```

`identityadmin` 和 `organization` 按阅读职责拆分文件，包本身保持稳定：

- `manager.go`：Manager 结构、共享事务模板；
- `administrator.go`：管理员查询和写操作；
- `doctor.go` / `directory.go`：医生或公共目录能力；
- `self.go`：本人资料；
- `validation.go`：输入和领域约束。

## HTTP 能力

HTTP 接口由 `app-api` 暴露，完整字段见 `contracts/api/` 与 `docs/api/openapi.json`。

| 范围 | 主要路径 |
|---|---|
| 登录 | `/api/v1/auth/phone/*`、`/api/v1/auth/token/*`、`/api/v1/auth/me` |
| 公共目录 | `/api/v1/directory/organization-context`、`/departments`、`/departments/:id/doctors` |
| 组织管理 | `/api/v1/admin/identity/organization-units/*` |
| 账号管理 | `/api/v1/admin/identity/accounts/*` |
| 医生管理 | `/api/v1/admin/identity/doctors/*` |

停用、恢复和撤销统一使用动作型 `POST /:id/disable|enable|revoke`。这些操作改变状态，不是物理删除。

## 本地运行

从仓库根目录执行：

```powershell
docker compose `
  --env-file .env `
  -f deploy/compose/docker-compose.yml `
  up -d mysql redis

.\scripts\db-bootstrap-local.ps1
.\scripts\migrate.ps1 -Service identity -Direction up
.\scripts\start-backend.ps1 -Restart
```

统一启动脚本按 UTF-8 读取 `.env`，避免 Windows PowerShell 破坏阿里云中文签名。

需要本地固定验证码时：

```text
APP_ENV=local
SMS_PROVIDER=local
LOCAL_SMS_CODE=246810
```

真实联调使用 `SMS_PROVIDER=aliyun`。生产环境配置为 `local` 时 Identity 会拒绝启动。

## 测试

```powershell
go test ./service/identity/rpc/...
.\scripts\check.ps1
```

MySQL 集成测试通过 `IDENTITY_TEST_MYSQL_DSN` 显式启用。阶段收尾已经验证组织、账号和医生生命周期，
以及真实 HTTP → RPC → Manager → MySQL 链路、幂等、乐观锁、层级保护和旧路由回归。

## 后续扩展规则

1. 先更新 `plan/` 和 `contracts/`，不要从 Handler 直接开始写；
2. 新管理员写能力进入 `IdentityAdminManager` 或 `OrganizationManager`，不要扩展 `AuthorizationManager`；
3. 写操作必须在同一事务中更新主数据、审计和 Outbox；
4. 涉及授权的变化必须递增 `authorization_version` 并发布版本；
5. 涉及手机号、验证码或 Token 的新 RPC 必须加入客户端和服务端正文日志屏蔽名单；
6. 新查询要明确是公共目录、本人查询还是管理员查询，不能共用一个返回对象泄漏字段；
7. 新接口完成后重新生成 Swagger，并补充数据库集成与 HTTP 全链路测试。

## 暂不包含

- 多医院租户；
- 多科室任职和排班；
- 多设备会话管理界面；
- 账号合并与注销工作流；
- 复杂审批和临时授权。

这些能力必须先进入 Plan 评审，不能直接扩展现有 CRUD。
