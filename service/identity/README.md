# Identity 身份服务

Identity Service 是账号、登录会话、角色权限、医院组织和医生档案的数据拥有者。当前规划范围已经完成，
后续业务服务通过 Access Token、Identity RPC 和授权事件使用这些能力，不得直接访问 Identity 数据表。

## 已实现能力

### 认证与会话

- 阿里云 PNVS 手机验证码发送与校验；
- 首次手机号登录自动创建患者账号；
- Ed25519 JWT Access Token；
- Redis Refresh Session、Token 轮换、重放检测和注销；
- Redis 授权版本校验，账号、角色或科室权限变化后旧 Token 失效；
- 受限的本地固定验证码校验器，仅允许 `local/test` 环境。

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
- 授权审计、敏感访问日志保护和 Outbox 事件；
- Outbox 轮询发布 Kafka，Consumer 将授权版本单调写入 Redis，成功后提交 Kafka Offset。

## 调用链

```text
Miniapp
  -> app-api Handler
  -> app-api Logic
  -> Identity gRPC Client
  -> Identity Unary Auth Interceptor
  -> Identity Logic
  -> Authentication / Session / Account / Organization Manager
  -> Store / Transaction Store
  -> MySQL（主数据、审计、Outbox）

MySQL Outbox -> Kafka -> Authorization Version Consumer -> Redis -> Commit Offset
```

职责边界：

- Handler：解析 HTTP 或接收 RPC，不实现业务规则；
- Logic：把协议对象转换为领域命令，取得当前 Principal；
- Manager：权限、状态机、层级、幂等、乐观锁和事务边界；
- Store：定义领域需要的持久化能力，使 Manager 不依赖具体数据库实现；
- `mysqlstore`：Store 的 MySQL 实现和具体 SQL；
- Interceptor：在进入需要认证的 RPC Logic 前校验 JWT 与授权版本，并把 Principal 写入 Context。

最新 Principal 只在登录和刷新会话时由 `session` 内部读取；普通请求直接使用 Interceptor 注入的 Principal，不提供额外的授权资料查询 RPC。

## 目录

```text
service/identity/rpc/
├── identity.go                    # RPC 启动、服务注册和鉴权拦截器
├── etc/                           # 本地配置
├── identityservice/               # 生成的 RPC Client 包装
└── internal/
    ├── authentication/            # 认证错误、Store/Session 端口
    │   ├── manager/               # 手机号验证码认证编排
    │   └── sms/                   # 阿里云与本地短信校验实现
    ├── account/                   # 账号领域模型、Store 端口与 manager
    ├── authorization/version/     # 授权版本事件与 Kafka→Redis Consumer
    ├── messaging/kafka/           # transport-only Reader/Writer
    ├── messaging/outbox/          # 通用 MySQL Outbox→Kafka Publisher
    ├── organization/              # 组织模型、Store 端口与 manager
    ├── logic/                     # RPC 用例适配
    ├── repository/mysqlstore/     # MySQL Store 与事务实现
    ├── repository/redisstore/     # Refresh Session 的 Redis 实现
    ├── server/                    # gRPC Server 方法
    ├── session/                   # Access/Refresh 会话
    └── svc/                       # 依赖装配
```

`account/manager` 和 `organization/manager` 按阅读职责拆分文件；领域根包保存模型、错误和 Store 端口：

- `manager.go`：Manager 结构、共享事务模板；
- `administrator.go`：管理员查询和写操作；
- `doctor.go` / `directory.go`：医生或公共目录能力；
- `unit_read.go` / `unit_write.go`：组织单元读写操作；
- `profile.go`：本人资料；
- `validation.go`：输入和领域约束。

## 本地运行

从仓库根目录执行：

```powershell
docker compose `
  --env-file .env `
  -f deploy/compose/docker-compose.yml `
  up -d mysql redis kafka

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

全新体验环境由 `identity-bootstrap-admin` 一次性任务在迁移后创建唯一医院根节点，并根据
`IDENTITY_BOOTSTRAP_ADMIN_PHONES` 创建多个初始超级管理员。该工具只用于部署初始化；正常运行后不得用它
替代管理员业务接口。具体配置与顺序见 `deploy/production/README.md`。

## 测试

```powershell
go test ./service/identity/rpc/...
.\scripts\check.ps1
```

MySQL 集成测试通过 `IDENTITY_TEST_MYSQL_DSN` 显式启用。阶段收尾已经验证组织、账号和医生生命周期，
以及真实 HTTP → RPC → Manager → MySQL 链路、幂等、乐观锁、层级保护和旧路由回归。

## 后续扩展规则

1. 先更新 `plan/` 和 `contracts/`，不要从 Handler 直接开始写；
2. 新账号管理能力进入 `account/manager`，新组织管理能力进入 `organization/manager`；不要为没有业务调用方的授权资料查询新增 RPC；
3. 写操作必须在同一事务中更新主数据和审计；需要发布集成事件时，Outbox 也必须进入同一事务；
4. 涉及授权的变化必须递增 `authorization_version`，在同一事务写入授权 Outbox，并由 Consumer 从 Kafka 同步到 Redis；组织变化当前只写审计，不发布事件；
5. 涉及手机号、验证码或 Token 的新 RPC 必须加入客户端和服务端正文日志屏蔽名单；
6. 新查询要明确是公共目录、本人查询还是管理员查询，不能共用一个返回对象泄漏字段；
7. 新接口完成后同步生成代码、消费方，并补充数据库集成与 HTTP 全链路测试。

## 暂不包含

- 多医院租户；
- 多科室任职和排班；
- 多设备会话管理界面；
- 账号合并与注销工作流；
- 复杂审批和临时授权。

这些能力必须先进入 Plan 评审，不能直接扩展现有 CRUD。
