# 公共组件

本目录只保存已经被多个服务复用、边界稳定的技术能力，不保存具体业务 Logic 或共享数据库 Model。

## 当前包

| 包 | 已实现职责 |
|---|---|
| `authn` | Principal、Ed25519 JWT、Context、HTTP/gRPC 鉴权和授权版本抽象 |
| `authn/versionredis` | Redis 授权版本 Reader/Writer 实现 |
| `authz` | permission 判断和组织范围等通用授权规则 |
| `observability/logging` | 结构化业务事件、Request ID 和公共字段 |
| `observability/httpaccess` | 不读取正文的 HTTP 访问日志中间件 |

## 边界

允许放入：

- 多个服务必须保持一致的协议无关技术实现；
- 不依赖任何 `service/*/internal` 的稳定抽象；
- 有独立测试和清晰失败语义的组件。

禁止放入：

- Identity、Appointment 等具体领域规则；
- 跨服务共享数据库实体；
- 只为了减少几行重复代码的临时工具；
- Token 私钥、数据库连接或供应商 Secret；
- 由某个业务服务拥有的写操作。

授权版本遵循 Identity 单写、其他服务只读。公共包可以提供 Reader/Writer 接口，但只有 Identity 负责更新
账号当前授权版本。
