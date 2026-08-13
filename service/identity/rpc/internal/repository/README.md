# Identity Repository

本目录只保存领域 Store 接口的数据库适配实现，不定义账号、组织、授权或会话规则。领域接口仍由
`account`、`authentication`、`organization`、`session` 和 `messaging/outbox` 等包拥有。

```text
repository/
├─ mysqlstore/   MySQL 查询、事务、审计和 Outbox 持久化
└─ redisstore/   Redis Refresh Session 持久化
```

## MySQL 文件导航

| 文件 | 内容 |
|---|---|
| `store.go` | 共享 MySQL 连接池的创建、健康检查和关闭 |
| `errors.go` | 多个 MySQL 适配文件共用的驱动错误识别 |
| `authentication.go` | 已验证手机号对应账号的查询或创建 |
| `principal.go` | 从账号、角色和权限表组装最新 Principal |
| `account_profile.go` | 本人展示资料读取与更新 |
| `doctor_directory.go` | 面向公共目录的有效医生查询 |
| `account_query.go` | 管理端账号详情、分页和手机号精确查找 |
| `account_transaction.go` | 账号和医生管理的事务、锁、审计及授权 Outbox |
| `organization_query.go` | 组织单元查询和组织事务入口 |
| `organization_transaction.go` | 组织单元写入、锁和审计；当前不发布组织事件 |
| `outbox.go` | 待发布授权事件的读取和发布状态更新 |

`Store` 不是业务 Store 接口，而是共享 `*sql.DB` 的 MySQL 适配器。不同领域的方法分散在上述文件中，
但共同挂在同一个 `Store` 上，以便账号、审计和 Outbox 在需要时使用同一数据库事务。

`errors.go` 只保留跨多个适配文件复用的 MySQL 驱动错误判断。领域错误仍由领域包定义，不能放入这里。
