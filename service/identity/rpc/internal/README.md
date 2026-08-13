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

## 领域目录

```text
internal/
├─ authentication/             登录凭据验证、账号解析和开始 Session
│  └─ provider/                微信、阿里云短信、本地测试 Provider
├─ account/
│  └─ manager/                 账号、医生、本人资料的业务操作
├─ authorization/
│  ├─ manager/                 只读取有效 Principal
│  └─ version/                 授权版本事件及 Kafka→Redis Consumer
├─ organization/
│  └─ manager/                 组织单元管理与公共目录读取
├─ session/                    Access/Refresh Token 生命周期
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

- `authentication` 验证微信或手机凭据，找到或创建登录账号；
- `authentication/provider` 只适配微信、阿里云手机号等外部登录渠道，并把已验证的外部身份交给认证 Manager；
- `account/manager` 管理账号状态、医生身份、科室和展示资料；
- `session` 读取最新 Principal，签发/刷新 Token，维护 Refresh Session。

Provider 不创建 Session、不签发 Token，也不负责账号管理。`svc/login_providers.go` 只根据配置选择 Provider，`svc/manager_wiring.go` 负责注入，渠道差异不会进入 RPC Logic。

### Authorization Manager 和 Version Consumer

- `authorization/manager` 只读取账号类型、角色、权限、科室和版本；
- `authorization/version/event.go` 定义授权版本发生变化时的集成事件；
- `authorization/version/consumer.go` 完成 Kafka 拉取、Redis 单调更新和 Offset 提交；
- 角色、权限、科室和账号状态的修改仍由 `account/manager` 完成。

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
svcCtx.Managers.Authorization
svcCtx.Security.Token
svcCtx.Workers.AuthorizationVersionConsumer
```

## Common 安全包

- `common/authn`：Principal、JWT 和认证中间件/拦截器；
- `common/authz`：permission 与科室范围；
- `common/authz/version`：Token 授权版本校验；
- `common/authz/version/redisstore`：授权版本的 Redis 实现。
