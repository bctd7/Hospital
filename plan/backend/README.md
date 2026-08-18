# 后端规划

后端当前由 App API、Identity RPC、Appointment RPC 和 Guidance RPC 四个可运行服务组成。这里从业务和架构角度
记录服务边界、稳定规则、已实现状态和经过确认的后续提案，不复制契约字段、逐接口清单或 SQL 表结构。

## 文档边界

- 本目录回答“为什么需要该能力、谁拥有数据、业务规则是什么、哪些已经实现或明确不做”；
- `service/*/README.md` 回答“当前代码在哪里、请求如何流转、依赖如何装配、怎样运行和验证”；
- `contracts/api`、`contracts/proto` 回答“接口叫什么、请求响应有哪些字段”；
- `migrations/` 回答“数据库当前有哪些表、列、索引和约束”。

Plan 可以用少量目录或流程说明验证方案确实可落地，但不维护生成代码文件清单；服务 README 可以概括业务能力，
但不重新发明 Plan 中的业务规则。App API 没有独立领域数据和状态机，因此不单独建立业务模块 Plan，其职责由
架构文档和 [`service/app/README.md`](../../service/app/README.md) 共同说明。

## 文档

- [架构与数据边界](./01-architecture-and-data-boundaries.md)；
- [交付与质量基线](./02-delivery-and-quality.md)；
- [模块索引](./modules/README.md)；
- [Identity 当前范围归档](./modules/implemented/01-identity-service.md)；
- [Appointment 已实现归档](./modules/implemented/02-appointment-service.md)；
- [Appointment 组织只读副本提案](./modules/proposals/01-appointment-organization-read-model.md)；
- [Guidance 当前范围归档](./modules/implemented/03-guidance-service.md)；

## 固定分层

```text
HTTP Handler
  -> App API Logic
  -> gRPC Client
  -> RPC Interceptor
  -> RPC Logic
  -> Domain Manager / Calculator
  -> Store / Repository
```

- Handler、Server 和 Logic 只做协议适配、上下文读取与错误转换；
- Manager 负责权限、状态机、幂等、事务和领域规则；
- 无状态且职责单一的计算能力可以使用含义明确的 Calculator，不强行包装成 Manager；
- 领域根包保存模型、稳定错误和窄 Store 端口；
- Repository 实现 Store，不反向决定业务状态；
- ServiceContext 只创建、装配和关闭依赖。

## 新功能流程

1. 先确认业务拥有者、调用者、权限、状态机和失败语义；
2. 更新 Plan 与契约源文件；
3. 增加 Migration、Manager、Repository 和协议适配；
4. 补充单元测试、数据库集成测试和必要的 HTTP 链路测试；
5. 更新相应 README 与已实现归档；
6. 运行 `scripts/quality/verify-repository.ps1`。

不为尚无数据所有权、发布边界或真实调用方的概念提前创建服务和空目录。
