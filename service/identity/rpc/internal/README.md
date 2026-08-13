# Identity RPC 内部代码导航

## 一条请求怎么读

```text
server/identity_service_server.go
  -> logic/<rpc>_logic.go
  -> 对应领域的 manager
  -> 领域 Store 接口
  -> repository/mysqlstore 或 Redis 实现
```

`server` 和 `logic` 是协议适配层；权限、状态机、事务和幂等规则必须留在
Manager。Logic 不得直接访问 Repository。

领域根包负责表达“是什么”：模型、稳定错误和 Store 端口；`manager/` 负责表达
“怎样操作”：权限判断、校验、状态迁移、事务编排和幂等流程。Repository 只实现
根包声明的 Store 端口，因此不会反向依赖业务 Manager。

## 领域目录

```text
internal/
├─ authentication/             登录凭据验证、账号解析和开始 Session
│  ├─ manager/                 微信登录、手机号登录和认证校验
│  └─ provider/                微信、阿里云短信、本地测试 Provider
├─ account/
│  ├─ model.go / errors.go       账号领域模型与稳定错误
│  ├─ store.go                   账号持久化端口
│  └─ manager/                 账号、医生、本人资料的业务操作
├─ authorization/
│  ├─ context/                   有效 Principal 的 Manager 与 Store 端口
│  └─ version/                   授权版本事件、Consumer 与 Redis 写入端口
├─ organization/
│  ├─ model.go / errors.go       组织领域模型与稳定错误
│  ├─ store.go                   组织持久化端口
│  └─ manager/                 组织单元管理与公共目录读取
├─ session/                    Manager、模型、Store 端口和 Token 工具
├─ messaging/
│  ├─ kafka/                   只提供 Reader/Writer 传输适配
│  └─ outbox/                  通用 MySQL Outbox→Kafka Publisher
├─ repository/                 Redis 与 MySQL 实现
├─ logic/                      一个 RPC 一个 go-zero Logic
├─ server/                     gRPC Server 方法
└─ svc/                        依赖装配
```

## 容易混淆的边界

### Authentication、Account 和 Session

- `authentication/manager` 验证微信或手机凭据，找到或创建登录账号；
- `authentication/provider` 只适配微信、阿里云手机号等外部登录渠道，并把已验证的外部身份交给认证 Manager；
- `account/manager` 管理账号状态、医生身份、科室和展示资料；
- `session` 读取最新 Principal，签发/刷新 Token，维护 Refresh Session。

Provider 不创建 Session、不签发 Token，也不负责账号管理。`svc/login_providers.go` 只根据配置选择 Provider，`svc/manager_wiring.go` 负责注入，渠道差异不会进入 RPC Logic。

### Authorization Version Consumer

- `authorization/version` 定义授权版本事件，并完成 Kafka 拉取、Redis 单调更新和 Offset 提交；
- `session` 仅在登录和刷新时从 MySQL 读取最新 Principal；普通请求使用 Interceptor 从 Token 恢复并注入的 Principal；
- 角色、权限、科室和账号状态的修改由 `account/manager` 完成。

### Outbox

`messaging/outbox` 不理解授权业务。账号写事务把授权事件与主数据、审计一起
保存进 MySQL Outbox；Publisher 只负责将任何待发送记录可靠发布到 Kafka。

```text
Account Manager
  -> MySQL 主数据 + 审计 + Outbox（同一事务）
  -> Outbox Publisher
  -> Kafka
  -> Authorization Version Consumer
  -> Redis
  -> Commit Kafka Offset
```

Redis 写失败时不提交 Offset；Commit 失败时消息可能重放。Redis 的版本更新
必须保持单调和幂等。

## ServiceContext

```text
svc/service_context.go  依赖分组、总装配顺序、关闭顺序
svc/resources.go        MySQL、Redis 的创建、连通性检查和关闭
svc/token_components.go JWT、密钥、授权版本 Store/Validator、Refresh Session Store
svc/login_providers.go  微信、阿里云手机号和本地验证码 Provider
svc/manager_wiring.go   Session 与 RPC 业务 Manager 装配
svc/messaging.go        Kafka、Outbox、授权版本 Consumer
```

Logic 通过以下分组访问依赖：

```go
svcCtx.Managers.Account
svcCtx.Managers.Authentication
svcCtx.Managers.Session
svcCtx.Security.Token
svcCtx.Workers.AuthorizationVersionConsumer
```

## Common 安全包

- `common/authn`：Principal、JWT 和认证中间件/拦截器；
- `common/authz`：permission 与科室范围；
- `common/authz/version`：Token 授权版本校验；
- `common/authz/version/redisstore`：授权版本的 Redis 实现。
