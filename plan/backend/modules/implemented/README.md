# 已实施后端模块

本目录只保存已经确认、完成实现并通过阶段验收的模块 Plan。它们用于说明当前行为和设计边界，不再作为
下一阶段需求草案；后续维护若改变业务规则，必须先更新对应 Plan 和契约。

- [01 Identity Service 首期实施归档](./01-identity-service.md)：手机号认证、Token、会话、角色权限、授权版本、
  组织、账号、医生、审计及授权版本 Outbox → Kafka → Redis 同步的统一首期边界。

业务审计没有整体归档到这里：Identity 审计已有实现，但 Appointment、Report 等领域仍需在
各自模块实施时完成审计规则。
