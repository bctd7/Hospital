# Contracts

本目录保存跨进程、跨团队和跨语言可见的契约源文件。

```text
contracts/
├── api/       # go-zero .api 文件，对外 HTTP 契约；会话接口拆在 identity-session.api
├── authz/     # 权限动作目录及生成后的 Go 常量
├── proto/     # 内部 RPC Protobuf 契约源文件
├── gen/       # Protobuf 生成的共享 Go 类型和客户端
└── events/    # Kafka 事件 Schema
```

规则：

- 先改契约，再生成代码；
- 契约变更需要考虑向后兼容；
- RPC 共享类型和客户端生成到 `contracts/gen`，业务 Logic 仍放在对应服务；
- 不在请求、响应或事件中暴露数据库内部结构；
- API、RPC 和事件都使用明确版本。
