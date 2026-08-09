# Database Migrations

数据库结构必须由迁移文件管理，不能只依赖 ORM 或 goctl 自动建表。

迁移按业务数据归属创建目录：

```text
migrations/
└── <domain>/
    ├── 000001_init.up.sql
    └── 000001_init.down.sql
```

当前已经建立：

- `identity/000001_identity_authorization`：账号授权事实、审计和 Outbox；
- `identity/000002_identity_login`：微信外部身份绑定和用户自报手机号。

正式迁移工具采用 `golang-migrate` v4，由仓库中的 `tools/db-migrate` 锁定依赖版本。Compose 只负责创建 MySQL 数据库和服务账号，表结构统一通过迁移执行器更新。

```powershell
.\scripts\migrate.ps1 -Service identity -Direction up
.\scripts\migrate.ps1 -Service identity -Direction version
```

约束：

- 迁移文件进入版本控制；
- 正式环境迁移前备份；
- 破坏性变更采用分阶段兼容方案；
- 每张表有明确的数据拥有者；
- 时间默认以 UTC 保存；
- 索引由真实查询驱动。
