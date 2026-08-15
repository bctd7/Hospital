# Scripts

本目录保存已经实现、可重复执行且失败时返回非零退出码的开发脚本。

| 脚本 | 用途 |
|---|---|
| `start-backend.ps1` | 按 UTF-8 加载 `.env`，构建并启动 Identity RPC 与 App API |
| `seed-comprehensive-test-data.ps1` | 向已经执行最新版迁移的本地 Identity 与 Appointment 数据库写入综合联调数据 |
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

综合测试数据仅允许在 `APP_ENV=local` 使用。它包含多院区、多科室、医生、患者、检查资源、完整预约状态、
预约与科室消息、草稿、正式及更正报告。首次向空库写入可直接执行；需要清理旧测试数据并按压平后的
`000001` 迁移重建时，显式使用 `-Reset`。该模式也会清空本地 Redis 中的登录会话、授权版本和 Appointment 缓存，
避免固定测试账号复用旧缓存。

```powershell
.\scripts\seed-comprehensive-test-data.ps1
.\scripts\seed-comprehensive-test-data.ps1 -Reset
```

核心体验账号和展示范围：

- `13482154556`：本地超级管理员，可切换四个科室，查看全部预约状态及科室消息；
- `15363658538`：报告患者“张明”，可看到当日待检查、两次到院提醒、正式报告、报告更正和已读/未读效果；
- `13800000001`、`13800000002`：放射科和超声科医生，本地短信模拟环境用于验证医生固定科室；
- `13900000002` 至 `13900000004`：用于取消、未到场、检查中、报告草稿及报告超时场景。

资源数据包含两个院区、四个科室、六个严格地址房间、七个项目，以及同项目多房间、同房间多项目、停用项目、
七天上午/下午窗口和日期容量。预约数据覆盖 `confirmed`、`in_progress`、`completed`、`no_show`、`canceled`；
科室消息能够展示新预约、取消、未到场、报告到期和报告超时。

基线登记只适用于已经人工确认结构与目标迁移完全一致的旧数据库。它只写迁移版本，不创建或修复表，必须
显式提供 `-AcknowledgeExistingSchema`。

生产部署、备份和换服使用 `deploy/production` 中的脚本。任何脚本都不得内置真实密码、AccessKey、
AppSecret 或 Token 私钥。
