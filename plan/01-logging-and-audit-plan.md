# 日志与审计能力实施方案

> 文档状态：待评审
>
> 适用基线：Go 1.26、go-zero 1.10.3
>
> 目标：明确日志能力的系统归类、部署边界、实现位置和验收标准

## 1. 结论

日志能力可以在当前阶段完成第一版。共享实现从一开始放入 `common/observability`，但不建设为独立微服务。

当前应区分两类能力：

1. **技术日志**：用于开发、运行监控和故障排查，属于横切的工程基础能力。
2. **业务审计记录**：用于回答“谁在什么时间对什么业务数据执行了什么操作”，属于业务与合规能力。

技术日志现在可以完整交付 V1。业务审计可以先建立统一结构和写入接口，但具体审计事件必须随用户角色、业务流程、敏感数据范围和保留期限一起确定。

## 2. 为什么不做成日志微服务

技术日志发生在每个请求和业务执行现场。如果每条日志都同步调用一个日志服务，会额外引入：

- 网络调用和响应延迟；
- 日志服务故障导致业务请求受影响的风险；
- 高并发下的日志服务容量压力；
- 服务间循环依赖和启动顺序问题；
- 敏感数据在网络中再次传播的问题。

因此，应用服务应直接将结构化日志写到标准输出，由部署平台或日志采集器统一收集。开发环境可以直接查看控制台，生产环境再接入 Loki、Elasticsearch、云日志服务或其他集中检索平台。

```text
app-api / future-rpc / consumer
        -> stdout JSON
        -> 日志采集器
        -> 集中存储与检索平台
```

日志平台是部署基础设施，不是 Hospital 的业务微服务。业务代码不能通过同步 RPC 依赖日志平台。

业务审计记录需要可靠保存，首期应由拥有该业务数据的服务在同一数据库事务或可靠异步链路中写入。只有未来出现跨系统统一审计、独立权限、独立保留周期和独立运维团队时，才评估独立审计服务。

## 3. 能力分类

| 类型 | 目的 | 示例 | 保存位置 | 当前是否实施 |
|---|---|---|---|---|
| 启动日志 | 记录服务启动、配置加载和退出 | 服务名、版本、监听端口 | 标准输出 | 是 |
| HTTP 访问日志 | 观察请求结果和性能 | 方法、路由、状态码、耗时 | 标准输出 | 是 |
| 应用错误日志 | 定位系统异常 | 错误码、调用位置、依赖错误 | 标准输出 | 是 |
| 业务过程日志 | 辅助排查业务执行 | 方案生成成功、规则计算失败 | 标准输出 | 是，但仅记录必要标识 |
| Trace | 串联 API、RPC、数据库和消息 | `trace_id`、`span_id` | Trace 后端或日志字段 | 建立传播基础 |
| 指标 | 聚合观察系统状态 | QPS、P95、错误率 | Prometheus 等时序系统 | 后续接入 |
| 业务审计记录 | 证明关键业务操作 | 管理员修改规则、查看敏感资料 | MySQL 审计表 | 随业务功能实施 |

日志、Trace、指标和审计解决的问题不同，不能用普通日志代替审计记录，也不能把日志内容当作业务事实数据。

## 4. go-zero 提供的能力与项目采用范围

go-zero 提供的是构建服务所需的框架和组件，不是要求项目把每项能力都拆成一个微服务。Hospital 可以按阶段使用以下部分：

