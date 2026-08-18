# RPC 契约

本目录保存内部同步调用的 Protobuf 契约。每个服务目录都采用“一个服务入口 + 多个能力消息文件”：入口文件只
声明 RPC 方法并用中文分组，能力文件保存请求、响应和共享消息。

```text
identity/v1/       identity.proto + authentication/profile/organization/account_admin
appointment/v1/    appointment.proto + catalog/resources/bookings/messages/reports
guidance/v1/       guidance.proto + precedence/configuration/planning/routing
```

拆分文件不会拆成多个 gRPC Service，也不会改变 Go package；它只让契约阅读者可以按能力定位消息。

## 规则

1. 只有明确的跨服务同步边界才增加 RPC；
2. 请求和响应表达业务语义，不直接暴露数据库 Model；
3. 兼容变更新增字段号，已经发布的字段号不得复用；
4. 认证由统一 gRPC Interceptor 执行，业务方法仍在 Manager 中校验 permission；
5. 公开目录、本人查询和管理员查询使用不同响应，避免字段泄漏；
6. 生成的 `contracts/gen` 与各 RPC Client 包不得单独手改；
7. 新敏感方法同步加入 RPC 正文日志屏蔽名单。

Proto 的 `go_package` 固定指向共享生成包。生成参数已经收口到仓库脚本：

```powershell
.\scripts\contracts\generate-protobuf.ps1
go test ./contracts/gen/... ./service/identity/rpc/... ./service/appointment/rpc/... ./service/guidance/rpc/...
```

HTTP 接口不要直接从 Proto 推导；对外契约仍在 `contracts/api/`。
