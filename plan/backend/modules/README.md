# 后端模块状态

## 已实现

- [Identity 当前范围](./implemented/01-identity-service.md)：认证、会话、授权版本、组织、账号和医生管理；
- [Appointment 当前范围](./implemented/02-appointment-service.md)：检查资源、预约、患者报到、房间候检叫号、检查、报告与消息。

## 后续提案

- [Appointment 组织只读副本](./proposals/01-appointment-organization-read-model.md)：减少 App API 对 Identity 与
  Appointment 的页面级拼接。该提案尚未实施，不影响当前功能。
- [Guidance 智能导诊业务逻辑](./proposals/02-guidance-service.md)：独立 Guidance 服务的规划、推荐、预约衔接与
  预约后导航业务闭环。检查项目先后关系和可复用的两点步行路线已经实现；工作人员项目三步配置、禁食/禁水/
  喝水准备规则和非强制患者提醒已经确认，尚未实现。

没有经过确认的新业务不在这里建立“阶段 6”或空模块。新一轮规划应先说明真实用户场景、数据拥有者和
不可逆业务规则，再决定扩展现有服务还是新增服务。