| go-zero 能力 | 主要作用 | Hospital 的采用决定 |
|---|---|---|
| REST API | HTTP 路由、请求处理和中间件 | 现在使用，作为小程序入口 |
| goctl | 从 `.api`、`.proto` 和数据库定义生成骨架 | 现在使用，契约优先 |
| `conf` / ServiceConf | 加载配置、初始化服务和优雅退出 | 现在使用 |
| `logx` / `logc` | 结构化、上下文关联日志 | 现在使用，由 `common/observability` 统一项目约定 |
| REST 内置中间件 | Trace、Recover、Timeout、Breaker、Shedding、MaxConns、MaxBytes、Metrics、Prometheus 等 | 现在保留安全且有明确用途的部分；默认请求日志单独替换 |
| OpenTelemetry Trace | 串联 HTTP、zRPC、SQL 和 Redis 调用 | 现在保留传播能力，确定部署环境后接 Collector |
| Prometheus Metrics | 自动记录 HTTP/RPC 请求量、状态和耗时 | 现在保留框架能力，确定监控环境后开放采集端点 |
| `sqlx` / Model | MySQL 访问、事务和代码生成 | 数据模型确定后使用 |
| Redis 组件 | 缓存、限流、幂等和分布式状态 | 出现明确缓存或幂等需求后使用 |
| JWT / 鉴权中间件 | 接口身份验证 | 微信登录和 Token 方案确定后使用 |
| zRPC / gRPC | 微服务间同步调用、连接池、负载均衡和熔断 | 出现第二个明确服务边界后使用 |
| etcd / 静态 Endpoint / Kubernetes DNS | RPC 服务发现 | 本地与 CI 使用静态地址；多实例生产环境再选择 etcd 或平台 DNS |
| Kafka `kq` | Kafka 生产与消费 | 出现明确异步事件、生产者和消费者后使用 |
| 并发与弹性组件 | MapReduce、限流、熔断、过载保护 | 根据真实性能和可靠性需求使用 |

当前最合适的组合是：REST、goctl、配置、`logx`、Trace、Recover、Timeout、MaxBytes 和基础 Metrics。MySQL、Redis、JWT 随第一条业务链路加入；zRPC、etcd 和 Kafka 暂不作为日志 V1 的依赖。

参考资料：

