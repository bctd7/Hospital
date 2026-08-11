# Go 服务

本目录只保存可以独立构建和运行的 Go 服务。当前已经落地两个服务：

| 服务 | 入口 | 端口 | 职责 |
|---|---|---:|---|
| App API | `service/app/api/app.go` | 8888 | 小程序 HTTP、Access Token 中间件、协议转换和页面聚合 |
| Identity RPC | `service/identity/rpc/identity.go` | 8080 | 登录、会话、账号、权限、组织和医生领域 |

```text
Miniapp
  -> App API
  -> Identity RPC
  -> MySQL / Redis / Aliyun PNVS
```

## App API

```text
service/app/api/
├── app.go
├── etc/
└── internal/
    ├── config/       # 配置结构
    ├── handler/      # HTTP 协议适配和路由
    ├── logic/        # HTTP 用例适配、RPC 调用
    ├── middleware/   # Access Token 等 HTTP 中间件
    ├── svc/          # RPC Client、Redis 等依赖装配
    └── types/        # goctl 生成的 HTTP 类型
```

App API 不拥有 Identity 数据，不直接访问 `hospital_identity`。需要账号、组织或医生数据时调用 Identity RPC。

## Identity RPC

详细能力、分层和扩展规则见 [Identity Service](./identity/README.md)。内部代码阅读入口见
[`identity/rpc/internal/README.md`](./identity/rpc/internal/README.md)。

## 契约与生成

- HTTP：`contracts/api/app.api`；
- RPC：`contracts/proto/identity/v1/identity.proto`；
- 生成代码：`contracts/gen/`、`identityservice/`、Handler、Types 和 Server 骨架。

```powershell
goctl api validate -api contracts/api/app.api
goctl api go -api contracts/api/app.api -dir service/app/api --style go_zero
```

生成文件进入版本控制，但不能单独手改标记为 `DO NOT EDIT` 的文件。业务逻辑放在 Logic/Manager，依赖创建放在
ServiceContext。

## 启动与检查

```powershell
.\scripts\start-backend.ps1 -Restart
.\scripts\check.ps1
```

## 新增服务的条件

只有同时具备明确的数据所有权、独立发布边界和真实调用方时才增加新服务。新增服务必须同步完成：

1. Plan 中的业务与服务边界；
2. 独立迁移目录和数据库账号；
3. HTTP/RPC/事件契约；
4. 认证拦截器、授权版本 Reader 和可观测性；
5. 本地 Compose、生产部署、CI 和测试；
6. 本目录服务索引。

未实现的 Appointment、Planning 等服务不在这里提前创建空目录。
