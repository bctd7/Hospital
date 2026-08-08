# Tests

测试目录用于跨包、跨服务的集成测试和契约测试。服务内部单元测试仍与对应 Go 文件放在一起。

```text
tests/
├── contract/       # API、Proto、事件兼容性
└── integration/    # MySQL、Redis、Kafka 真实链路
```

工程基线至少验证：

- `.api` 可以通过 goctl 校验；
- Health Logic 返回预期结构；
- 未知路由和非法参数返回一致错误；
- Docker Compose 配置可解析；
- 后续数据库迁移能在空库执行。
