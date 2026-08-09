# Database Migrations

数据库结构必须由版本化迁移管理，不能只依赖 ORM、goctl 或容器初始化脚本自动建表。

迁移按业务数据归属放置：

```text
migrations/
└── <domain>/
    ├── 000001_init.up.sql
    └── 000001_init.down.sql
```

当前 Identity 迁移：

- `000001_identity_authorization`：账号授权事实、审计与 Outbox；
- `000002_identity_login`：微信外部身份映射和手机号登录数据。

正式迁移执行器位于 `tools/db-migrate`，使用仓库锁定的 `golang-migrate` v4。Compose 只负责创建数据库和服务账号，表结构统一由迁移执行器更新。

本地常用命令：

```powershell
.\scripts\migrate.ps1 -Service identity -Direction up
.\scripts\migrate.ps1 -Service identity -Direction version
```

生产部署由 `deploy/production` 中的 `identity-migrate` 一次性任务执行 `up`。每次生产迁移前必须先备份；迁移失败时，Identity RPC 和 App API 不会继续启动为不兼容版本。

约束：

- 已在共享环境执行的迁移文件不得改写，修复必须新增版本；
- 业务迁移使用服务数据库账号，不使用 MySQL root；
- 破坏性变化采用分阶段兼容方案；
- 时间默认按 UTC 保存，索引由真实查询驱动；
- 回滚或基线登记必须显式执行，并先核对数据库实际结构。
