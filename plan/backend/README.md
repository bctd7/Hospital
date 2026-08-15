# 后端规划

后端当前由 App API、Identity RPC 和 Appointment RPC 三个可运行服务组成。这里保留稳定架构和模块状态，
不复制契约字段或 SQL 表结构。

## 文档

- [架构与数据边界](./01-architecture-and-data-boundaries.md)；
- [交付与质量基线](./02-delivery-and-quality.md)；
- [模块索引](./modules/README.md)；
- [Identity 已实现归档](./modules/implemented/01-identity-service.md)；
- [Appointment 已实现归档](./modules/implemented/02-appointment-service.md)；
- [Appointment 组织只读副本提案](./modules/proposals/01-appointment-organization-read-model.md)；

## 固定分层

```text
HTTP Handler
  -> App API Logic
  -> gRPC Client
  -> RPC Interceptor
  -> RPC Logic
  -> Domain Manager
  -> Store / Repository
```

- Handler、Server 和 Logic 只做协议适配、上下文读取与错误转换；
- Manager 负责权限、状态机、幂等、事务和领域规则；
- 领域根包保存模型、稳定错误和窄 Store 端口；
- Repository 实现 Store，不反向决定业务状态；
- ServiceContext 只创建、装配和关闭依赖。

## 新功能流程

1. 先确认业务拥有者、调用者、权限、状态机和失败语义；
2. 更新 Plan 与契约源文件；
3. 增加 Migration、Manager、Repository 和协议适配；
4. 补充单元测试、数据库集成测试和必要的 HTTP 链路测试；
5. 更新相应 README 与已实现归档；
6. 运行 `scripts/check.ps1`。

不为尚无数据所有权、发布边界或真实调用方的概念提前创建服务和空目录。
