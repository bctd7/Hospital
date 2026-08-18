# 仓库脚本

`scripts/` 是开发、生成、数据库和质量检查的人工入口。目录按操作目的划分，不再把所有 PowerShell 与大段测试
SQL 平铺在同一级。

```text
scripts/
├─ contracts/
│  └─ generate-protobuf.ps1          # 验证或生成 gRPC 契约
├─ database/
│  ├─ bootstrap-local.ps1            # 初始化本地数据库和服务账号
│  ├─ migrate.ps1                    # 统一迁移入口
│  ├─ validate-flat-migrations.ps1   # 校验单一初始版本与灌数表引用
│  ├─ seed-comprehensive-test-data.ps1
│  └─ seed-comprehensive-test-data.sql
├─ development/
│  └─ start-backend.ps1              # 构建并启动四个后端进程
├─ quality/
│  └─ verify-repository.ps1          # 契约、Go、Compose 和小程序总检查
└─ lib/
   └─ environment.ps1                # PowerShell 共用的 UTF-8 环境读取
```

## 常用入口

```powershell
Copy-Item .env.example .env
.\scripts\database\bootstrap-local.ps1
.\scripts\database\seed-comprehensive-test-data.ps1 -Reset
.\scripts\development\start-backend.ps1 -Restart
.\scripts\quality\verify-repository.ps1
```

## 组织规则

- 人工工作流放 `scripts/`，可被镜像复用的独立 Go 命令放 `tools/`；
- 跨脚本共用代码只放 `lib/`，业务 SQL 不进入通用库；
- 脚本必须从自身位置解析仓库根目录，不能依赖调用者当前目录；
- 失败必须返回非零退出码，破坏性数据库操作必须使用显式 `-Reset` 或确认参数；
- 不保留仅为旧文件路径服务的兼容脚本；修改入口后同步仓库文档和部署调用方。

详细说明：

- [契约生成](./contracts/README.md)
- [数据库与综合测试数据](./database/README.md)
- [本地后端启动](./development/README.md)
- [统一质量检查](./quality/README.md)
- [Go 工具](../tools/README.md)
