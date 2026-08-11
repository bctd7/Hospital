# 接口契约

本目录是跨进程、跨团队和跨语言接口的事实来源，不保存业务规划或运行说明。

```text
contracts/
├── api/       # 对外 HTTP 的 go-zero .api 源文件
├── proto/     # 内部 gRPC Protobuf 源文件
├── gen/       # Protobuf 生成的共享 Go 类型和 Client
├── authz/     # permission 目录与生成 Go 常量
└── events/    # Outbox/Kafka 事件 Schema
```

## 当前入口

- HTTP：`contracts/api/app.api`；
- Identity RPC：`contracts/proto/identity/v1/identity.proto`；
- 权限目录：`contracts/authz/permissions.yaml`；
- 事件信封：`contracts/events/event-envelope.schema.json`；
- 生成的 HTTP 文档：`docs/api/openapi.json`。

## 修改规则

1. 先修改契约源文件，再生成代码；
2. Handler、Types、Proto Go 文件和 RPC Client 等生成物不得单独手改；
3. 公开命名使用业务资源，不暴露表名、SQL 字段或内部 Manager；
4. 新字段优先向后兼容，删除或改变语义需要显式版本策略；
5. HTTP、RPC 和事件使用独立边界，不复用数据库 Model；
6. 手机号、验证码、Token 和 Secret 不写入示例；
7. 契约变更必须同步测试和 Swagger 快照。

## 校验

```powershell
goctl api validate -api contracts/api/app.api
goctl api swagger --api contracts/api/app.api --dir docs/api --filename openapi
go test ./contracts/...
```

业务目的、状态机和完成范围写在 `plan/`；本地启动写在对应 README；字段、路径和方法只在本目录维护。
