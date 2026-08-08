# Contracts

本目录保存跨进程、跨团队和跨语言可见的契约源文件。

```text
contracts/
├── api/       # go-zero .api 文件，对外 HTTP 契约
├── proto/     # 后续增加的内部 RPC Protobuf 契约
└── events/    # Kafka 事件 Schema
```

规则：

- 先改契约，再生成代码；
- 契约变更需要考虑向后兼容；
- 生成的 Go 代码放进对应服务，不放在本目录；
- 不在请求、响应或事件中暴露数据库内部结构；
- API、RPC 和事件都使用明确版本。
