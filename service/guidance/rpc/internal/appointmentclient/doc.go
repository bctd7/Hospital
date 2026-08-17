// Package appointmentclient 集中放置 Guidance 对 Appointment RPC 的出站适配器。
//
// 这里不保存 Appointment 数据，也不实现 Guidance 业务规则。每个适配器只把
// Appointment 的 RPC 请求和响应转换为对应业务包声明的窄接口：
//   - Configuration：项目完整配置所需的查询和 TCC 参与者；
//   - Planning：智能预约所需的项目、窗口、预约查询和批量预约；
//   - PrecedenceDirectory：先后规则校验所需的只读项目目录。
package appointmentclient
