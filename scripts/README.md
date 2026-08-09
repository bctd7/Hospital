# Scripts

本目录保存可重复执行的开发、校验和运维脚本。

- `check.ps1`：执行 API、事件、Compose、Go 和微信小程序全量检查；
- `db-bootstrap-local.ps1`：仅在本地启动 MySQL、幂等创建服务数据库和账号，并执行迁移；
- `migrate.ps1`：按服务执行数据库升级、回滚、版本查询或一次性基线登记；
- `check-foundation.ps1`：兼容旧命令，内部直接调用 `check.ps1`。

本地首次准备数据库：

```powershell
Copy-Item .env.example .env
# 检查并修改 .env 后执行
.\scripts\db-bootstrap-local.ps1
```

日常迁移和检查：

```powershell
.\scripts\migrate.ps1 -Service identity -Direction up
.\scripts\migrate.ps1 -Service identity -Direction version
.\scripts\check.ps1
```

旧数据库如果已经执行了 Identity 的 `000001` 和 `000002`，但没有迁移版本表，必须先核对表结构，
再进行一次基线登记：

```powershell
.\scripts\migrate.ps1 `
    -Service identity `
    -Direction baseline `
    -BaselineVersion 2 `
    -AcknowledgeExistingSchema
```

基线登记只写入迁移版本，不执行建表 SQL，也不会自动修复不完整的旧数据库。数据库已经存在迁移版本时，
该命令会拒绝覆盖。

服务代码生成以 `contracts/` 中的契约为输入。代码生成、数据库迁移和发布脚本应保持非交互、可重复执行，并在失败时返回非零退出码。
