# Guidance 数据库

Guidance 使用独立数据库 `hospital_guidance`。当前压平后的初始迁移包含：

- `guidance_precedence_rules`：工作人员直接配置的项目先后关系，不保存传递推导结果；
- `guidance_precedence_graph_locks`：串行化关系图写入，避免并发请求分别通过循环检查后共同形成循环；
- `guidance_item_configurations`：自然语言说明、禁食/禁水/喝水准备规则和患者提醒；
- `guidance_configuration_transactions`：完整项目创建/更新的 TCC 协调进度、请求指纹和补偿快照；
- `guidance_smart_appointment_plans`：短期智能预约候选及其原子确认结果，用于幂等重试。

```powershell
.\scripts\migrate.ps1 -Service guidance -Direction up
```

表中只保存 Appointment 项目和科室的稳定 ID 与必要名称快照，不建立跨数据库外键，也不直接读取 Appointment
数据库；所有项目事实和预约写入均通过 Appointment gRPC 完成。
