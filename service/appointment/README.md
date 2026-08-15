# Appointment 服务

Appointment 是检查资源、预约、容量、检查执行、报告和消息的数据拥有者。当前规划范围已经实现，完整业务
规则见 [Appointment 已实现归档](../../plan/backend/modules/implemented/02-appointment-service.md)。

## 当前能力

- 一个项目属于一个科室，一个房间属于一个科室；
- 房间与本科室项目多对多关联；
- 房间开放窗口与项目预约窗口独立配置；
- 按房间、具体日期和上午/下午共享容量；
- 患者本周预约、重复预约限制、每周 10 次额度和取消；
- 待检查、检查中、已完成、未到场和已取消状态；
- 科室预约查询、时间范围内开始检查；
- 项目报告模板、报告草稿、完成发布和不可覆盖的更正版本；
- 患者与科室消息以及逐账号已读状态。

医生是授权操作者，不是预约资源。患者自行选择房间，系统不绑定医生、不计算医生容量，也不自动挑选最优
科室或房间。

## 缓存

资源热点使用 Redis Cache Aside：常规资源 5 分钟并带随机抖动，预约候选 20 秒；同实例合并并发回源，
写后递增科室 generation。Redis 失败回源 MySQL。

容量扣减、患者时段、每周额度和状态迁移始终在 MySQL 事务内复核。缓存不参与预约授权，因此不会导致超卖。

## 运行与验证

```powershell
.\scripts\migrate.ps1 -Service appointment -Direction up
.\scripts\start-backend.ps1 -Restart
go test ./service/appointment/...
go test ./service/app/api/...
goctl api validate -api contracts/api/app.api
```
