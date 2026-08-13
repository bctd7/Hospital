# Hospital 后端技术架构基线

> 状态：当前工程技术基线

## 1. 技术选型

| 层次 | 当前选择 |
|---|---|
| 对外 API | Go + go-zero REST，契约位于 `contracts/api` |
| 内部 RPC | go-zero zRPC / gRPC，契约位于 `contracts/proto` |
| 数据库 | MySQL 8.4，服务拥有独立逻辑数据库或 Schema |
| 会话与缓存 | Redis |
| 事件总线 | Kafka |
| 可靠事件 | 本地事务 Outbox + 发布 Worker |
| 部署 | Docker Compose 起步，服务保持独立配置和进程边界 |
| 可观测性 | 结构化日志、request ID、Trace、Metrics、业务审计 |

代码在 Monorepo 中维护，统一依赖和 CI，但不允许通过共享仓库绕过服务边界或跨库写入。

## 2. 调用关系

```text
微信小程序
  -> app-api (HTTP JSON)
      -> identity-rpc
      -> appointment-rpc
      -> navigation-rpc
      -> 当前仍以内置模块交付的 Patient / Report / Message

业务数据库事务
  -> outbox
  -> Kafka
  -> 下游消费者 / 通知 / 投影
```

产品唯一客户端是微信小程序，不规划 Web/H5 管理后台或另一套原生 App。患者、医生和超级管理员在
同一小程序内按身份与 permission 切换页面能力。

微信小程序只知道 App API，不访问内部 RPC、数据库、Redis 或 Kafka。页面聚合由 App API 完成，不能
让前端理解微服务拓扑。

## 3. 工程结构原则

```text
apps/miniapp/                 # 小程序
service/app/api/              # 对外 HTTP 聚合层
service/identity/rpc/         # Identity RPC
service/<domain>/rpc/         # 后续独立业务服务
contracts/api/                # HTTP 契约
contracts/proto/              # 内部 RPC 契约
common/authn, common/authz/   # 身份验证与通用授权
migrations/<domain>/          # 数据所有者的迁移
tools/                        # 部署或受控维护工具
```

- 生成文件通过 goctl/protoc 更新，不手工编辑；
- Handler 只做协议解析，Logic 编排用例，Repository 负责数据访问；
- 配置分环境维护，Secret 只从环境或 Secret Manager 注入；
- 共享代码只包含稳定横切能力，不能把业务数据访问放进 `common`；
- 未达到独立部署、数据所有权或安全边界的模块先保留在现有服务内部。

## 4. 数据与一致性

- 单服务写操作在本地数据库事务内完成；
- 服务不能直接写其他服务数据库；
- 同步查询或命令使用 RPC，并设置超时和明确错误；
- 状态传播使用 Outbox + Kafka，消费者必须幂等；
- 不使用分布式事务锁住多个服务数据库；
- 聚合页面允许短暂最终一致，核心预约占号必须在 Appointment 本地强一致；
- 所有核心对象使用稳定内部 ID，外部医院编号放映射表；
- 当前状态、状态历史和必要快照分开保存。

详细所有权见 [服务与数据边界](./02-service-and-data-boundaries.md)。

## 5. 身份与安全

当前唯一认证方式是手机号 + 阿里云 PNVS 验证码。
Identity 签发短期 Access Token 和轮换 Refresh Token；App API 与业务服务本地验证 Access Token。

- 密码、AccessKey、HMAC Key、JWT 私钥不进入 Git；
- 验证码、完整手机号、Token 和患者敏感信息不写普通日志；
- 短信接口必须有手机号/IP/日级限流；
- 业务服务按角色、permissions、部门范围、资源归属和业务状态共同授权；
- 前端隐藏入口不是授权；
- 高风险身份、报告、导出和跨部门操作写业务审计。

## 6. 可靠性与可观测性

- 所有入口传播 request ID / trace ID；
- RPC 和外部服务调用具有超时，重试只用于幂等操作；
- 关键写接口使用业务幂等键；
- 健康检查区分进程存活和依赖就绪；
- 日志使用结构化字段，不拼接敏感载荷；
- Metrics 覆盖延迟、错误率、数据库连接、Redis、Kafka Lag、短信发送与验证结果；
- 审计记录与运行日志分开存储和保留。

详细规则见 [可观测性基线](./03-observability.md) 和
[业务审计模块](../modules/02-business-audit.md)。

## 7. 测试与交付基线

每个变更按风险执行：

- Go 单元测试和关键 Repository 集成测试；
- 契约生成结果与源文件一致性检查；
- 数据库迁移可在空库执行，并提供回滚或明确不可逆说明；
- 鉴权、越权、幂等、并发占号和 Token 轮换测试；
- `go test ./...`；
- 前端类型检查、单元测试和小程序构建；
- Docker Compose 配置验证及必要的端到端联调。

## 8. 演进原则

先实现一个本人、一个项目、一次预约的最小闭环。检查房间由 Appointment 拥有，移动信息按检查顺序需要
增量实现；Patient、Report、Message 可先作为清晰模块，等独立数据所有权、发布节奏或容量需求
成立后再拆服务。不得为了完整的微服务名称提前制造空进程和空数据库。
