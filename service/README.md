# 后端服务

本目录保存可独立构建和部署的 Go 服务。当前包含面向微信小程序的 `app-api`，后续仅在业务边界、数据所有权、发布节奏或扩缩容要求明确时增加 RPC 服务或异步 Consumer。

## API 服务结构

```text
service/app/api/
├── app.go                    # 服务启动入口
├── etc/                      # 运行配置
└── internal/
    ├── config/               # 配置结构
    ├── handler/              # HTTP 协议适配
    ├── logic/                # 业务用例与规则
    ├── svc/                  # 共享依赖装配
    └── types/                # API 请求响应类型
```

API 路由和类型以 `contracts/api/app.api` 为源，通过 goctl 生成：

```powershell
goctl api validate -api contracts/api/app.api

goctl api go `
  -api contracts/api/app.api `
  -dir service/app/api `
  --style go_zero

go mod tidy
go build ./...
```

生成文件进入版本控制。业务实现位于 `internal/logic/`，共享依赖在 `internal/svc/` 中构造；不要直接修改标有 `DO NOT EDIT` 的生成文件。
