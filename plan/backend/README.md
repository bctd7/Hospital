# 后端开发与规划指南

本目录保存后端当前有效的架构决定和模块计划。接口字段不在 Plan 中重复维护：HTTP 以
`contracts/api/` 为准，RPC 以 `contracts/proto/` 为准，事件以 `contracts/events/` 为准。

## 文档结构

- [overview](./overview/)：技术基线、服务边界、可观测性、能力地图和交付路线；
- [modules](./modules/)：可以独立实现和验收的业务模块。

Identity 与组织模块已经完成首期实现；下一阶段从
[预约检查服务](./modules/05-appointment-and-examination-booking.md)继续。

## 新增后端功能的标准流程

### 1. 业务设计

在 `plan/backend/modules/` 明确：

- 用户、角色和业务目标；
- 状态机、并发规则和失败处理；
- 数据拥有者及跨服务边界；
- 权限、隐私、审计和验收标准；
- 本期实现与明确不实现的内容。

没有清晰数据归属和调用方的能力，不提前创建服务、表或 RPC。

### 2. 契约优先

按影响范围更新：

- `contracts/api/*.api`：小程序可见的 HTTP；
- `contracts/proto/**/*.proto`：服务间同步 RPC；
- `contracts/events/`：异步事实事件；
- `migrations/<service>/`：业务事实表和版本迁移。

契约必须使用稳定业务名称，不暴露数据库表结构，也不为内部实现方便复制相同概念。

### 3. 分层实现

```text
HTTP Handler
  -> App API Logic
  -> RPC Client
  -> RPC Interceptor
  -> RPC Logic
  -> Domain Manager
  -> Store / Repository
```

- Handler/Server：协议适配；
- Logic：请求转换和用例调用；
- Manager：权限、状态机、幂等、事务与领域规则；
- Store：持久化接口；
- `mysqlstore`：SQL 和事务实现；
- `ServiceContext`：只负责依赖创建和注入。

复杂业务下沉 Manager，不通过移动 goctl 生成的 Logic 建立自定义层级目录。

### 4. 写操作基线

管理写操作默认考虑：

- `operation_id` 幂等；
- `version`/`management_version` 乐观锁；
- 主数据、审计和 Outbox 同一事务；
- 授权变化后的 `authorization_version`；
- Outbox 发布 Kafka、Consumer 幂等投影 Redis，并在成功后提交 Offset；
- 明确的 gRPC/HTTP 错误码；
- 敏感正文日志屏蔽；
- 数据库集成和 HTTP 全链路测试。

只有单表简单写入且不涉及上述规则时，才允许减少结构。

### 5. 生成与验证

```powershell
goctl api validate -api contracts/api/app.api
goctl api swagger --api contracts/api/app.api --dir docs/api --filename openapi
.\scripts\check.ps1
```

Proto 变更后使用仓库既有生成命令更新 `contracts/gen` 和 RPC 包，生成文件不得单独手改。

## Definition of Done

一个后端功能只有同时满足以下条件才算完成：

1. Plan、契约和实现一致；
2. 权限、错误、并发和事务规则有测试；
3. 数据库迁移可在空库执行；
4. App API 与下游 RPC 均执行自身安全边界；
5. 敏感字段不进入日志和接口文档示例；
6. OpenAPI 快照已更新；
7. `scripts/check.ps1` 通过；
8. 过时接口、Mock、重复 Manager 和被替代文档已经删除。
