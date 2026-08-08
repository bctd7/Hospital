# Hospital 项目规划文档

本目录用于保存 Hospital 项目的总体技术规划和后续实施计划。

当前已经确认：

- 产品形态：微信小程序优先，后端提供可供未来独立 App 复用的 API。
- 后端方向：Go + go-zero 微服务框架。
- 基础设施：Redis、关系型数据库、Kafka、Docker。
- 开发方式：单仓库管理，按业务能力形成可独立部署单元。
- 当前阶段：只确定总体技术路线和可扩展的项目组织，不提前锁定微服务数量与边界。

## 文档索引

- [00-overall-technical-plan.md](./00-overall-technical-plan.md)：总体技术架构、技术栈、仓库结构和演进路线。
- [01-logging-and-audit-plan.md](./01-logging-and-audit-plan.md)：日志与审计的能力归类、部署边界、实现步骤和验收标准。
- [02-access-control-overview.md](./02-access-control-overview.md)：患者、管理员、RBAC 和数据访问边界的初步讨论。
- [03-business-module-overview.md](./03-business-module-overview.md)：核心排序之外的基础、近期和中长期业务模块。
- [04-core-examination-planning-overview.md](./04-core-examination-planning-overview.md)：检查顺序规划的核心流程、产品边界和待讨论问题。

## 后续文档建议

只有在总体方案确认后，再按实际需要补充：

1. `05-service-boundary.md`：业务范围确认后的微服务边界与依赖。
2. `06-data-design.md`：数据库选型、数据归属和迁移规范。
3. `07-api-contract.md`：小程序 API、错误码和鉴权约定。
4. `08-infrastructure.md`：Redis、Kafka、etcd 和 Docker Compose。
5. `09-observability-and-security.md`：指标、链路、安全与隐私。
6. `10-implementation-roadmap.md`：迭代顺序和验收标准。
