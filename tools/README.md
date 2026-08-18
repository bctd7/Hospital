# 仓库工具

`tools/` 保存会被本地脚本或生产镜像调用的 Go 命令，不保存服务业务代码。每个叶子目录都是一个独立的
`package main`，名称直接表达操作对象和动作。

```text
tools/
├─ database/
│  └─ migrate/          # 执行版本化 MySQL 迁移或查询当前版本
└─ identity/
   ├─ bootstrap-admin/  # 初始化唯一医院根节点与配置中的超级管理员
   └─ token-keygen/     # 生成 Identity Access Token 使用的 Ed25519 密钥对
```

## 使用边界

- 日常数据库操作从 `scripts/database/migrate.ps1` 进入，不直接拼接迁移器参数；
- `bootstrap-admin` 是部署和体验数据初始化的一次性幂等任务，不是后台账号管理接口；
- `token-keygen` 只把新密钥输出到当前终端，调用者自行写入安全环境，不提交结果；
- 工具不得引用 App API Handler，也不得绕过服务实现日常业务写入；
- 新工具必须有单一用途、非零失败码、参数或环境变量说明以及最小测试。

## 直接运行

```powershell
go run ./tools/identity/token-keygen
go test ./tools/...
```

数据库迁移器和管理员初始化器需要环境变量，具体入口与安全限制见
[`scripts/database/README.md`](../scripts/database/README.md)。
