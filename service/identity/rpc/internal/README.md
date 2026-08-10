# Identity RPC 内部代码导航

这个目录保持 go-zero 和 Go 的标准包结构。不要仅为了文件分组移动 Go 文件：在 Go 中，目录就是包；移动文件会改变导入路径、可见性和测试边界。

## 从一次 RPC 请求开始阅读

建议始终按下面的顺序阅读，不要从测试文件或 SQL 开始：

```text
server/identity_service_server.go
  -> logic/*_logic.go
  -> account|authorization|organization|session/manager.go
  -> 对应目录的 store.go 接口
  -> repository/mysqlstore/*_store.go
  -> migrations/identity/*.sql
```

对应 Java 分层：

| Go 目录 | 主要职责 | Java 类比 |
| --- | --- | --- |
| `server/` | 接收 gRPC 请求并创建 Logic | Controller 入口 |
| `logic/` | DTO 转换、取得当前用户、转换 gRPC 错误 | Controller / Application Adapter |
| `account/`、`authorization/`、`organization/`、`session/` | 业务规则和事务编排 | Service |
| 各业务目录的 `store.go` | 数据访问接口 | Mapper / Repository 接口 |
| `repository/mysqlstore/` | MySQL 实现和 SQL | Mapper XML / Repository 实现 |
| `repository/redis_session_store.go` | Redis 会话实现 | Redis Repository 实现 |
| `svc/service_context.go` | 创建并组装依赖 | Spring Configuration / Bean 装配 |

## 目录职责

### `logic/`

RPC 适配层。文件较多是因为 goctl 默认按“一个 RPC 方法一个文件”生成。详细分类和当前完成状态见 [logic/README.md](logic/README.md)。

### `account/`

账号注册、手机号绑定、手机号查询和登录流程的业务编排。

- `manager.go`：账号业务
- `phone_login.go`：手机号验证码登录业务
- `*_test.go`：对应 Manager 的单元测试

### `authorization/`

角色、科室归属、账号状态和权限版本的业务规则。

- `manager.go`：授权变更业务
- `store.go`：授权 Store 和事务 Store 接口
- `manager_test.go`：授权业务单元测试

### `organization/`

医院、院区和科室的组织管理业务。

- `model.go`：`Unit`、`UnitType`、`Status`、`ListFilter`
- `errors.go`：组织业务错误
- `store.go`：普通读取与事务操作接口
- `manager.go`：查询、创建、修改、启用和禁用规则
- `manager_test.go`：Manager 单元测试

### `session/`

访问令牌、刷新令牌和会话生命周期。

- `manager.go`：会话业务
- `store.go`：会话和授权上下文读取接口
- `token.go`：令牌相关模型或辅助代码

### `repository/`

数据访问实现按存储技术分组：

```text
repository/
  redis_session_store.go
  redis_session_store_integration_test.go
  mysqlstore/
    store.go
    account_store.go
    authorization_store.go
    authorization_tx_store.go
    organization_store.go
    organization_tx_store.go
```

`mysqlstore/store.go` 只负责数据库连接和共享 `Store`；具体 SQL 按业务放在各自的 `*_store.go` 中。

## 如何减少测试文件对 Review 的干扰

测试继续与被测试包放在一起，这是 Go 的惯例，也是访问包内未导出实现所必需的。Review 时建议分两遍：

1. 第一遍只看非 `_test.go` 文件，理解生产链路。
2. 第二遍只打开与本次改动同名的 `_test.go`，验证行为边界。

测试类型：

- `manager_test.go`：不连接真实数据库的业务单元测试。
- `*_logic_test.go`：RPC Logic、身份和错误码测试。
- `*_integration_test.go`：连接 MySQL 或 Redis 的集成测试，通常需要测试环境变量。
