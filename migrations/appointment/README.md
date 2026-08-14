# Appointment 数据库

`hospital_appointment` 是 Appointment 服务独占的业务数据库。Identity 的科室、院区和账号 ID 只作为跨服务稳定标识保存，不建立跨数据库外键。

当前尚未发布且没有需要保留的业务数据，因此 Appointment 迁移已压平为一份最新初始 Schema：

- `appointment_examination_items`：检查项目及其所属科室；
- `appointment_examination_item_operations`、`appointment_examination_item_audit`：项目操作的幂等结果和审计；
- `appointment_rooms`：严格保存院区、楼栋、楼层和房间号；
- `appointment_room_examination_items`：房间可执行的检查项目；
- `appointment_room_weekly_windows`：房间周开放时间和共享容量；
- `appointment_item_weekly_windows`：项目独立的周预约时间和停止新增时间；
- `appointment_resource_operations`、`appointment_resource_audit`：房间、关系和窗口操作的幂等结果与审计。
- `appointment_bookings`：待检查、检查中、已完成和未到场状态；
- `appointment_patient_weekly_quota_usage`：患者每自然周已经消耗的预约额度；
- `appointment_examination_reports`：一次预约唯一的报告主体和检查资源快照；
- `appointment_examination_report_versions`：可编辑草稿、当前正式版和不可覆盖的历史更正版本。

本地执行：

```powershell
.\scripts\migrate.ps1 -Service appointment -Direction up
.\scripts\migrate.ps1 -Service appointment -Direction version
```

在正式发布前如需继续调整初始结构，可以再次重建本地 Appointment 数据库。发布后不得修改已执行迁移，必须新增版本。
