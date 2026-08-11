# RPC Contracts

本目录保存内部同步调用的 Protobuf 契约。当前已实现
`identity/v1/identity.proto`，由 App API 调用 Identity RPC。

## 规则

1. 只有明确的跨服务同步边界才增加 RPC；
2. 请求和响应表达业务语义，不直接暴露数据库 Model；
3. 兼容变更新增字段号，已经发布的字段号不得复用；
4. 认证由统一 gRPC Interceptor 执行，业务方法仍在 Manager 中校验 permission；
5. 公开目录、本人查询和管理员查询使用不同响应，避免字段泄漏；
6. 生成的 `contracts/gen` 与 `identityservice` 包不得单独手改；
7. 新敏感方法同步加入 RPC 正文日志屏蔽名单。

Proto 的 `go_package` 固定指向共享生成包。具体生成参数以仓库当前 goctl/protoc 工具链为准，生成后必须运行：

```powershell
gofmt -w contracts/gen service/identity/rpc/identityservice
go test ./contracts/gen/... ./service/identity/rpc/...
```

HTTP 接口不要直接从 Proto 推导；对外契约仍在 `contracts/api/`。
