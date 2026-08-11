# Identity RPC 内部代码导航

本目录保持 Go 包和 go-zero 生成结构。README 只帮助阅读当前代码；领域规则见 `plan/backend/modules/`，
接口字段见 `contracts/`。

## 阅读一条请求

```text
server/identity_service_server.go
  -> logic/<rpc>_logic.go
  -> account|authorization|identityadmin|organization|session Manager
  -> 对应 store.go
  -> repository/mysqlstore/*_store.go
  -> migrations/identity/*.sql
```

不要从 `_test.go` 或 SQL 反向猜业务规则。先读 Manager，再用测试和 SQL 验证边界。

## 分层职责

| Go 目录 | 职责 |
|---|---|
| `server/` | 接收 gRPC 请求并创建 Logic |
| `logic/` | 协议对象转换、取得 Principal 和错误映射 |
| `account/`、`identityadmin/`、`organization/`、`session/` | 领域规则和事务编排 |
| 各领域 `store.go` | 定义领域需要的持久化能力 |
| `repository/mysqlstore/` | MySQL Store、事务和 SQL 实现 |
| `repository/redis_session_store.go` | Refresh Session 的 Redis 实现 |
| `svc/service_context.go` | 创建并注入运行依赖 |

## 包职责

### `account`

手机号/微信账号登录、账号创建、手机号绑定和本人账号查询。

### `authorization`

只读取授权上下文。管理员写操作不放在这里。

### `identityadmin`

账号、医生和本人展示资料：

- `manager.go`：Manager 与共享事务模板；
- `administrator.go`：管理员账号查询、手机号搜索和账号启停；
- `doctor.go`：医生目录、开通、资料修改、调岗和撤销；
- `self.go`：本人展示资料；
- `validation.go`：输入与业务约束；
- `store.go`：普通 Store 和事务 Store 接口。

### `organization`

医院、院区和科室：

- `manager.go`：Manager、命令和内部原始读取；
- `administrator.go`：管理员组织 CRUD；
- `directory.go`：公共目录；
- `validation.go`：组织类型、名称和 UUID 规则；
- `store.go`：读取与事务接口。

### `session`

Access/Refresh Token、Session 轮换、重放检测、注销和授权版本比较。

### `logic`

一个 RPC 方法一个文件，保持 goctl 默认的扁平目录，不额外分包。分类索引见
[`logic/README.md`](./logic/README.md)。

### `repository`

```text
repository/
├── redis_session_store.go
└── mysqlstore/
    ├── store.go
    ├── account_store.go
    ├── authorization_store.go
    ├── identity_admin_store.go
    ├── identity_admin_tx_store.go
    ├── organization_store.go
    └── organization_tx_store.go
```

`mysqlstore/store.go` 创建共享数据库连接；具体 SQL 按领域放入对应文件。Manager 只依赖 Store 接口，不依赖
MySQL 类型。

## 事务

Manager 通过 Store 的 `Within...Transaction` 开始事务，并在回调内使用 TxStore。普通 Store 不能保证多条
语句原子性，TxStore 不能在没有 BeginTx 创建的事务对象时使用。

写操作在一个事务中完成：目标锁定、乐观锁、主数据、审计、Outbox 和幂等结果。提交后才发布 Redis
授权版本。

## Review 建议

1. 第一遍排除 `_test.go`，按 Server → Logic → Manager → Store 阅读；
2. 第二遍只看同领域测试，验证权限、状态、并发和错误；
3. 最后查看 MySQL 集成测试和迁移，确认真实持久化行为。

测试文件与包放在一起是 Go 的常规做法。不要为减少文件数量把测试搬到独立目录，也不要移动生成 Logic
破坏包路径。
