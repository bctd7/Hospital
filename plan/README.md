# Hospital 规划索引

本目录只保留当前有效的架构决定、已经完成的业务归档和仍需评审的后续事项。HTTP、gRPC 和事件字段不在
Plan 中重复维护，分别以 `contracts/api/`、`contracts/proto/` 和 `contracts/events/` 为事实来源。

## 当前状态

| 模块 | 状态 | 说明 |
|---|---|---|
| Identity | 当前范围已完成 | 手机号认证、会话、授权版本、组织、账号与医生管理已经形成闭环 |
| Appointment | 当前范围已完成 | 检查资源、预约、报到、候检叫号、检查、报告与消息已经形成闭环 |
| Guidance | 两项基础能力已实现 | 独立服务；已实现检查项目先后关系和两点步行路线，方案生成及完整预约后导航继续规划 |
| 小程序 | 已接入上述真实接口 | 患者端与工作人员端共用登录身份；患者端已可搜索高德地点并在微信地图展示两点步行路线 |
| 就诊人、缴费、医保、票据等 | 未规划或仅保留入口 | 不把展示入口误写成后端已实现能力 |

Appointment 原先按阶段拆分的 1～5 号实施稿已经合并为
[Appointment 已实现归档](./backend/modules/implemented/02-appointment-service.md)。旧 Mock 方案、重复接口清单和已经
完成的阶段待办不再保留；需要追溯时使用 Git 历史。

Identity 与 Appointment 当前没有仍待收尾的实施阶段。归档末尾的“当前不包含”只说明产品边界，不是遗留
任务；下一阶段从智能导诊与检查导航重新进入业务讨论。

## 阅读顺序

- [后端规划](./backend/README.md)：服务边界、交付规则和模块状态；
- [前端规划](./frontend/README.md)：小程序页面事实、身份版本和剩余边界；
- [Identity 当前范围归档](./backend/modules/implemented/01-identity-service.md)；
- [Appointment 已实现归档](./backend/modules/implemented/02-appointment-service.md)；
- [智能导诊与检查导航页面方案](./frontend/06-intelligent-guidance-and-navigation.md)；
- [Guidance 智能导诊业务逻辑](./backend/modules/proposals/02-guidance-service.md)。

## 文档边界

- README 回答“当前有什么、从哪里读、如何运行”；
- Plan 回答“为什么这样设计、当前完成到哪里、后续要确认什么”；
- 契约回答“路径、字段、错误和版本是什么”；
- Migration 回答“数据库最终结构是什么”。

已实现功能发生变化时，同一次提交应更新相应归档和模块 README。被替代的方案直接修改或删除，不长期并列
保存多个相互冲突的版本。
