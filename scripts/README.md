# Scripts

本目录保存可重复执行、非交互且失败时返回非零退出码的开发、校验和运维脚本。

- `check.ps1`：校验 API、事件、Compose、Go 和微信小程序；
- `db-bootstrap-local.ps1`：仅在本地启动 MySQL、幂等创建服务数据库和账号，并执行迁移；
- `migrate.ps1`：按服务执行数据库升级、受限回滚、版本查询或旧库基线登记；
- `lib/environment.ps1`：供脚本复用的环境变量读取与校验逻辑。

本地首次准备数据库：

```powershell
Copy-Item .env.example .env
# 检查并修改 .env 后执行
.\scripts\db-bootstrap-local.ps1
```

日常迁移和全量检查：

```powershell
.\scripts\migrate.ps1 -Service identity -Direction up
.\scripts\migrate.ps1 -Service identity -Direction version
.\scripts\check.ps1
```

旧数据库如果已经执行 Identity `000001` 和 `000002`，但没有迁移版本表，必须先核对表结构，再进行一次基线登记：

```powershell
.\scripts\migrate.ps1 `
    -Service identity `
    -Direction baseline `
    -BaselineVersion 2 `
    -AcknowledgeExistingSchema
```

基线登记只写入迁移版本，不执行建表 SQL，也不会修复不完整的旧数据库；数据库已有迁移版本时会拒绝覆盖。

服务代码生成以 `contracts/` 中的契约为输入。生产部署、备份和换服流程位于 `deploy/production`，不得把真实密钥写入脚本或 Git。
