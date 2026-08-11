# Scripts

本目录保存已经实现、可重复执行且失败时返回非零退出码的开发脚本。

| 脚本 | 用途 |
|---|---|
| `start-backend.ps1` | 按 UTF-8 加载 `.env`，构建并启动 Identity RPC 与 App API |
| `db-bootstrap-local.ps1` | 启动本地 MySQL、幂等创建数据库/账号并执行迁移 |
| `migrate.ps1` | 执行升级、受限回滚、版本查询和显式基线登记 |
| `check.ps1` | 校验契约、Compose、Go 和微信小程序 |
| `lib/environment.ps1` | 脚本共享的 UTF-8 环境变量加载与校验 |

## 常用命令

```powershell
Copy-Item .env.example .env
.\scripts\db-bootstrap-local.ps1
.\scripts\start-backend.ps1 -Restart
.\scripts\migrate.ps1 -Service identity -Direction version
.\scripts\check.ps1
```

`start-backend.ps1` 会覆盖当前进程继承的同名变量，以 `.env` 的 UTF-8 值为准，避免 Windows PowerShell
错误解码阿里云中文签名。运行日志写到系统临时目录 `hospital-backend`。

基线登记只适用于已经人工确认结构与目标迁移完全一致的旧数据库。它只写迁移版本，不创建或修复表，必须
显式提供 `-AcknowledgeExistingSchema`。

生产部署、备份和换服使用 `deploy/production` 中的脚本。任何脚本都不得内置真实密码、AccessKey、
AppSecret 或 Token 私钥。
