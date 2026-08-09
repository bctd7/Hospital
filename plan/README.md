# Hospital 项目规划文档

本目录按照“技术与总体架构在前、具体业务需求在后”的顺序组织。

## 当前已确认

- 产品形态：微信小程序优先，后端 API 可供未来独立 App 复用；
- 小程序前端：uni-app + Vue 3 + TypeScript + Vite，一级导航为首页、挂号、消息、我的；
- 后端：Go + go-zero；
- 数据与基础设施：MySQL 8.4、Redis、Kafka、Docker；
- 仓库：Monorepo；
- 首期核心服务：Identity、Appointment、Planning、Navigation；
- 权限：Identity 管理身份事实并签发 Token，各业务服务本地完成最终授权，不建设集中 Permission Service；
- 数据：每个服务拥有独立逻辑数据库，首期允许部署在同一 MySQL 实例；
- 跨服务协作：同步使用 zRPC，状态传播使用 Outbox + Kafka，页面聚合由 `app-api` 完成。

## 推荐阅读顺序

### 总体架构

1. [00-overall-technical-plan.md](./00-overall-technical-plan.md)：技术选型、go-zero 使用方式、基础设施和工程基线。
2. [01-overall-functional-framework.md](./01-overall-functional-framework.md)：产品定位、总体功能模块和患者主流程。
3. [02-service-and-data-boundaries.md](./02-service-and-data-boundaries.md)：服务拆分、数据库所有权、跨服务共享和一致性方案。
4. [03-identity-and-access-control.md](./03-identity-and-access-control.md)：Identity Service、本地授权、部门范围和权限扩展。
5. [04-logging-and-audit.md](./04-logging-and-audit.md)：日志、审计、公共代码和集中日志平台。
6. [05-wechat-registration-and-doctor-onboarding.md](./05-wechat-registration-and-doctor-onboarding.md)：微信自动注册、手机号登记和医生身份开通。

### 客户端规划

1. [微信小程序规划包](./miniapp/)：前端外壳、页面信息架构、接口边界和后续联调规划。

### 具体业务需求

具体业务流程统一放在 [requirements](./requirements/) 目录：

1. [业务模块全景](./requirements/00-business-module-overview.md)；
2. [需求讨论与交付路线](./requirements/01-requirements-and-delivery-roadmap.md)；
3. [核心检查顺序规划](./requirements/02-core-examination-planning.md)。

后续预约、地图、就诊人、报告、通知和公告均在 `requirements/` 中分别建立文档；数据库表设计、API 和事件契约在对应业务规则确认后再编写。
