# 后端规划

## 总体规划

[overview](./overview/) 保存跨模块的大方向：技术选型、系统边界、可观测性、业务全景和交付路线。

## 模块设计

[modules](./modules/) 保存可以独立讨论和实现的业务模块：

1. [Identity 身份与访问控制](./modules/01-identity-and-access-control.md)
2. [检查顺序规划](./modules/02-examination-planning.md)
3. [业务审计](./modules/03-business-audit.md)
4. [Identity 组织、科室、医生与用户管理](./modules/04-organization-staff-and-seed-data.md)

总体规划回答“项目如何组织、为何这样拆、先做什么、哪些能力复用、何时成为微服务”；模块设计
回答“业务流程、状态机、数据模型、具体技术或算法、怎样验收”。接口和事件进入实现前必须在
`contracts/` 中落地。
