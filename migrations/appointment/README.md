# Appointment 数据库

`hospital_appointment` 是 Appointment 服务独占的业务数据库。Identity 的科室和账号 ID 只作为跨服务稳定标识保存，
不得建立跨数据库外键。

当前迁移创建：

- `appointment_examination_items`：检查项目当前状态；
- `appointment_examination_item_operations`：完整幂等结果和请求指纹；
- `appointment_examination_item_audit`：不包含完整描述正文的变更审计摘要。
- `appointment_rooms`：科室检查房间，强约束保存院区、楼栋、楼层和房间号；
- `appointment_room_examination_items`：房间与本部门检查项目关系；
- `appointment_room_weekly_windows`：房间周开放时间与共享容量；
- `appointment_item_weekly_windows`：项目周预约时间与停止新增时间；
- `appointment_resource_operations`、`appointment_resource_audit`：第二阶段资源幂等结果和审计摘要。

本地执行：

```powershell
.\scripts\migrate.ps1 -Service appointment -Direction up
.\scripts\migrate.ps1 -Service appointment -Direction version
```
