# 后端模块状态

## 已实现

- [Identity 首期](./implemented/01-identity-service.md)：认证、会话、授权版本、组织、账号和医生管理；
- [Appointment 当前范围](./implemented/02-appointment-service.md)：检查资源、预约、容量、检查、报告与消息。

## 后续提案

- [Appointment 组织只读副本](./proposals/01-appointment-organization-read-model.md)：减少 App API 对 Identity 与
  Appointment 的页面级拼接。该提案尚未实施，不影响当前功能。
- [Appointment 检查项目预计时长](./proposals/02-appointment-examination-duration.md)：为项目增加规划时长并在预约中
  保存快照，是智能导诊前需要完成的 Appointment 收尾项。

没有经过确认的新业务不在这里建立“阶段 6”或空模块。新一轮规划应先说明真实用户场景、数据拥有者和
不可逆业务规则，再决定扩展现有服务还是新增服务。
