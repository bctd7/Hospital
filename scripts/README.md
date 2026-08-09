# Scripts

本目录保存可重复执行的开发、校验和运维脚本。

- `db-bootstrap-local.ps1`：幂等创建本地 Identity 数据库和账号，并执行未运行迁移；
- `migrate.ps1`：执行数据库升级、版本查询、受限回滚或旧库基线登记；

- `check-foundation.ps1`：校验 API 契约、事件 JSON、Compose 配置和现有 Go 代码。

服务代码生成以 `contracts/` 中的契约为输入。代码生成、数据库迁移和发布脚本应保持非交互、可重复执行，并在失败时返回非零退出码。
