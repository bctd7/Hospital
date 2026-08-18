# Hospital 规划索引

本目录只保留当前有效的架构决定、已经完成的业务归档和仍需评审的后续事项。HTTP、gRPC 和事件字段不在
Plan 中重复维护，分别以 `contracts/api/`、`contracts/proto/` 和 `contracts/events/` 为事实来源。

## 当前状态

| 模块 | 状态 | 说明 |
|---|---|---|
| Identity | 当前范围已完成 | 手机号认证、会话、授权版本、组织、账号与医生管理已经形成闭环 |
| Appointment | 当前范围已完成 | 检查资源、预约、报到、候检叫号、检查、报告与消息已经形成闭环 |
| Guidance | 当前范围已完成 | 三步项目配置、准备规则、智能预约、整组确认、当日顺序和分阶段路线已经形成闭环 |
| 小程序 | 已接入上述真实接口 | 患者端与工作人员端共用登录身份；工作人员配置、患者规划、当日顺序和地图页面均使用真实 HTTP |

Appointment 原先按阶段拆分的 1～5 号实施稿已经合并为
[Appointment 已实现归档](./backend/modules/implemented/02-appointment-service.md)。旧 Mock 方案、重复接口清单和已经
完成的阶段待办不再保留；需要追溯时使用 Git 历史。

Identity、Appointment 与 Guidance 当前范围没有仍待收尾的实施阶段。归档末尾的“当前不包含”和 Guidance
文档中的后续扩展只说明产品边界，不是已承诺但遗漏的任务。

缴费、医保电子凭证、电子票据、住院病案复印等未规划业务不在当前小程序展示占位入口；需要正式规划并具备真实
业务闭环后再重新进入产品范围。

## 阅读顺序

- [后端规划](./backend/README.md)：服务边界、交付规则和模块状态；
- [前端规划](./frontend/README.md)：小程序页面事实、身份版本和剩余边界；
- [Identity 当前范围归档](./backend/modules/implemented/01-identity-service.md)；
- [Appointment 已实现归档](./backend/modules/implemented/02-appointment-service.md)；
- [智能导诊与检查导航页面方案](./frontend/06-intelligent-guidance-and-navigation.md)；
- [Guidance 当前范围归档](./backend/modules/implemented/03-guidance-service.md)。

## 文档边界

- README 回答“当前有什么、从哪里读、如何运行”；
- Plan 回答“为什么这样设计、当前完成到哪里、后续要确认什么”；
- 契约回答“路径、字段、错误和版本是什么”；
- Migration 回答“数据库最终结构是什么”。

已实现功能发生变化时，同一次提交应更新相应归档和模块 README。被替代的方案直接修改或删除，不长期并列
保存多个相互冲突的版本。
