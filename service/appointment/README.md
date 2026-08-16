# Appointment 服务

Appointment 是检查资源、预约、容量、检查执行、报告和消息的数据拥有者。当前规划范围已经实现，完整业务
规则见 [Appointment 已实现归档](../../plan/backend/modules/implemented/02-appointment-service.md)。

## 当前能力

- 科室项目、严格地址房间、房间—项目关系，以及相互独立的房间开放窗口和项目预约窗口；
- 按房间、具体日期和上午/下午共享容量，患者只预约本周并受重复预约和每周 10 次额度约束；
- `confirmed`、`queued`、`called`、`in_progress`、`report_pending`、`completed`、`no_show` 和
  `canceled` 八种预约状态；
- 患者检查报到、房间共享候检队列、一分钟叫号、过号顺延和窗口结束后的未到场处理；
- 工作人员叫下一位、确认开始检查、结束检查并立即释放资源；
- 项目报告模板、报告草稿、正式发布和不可覆盖的更正版本；
- 患者与科室消息以及逐账号已读状态。

医生是授权操作者，不是预约资源。患者自行选择房间，系统不绑定医生、不计算医生容量，也不自动挑选最优
科室或房间。

## 缓存

资源热点使用 Redis Cache Aside：常规资源 5 分钟并带随机抖动，预约候选 20 秒；同实例合并并发回源，
写后递增科室 generation。Redis 失败回源 MySQL。

容量扣减、患者项目占用、每周额度、候检轮次和状态迁移始终在 MySQL 事务内复核。缓存不参与预约授权或
叫号裁决，因此不会导致超卖或重复叫号。

## 运行与验证

```powershell
.\scripts\migrate.ps1 -Service appointment -Direction up
.\scripts\start-backend.ps1 -Restart
go test ./service/appointment/...
go test ./service/app/api/...
goctl api validate -api contracts/api/app.api
```
