# 本地开发脚本

`start-backend.ps1` 从项目 `.env` 读取配置，依次构建并在后台启动 Identity、Appointment、Guidance RPC 与
App API。日志写入系统临时目录 `hospital-backend`。

```powershell
.\scripts\development\start-backend.ps1
.\scripts\development\start-backend.ps1 -Restart
```

不带 `-Restart` 时，任一端口已被占用都会停止执行；带参数时也只会终止预期的 Hospital 后端进程，不会清理数据库。
