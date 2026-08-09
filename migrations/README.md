# Database Migrations

数据库结构必须由迁移文件管理，不能只依赖 ORM 或 goctl 自动建表。

迁移按业务数据归属创建目录：

```text
migrations/
└── <domain>/
    ├── 000001_init.up.sql
    └── 000001_init.down.sql
```

当前已经建立 `identity/000001_identity_authorization`，用于 Identity Service 的账号授权事实、审计和 Outbox。正式迁移工具仍需在 Goose、Atlas 或 golang-migrate 中确定一个；本地全新 MySQL 卷会通过 Compose 初始化脚本执行首个迁移。

约束：

- 迁移文件进入版本控制；
- 正式环境迁移前备份；
- 破坏性变更采用分阶段兼容方案；
- 每张表有明确的数据拥有者；
- 时间默认以 UTC 保存；
- 索引由真实查询驱动。
