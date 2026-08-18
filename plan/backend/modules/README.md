# 后端模块状态

这里按业务模块维护“当前已经具备什么”和“后续准备改变什么”，不按 HTTP/RPC 接口、数据表或代码包建立章节。
逐接口字段以 `contracts/` 为准，代码导航和运行方式以对应 `service/*/README.md` 为准。

## 已实现

- [Identity 当前范围](./implemented/01-identity-service.md)：认证、会话、授权版本、组织、账号和医生管理；
- [Appointment 当前范围](./implemented/02-appointment-service.md)：检查资源、预约、患者报到、房间候检叫号、检查、报告与消息。
- [Guidance 当前范围](./implemented/03-guidance-service.md)：检查规则、智能预约、整组确认、当日顺序和分阶段地图路线。

## 后续提案

- [Appointment 组织只读副本](./proposals/01-appointment-organization-read-model.md)：减少 App API 对 Identity 与
  Appointment 的页面级拼接。该提案尚未实施，不影响当前功能。
没有经过确认的新业务不在这里建立“阶段 6”或空模块。新一轮规划应先说明真实用户场景、数据拥有者和
不可逆业务规则，再决定扩展现有服务还是新增服务。
