# 数据库迁移

本目录是数据库结构变更的版本事实来源。README 只说明当前迁移工具和执行方式；业务数据设计写在对应模块
Plan，表字段以 SQL 为准。

```text
migrations/
└── <service>/
    ├── 000001_name.up.sql
    └── 000001_name.down.sql
```

当前 Identity 与 Appointment 各自拥有独立数据库，并均已在首次发布前压平为单一最新初始版本：

- `identity/000001_identity_initial_schema`：12 张业务表；
- `appointment/000001_appointment_initial_schema`：17 张业务表。

`schema_migrations` 由迁移工具维护，不属于业务模型。仓库中已经不存在需要按顺序回放的旧业务迁移。

## 执行器

迁移工具位于 `tools/db-migrate`，使用仓库锁定的 `golang-migrate`。Compose 只创建数据库和服务账号，
不会替代版本化迁移。

```powershell
.\scripts\migrate.ps1 -Service identity -Direction up
.\scripts\migrate.ps1 -Service identity -Direction version
.\scripts\migrate.ps1 -Service appointment -Direction up
.\scripts\migrate.ps1 -Service appointment -Direction version
```

本地首次初始化使用：

```powershell
.\scripts\db-bootstrap-local.ps1
```

## 约束

- 已在共享环境执行的迁移不得改写；
- 修复结构必须新增版本；
- 业务迁移使用服务账号，不使用 MySQL root；
- 迁移同时提供经过验证的 `up` 和 `down`；
- 破坏性变化使用扩展、迁移数据、切换读取、删除旧结构的分阶段方案；
- 生产迁移前备份，迁移失败时不得启动不兼容服务；
- 新服务拥有独立迁移目录、数据库账号和 CI 空库验证。

Identity 当前表说明见 [identity/README.md](./identity/README.md)。
