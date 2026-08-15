# 跨服务测试

当前 Go 单元测试和数据库/Redis 集成测试与对应包放在一起，小程序测试位于 `apps/miniapp/tests/`。
本目录暂不保存重复测试套件，只为未来真正跨多个独立服务的契约或端到端测试预留入口。

## 当前测试位置

| 类型 | 位置 |
|---|---|
| Go 单元测试 | 与被测包同目录的 `*_test.go` |
| Identity MySQL 集成测试 | `service/identity/rpc/internal/repository/mysqlstore/*_integration_test.go` |
| Redis Session 集成测试 | `service/identity/rpc/internal/repository/redisstore/session_integration_test.go` |
| Appointment MySQL 集成测试 | `service/appointment/rpc/internal/repository/mysqlstore/*_integration_test.go` |
| HTTP Logic/错误映射 | `service/app/api/internal/**/**/*_test.go` |
| 小程序单元测试 | `apps/miniapp/tests/` |
| 全量工程检查 | `scripts/check.ps1` |

只有需要同时启动多个独立服务、且无法归属某个服务包的测试才进入本目录。不要把服务内部测试为了“看起来
整齐”搬离包目录。
