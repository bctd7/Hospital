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

正式迁移工具采用 `golang-migrate` v4，由仓库中的 Go 迁移执行器锁定依赖版本。Compose 只负责本地
MySQL 进程和全新数据卷的数据库账号引导，表结构统一通过 `scripts/migrate.ps1` 执行。

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
- 已经被共享环境执行的迁移文件不再改写，通过新版本迁移修正；
- 业务迁移使用服务数据库账号，不使用 MySQL root 账号。
