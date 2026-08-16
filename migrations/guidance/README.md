# Guidance 数据库

Guidance 使用独立数据库 `hospital_guidance`。当前初始迁移只包含检查项目直接先后关系及其并发图锁：

- `guidance_precedence_rules`：工作人员直接配置的项目先后关系，不保存传递推导结果；
- `guidance_precedence_graph_locks`：串行化关系图写入，避免并发请求分别通过循环检查后共同形成循环。

```powershell
.\scripts\migrate.ps1 -Service guidance -Direction up
```

表中只保存 Appointment 项目和科室的稳定 ID 与名称快照，不建立跨数据库外键，也不直接读取 Appointment 数据库。