- [go-zero 组件总览](https://go-zero.dev/components/)
- [go-zero HTTP 中间件](https://go-zero.dev/guides/http/server/middleware/)
- [go-zero 链路追踪](https://go-zero.dev/components/observability/tracing/)
- [go-zero 服务发现](https://go-zero.dev/guides/microservice/service-discovery/)
- [go-zero Kafka](https://go-zero.dev/components/queue/kafka/)

## 5. 什么是集中日志平台

应用输出日志之后，还需要有地方负责收集、保存、检索、控制权限和设置告警。完成这些工作的系统称为集中日志平台。

例如：

```text
多个服务输出 stdout JSON
    -> Promtail / Fluent Bit / Vector 等采集器
    -> Loki / Elasticsearch / 云厂商日志服务
    -> Grafana / Kibana / 云控制台查询与告警
```

如果没有集中日志平台，只能登录每台机器或查看单个容器日志。服务数量或实例数量增加后，很难使用同一个 `request_id` 或 `trace_id` 搜索完整调用链。

Hospital 当前只需要保证日志以稳定的 JSON 输出到标准输出。集中日志平台的选型取决于最终部署位置：单台服务器可以采用 Loki + Grafana；使用云平台时通常优先评估云日志服务；进入 Kubernetes 后可以使用 DaemonSet 采集器。该平台不是日志 V1 的阻塞项。

## 6. 当前版本的实现边界

### 4.1 V1 包含

- 使用 JSON 结构化日志；
- 为请求生成或透传 `request_id`；
- 将 `request_id` 写入响应头和请求上下文；
- 记录 HTTP 方法、规范化路由、状态码、耗时、服务名和环境；
- 将 go-zero Trace 上下文中的 `trace_id` 自动关联到 Logic 日志；
- 保留 go-zero 的 Trace、Recover、Timeout、Metrics 等内置中间件；
- 对服务启动、关闭和不可恢复错误进行记录；
- 定义业务日志字段规范和敏感信息禁记清单；
- 为中间件、字段脱敏和上下文传播编写测试。

### 4.2 V1 不包含

- 独立日志微服务；
- Loki、Elasticsearch 或云日志平台部署；
- 完整 OpenTelemetry Collector、Prometheus 和 Grafana；
- 把请求体和响应体完整写入日志；
- 业务审计表及具体审计事件；
- 基于日志的复杂告警规则。

这些能力不影响当前 API 服务完成日志 V1，后续可以在不改业务接口的情况下接入。

## 7. go-zero 日志使用策略

当前 `Config` 嵌入了 `rest.RestConf`，因此可以直接在 `app-api.yaml` 中配置 go-zero 的 `Log` 和 `Middlewares`。

go-zero 1.10.3 默认开启 Trace、Log、Recover、Timeout、Metrics 等 REST 中间件。框架的默认 Log 中间件在服务端错误场景可能输出请求内容，其中可能包含请求头、Token 或业务字段，不适合直接作为医院项目的生产访问日志。

项目采用以下策略：

1. 保留 go-zero 的 Trace、Recover、Timeout 和 Metrics。
2. 关闭内置 HTTP `Middlewares.Log`。
3. 使用项目自有的轻量访问日志中间件，只记录白名单字段。
4. 继续使用 `logx.WithContext(ctx)` 记录 Logic 日志，使日志自动关联 Trace 上下文。
5. 日志输出到标准输出，不在应用容器内部维护长期日志文件。

推荐配置方向：

```yaml
Name: app-api
Host: 0.0.0.0
Port: 8888

Log:
  ServiceName: app-api
  Mode: console
  Encoding: json
  Level: info

Middlewares:
  Trace: true
  Log: false
  Recover: true
  Timeout: true
  Metrics: true
```

本地环境如需更易读的输出，可使用 `plain`；测试、预发布和生产统一使用 `json`。

`Level` 表示允许输出的最低等级，不是单条错误日志的等级。生产环境保持 `Level: info`，才能保留正常访问、启动和关键业务状态日志；错误事件使用 go-zero 的 `error` 等级。如果把全局最低等级设置成 `error`，所有 `info` 访问日志都会被丢弃，出现故障时会缺少前后文。

## 8. 代码放置位置

日志必然被 API、RPC 和 Consumer 共同使用，因此共享实现放在 `common`；各服务只保留自身配置和初始化入口：

```text
common/observability/
├── logging/
│   ├── context.go               # request_id 等公共日志上下文
│   ├── fields.go                # 公共字段名和事件名约定
│   └── sensitive.go             # 敏感字段规则
└── httpaccess/
    ├── middleware.go            # 可复用的 HTTP 访问日志中间件
    ├── response_writer.go       # 捕获 HTTP 状态码
    └── middleware_test.go

service/app/api/
├── app.go                       # 挂载公共中间件
├── etc/app-api.yaml             # app-api 的日志配置
└── internal/logic/              # 使用上下文 Logger 记录业务事件
```

职责分配：

- `common/observability/logging`：维护跨服务字段、上下文注入和敏感信息规则；
- `common/observability/httpaccess`：处理 `request_id`，统计状态码和耗时，只输出白名单字段；
- `app.go`：在注册路由前挂载公共中间件；
- `logic/`：使用上下文 Logger 记录必要的业务执行结果；
- `etc/app-api.yaml`：配置日志格式、等级和内置中间件开关。

`common` 只提供库代码，不启动进程、不监听端口、不访问业务数据库。服务仍然各自输出日志，因此共享代码不会形成运行时单点故障。

## 9. 日志字段规范

### 7.1 通用字段

| 字段 | 说明 |
|---|---|
| `timestamp` | UTC 时间 |
| `level` | `debug`、`info`、`error` 或 `severe` |
| `service` | 服务名，例如 `app-api` |
| `environment` | `local`、`test`、`staging` 或 `production` |
| `request_id` | 单次入口请求标识 |
| `trace_id` | 跨 API、RPC 和消息链路标识 |
| `event` | 稳定的日志事件名称 |
| `error_code` | 项目统一错误码，成功时省略 |
| `duration_ms` | 操作耗时，单位毫秒 |

### 7.2 HTTP 访问日志字段

- `http_method`；
- `http_route`，记录路由模板，不记录带业务 ID 的原始 URL；
- `http_status`；
- `duration_ms`；
- `request_id`；
- `trace_id`；
- 经可信代理处理后的客户端网络信息，仅在确有需要时保留。

不要默认记录 Query、请求体、响应体、Cookie、Authorization 或完整 User-Agent。

### 7.3 业务日志字段

业务日志使用稳定事件名，不使用难以检索的长句：

```text
planning.plan_generation_started
planning.plan_generation_completed
planning.plan_generation_failed
```

业务对象只记录内部不可逆或低敏标识。是否允许记录用户 ID、方案 ID、检查项目 ID，需要在数据分级文档中确定。

## 10. 敏感信息规则

以下内容不得进入普通日志：

- `Authorization`、Access Token、Refresh Token；
- 微信临时 `code`、AppSecret、Session Key；
- 密码、验证码、数据库连接串和 Secret；
- 身份证号、手机号、姓名、住址等直接身份信息；
- 完整病历、诊断、检查结果和其他健康数据；
- 完整请求体、响应体、Cookie 和文件内容。

需要排查问题时，优先记录错误码、规则版本、内部追踪 ID、对象数量和执行阶段。任何新增字段都应先判断其敏感等级、必要性和保留期限。

## 11. 实施步骤

### 步骤一：确定配置基线

1. 在 `app-api.yaml` 增加 `Log` 和 `Middlewares` 配置。
2. 为配置增加环境字段，避免从日志内容推断运行环境。
3. 测试配置文件能够被 go-zero 正确加载。

### 步骤二：建立请求上下文

1. 接收合法的 `X-Request-ID`，不存在或不合法时生成新值。
2. 使用 `logx.ContextWithFields` 将 `request_id` 放入日志上下文，保证后续 `logx.WithContext(ctx)` 自动携带该字段。
3. 在响应头返回 `X-Request-ID`。
4. 限制外部 ID 的长度和字符范围，防止日志注入。

### 步骤三：实现脱敏访问日志

1. 包装 `http.ResponseWriter` 获取最终状态码。
2. 记录请求开始时间并计算耗时。
3. 只输出字段白名单，不序列化整个请求。
4. 2xx/3xx 使用 `info`，4xx/5xx 使用 `error`；后续日志量明显增大时，再将可预期的参数校验类 4xx 调整为 `info`。
5. 健康检查按普通请求处理，不增加特殊规则。

### 步骤四：统一 Logic 日志约定

1. 使用 `logx.WithContext(ctx)`，不创建脱离上下文的全局业务 Logger。
2. 正常 CRUD 不逐行打印；只记录重要状态变化、异常和耗时操作。
3. 错误日志只在最能补充上下文的一层记录一次，避免 Handler、Logic、Model 重复打印同一错误。
4. 使用统一错误码，日志保存内部原因，客户端返回安全的错误信息。

### 步骤五：测试与校验

至少覆盖：

- 无请求 ID 时生成新 ID；
- 合法请求 ID 可以透传；
- 非法或超长请求 ID 被替换；
- 响应头包含请求 ID；
- 访问日志包含方法、路由、状态码和耗时；
- 日志不包含 Authorization、Cookie 和请求体；
- Logic 日志能够关联请求上下文；
- panic 被 Recover 捕获且不泄露敏感请求数据。

### 步骤六：环境验收

1. 本地启动 API 并请求 `/api/v1/health`。
2. 验证控制台输出可解析的结构化日志。
3. 使用相同 `X-Request-ID` 请求并确认响应透传。
4. 构造 4xx 和 5xx 场景，确认日志等级和字段正确。
5. 搜索日志，确认没有 Token、Cookie、请求体和敏感字段。

## 12. 完成标准

日志 V1 满足以下条件即可视为完成：

- 所有 HTTP 请求都能通过 `request_id` 定位；
- Logic 日志可以关联同一请求的 Trace 上下文；
- 访问日志为结构化输出，字段名称稳定；
- 生产模式不输出请求体、响应体、凭据和医疗敏感信息；
- 日志失败不会阻断正常业务请求；
- 中间件具备自动化测试；
- 本地、测试和生产的日志级别与格式可配置；
- 后续增加 RPC 或 Consumer 时可以复用约定，不需要建立日志微服务。

## 13. 后续讨论项

实现前还需要确认以下选择：

1. 外部请求头使用 `X-Request-ID`，还是统一采用 W3C Trace Context 并只向客户端返回 Trace ID；
2. 生产日志保留周期和访问权限；
3. 第一批必须进入业务审计表的操作；
4. 是否在首期接入集中日志平台，还是先保留标准输出接口。

当前决定是：客户端保留 `X-Request-ID`，服务内部同时使用 Trace；生产最低日志等级为 `info`，4xx/5xx 请求使用 `error`；健康检查不做特殊处理；日志先输出标准输出，待部署环境确定后再选择采集平台。
