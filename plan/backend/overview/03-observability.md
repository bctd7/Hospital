# 后端可观测性基线

> 状态：结构化访问日志和上下文能力已进入工程基线；集中平台待部署环境确定

## 1. 能力边界

可观测性包括运行日志、访问日志、Trace、Metrics 和告警，用于运行监控与故障排查。它是所有
服务共享的技术能力，不是业务微服务，也不能替代业务审计记录。

```text
API / RPC / Consumer
  -> stdout JSON + Metrics + Trace
  -> 采集器 / OTel Collector
  -> Loki、Prometheus、Grafana 或云平台
```

业务请求不能同步依赖日志平台。采集或存储故障不能阻塞核心业务。

## 2. 当前实现

共享代码位于：

```text
common/observability/logging/
common/observability/httpaccess/
```

当前约定：

- 使用 JSON 结构化日志；
- 生成或校验 `X-Request-ID`，写入响应头和 Context；
- 传播 go-zero Trace 上下文；
- 使用白名单访问日志，记录方法、路由模板、状态码和耗时；
- 保留 Recover、Timeout、Metrics 等安全的框架中间件；
- 关闭可能输出请求内容的默认访问日志；
- 服务日志写标准输出，不在容器内长期保存文件。

## 3. 字段规范

通用字段：

```text
timestamp, level, service, environment
request_id, trace_id, event, error_code, duration_ms
```

HTTP 字段记录 `http_method`、规范化 `http_route` 和 `http_status`。不默认记录 Query、请求体、
响应体、Cookie、Authorization 或完整 User-Agent。

业务过程日志使用稳定事件名，例如 `appointment.schedule_generation_failed`，并只记录排障所需的内部 ID、
版本、数量和阶段。

## 4. 敏感信息

普通日志禁止记录：

- Access/Refresh Token、Authorization、Cookie；
- 短信验证码、完整手机号、微信 code、OpenID 和 Secret；
- 数据库连接串、私钥和第三方 AccessKey；
- 姓名、证件号、住址、完整病历、报告内容和健康数据；
- 完整请求体、响应体和上传文件。

新增日志字段前必须确认必要性、敏感等级、访问权限和保留期限。

## 5. 环境和验收

- 本地可以使用可读输出，测试/预发/生产使用 JSON；
- 生产最低等级通常为 `info`，不能只保留 error 而丢失上下文；
- 测试覆盖 request ID 生成、透传、非法值替换、状态码、耗时和脱敏；
- Identity RPC 和 Kafka Consumer 已使用统一 Context 生命周期；未来新服务接入时继续复用相同日志字段、
  脱敏和取消规则；
- 部署环境确定后再选择 Loki/ELK/云日志、Prometheus 和 OTel Collector；
- 告警以错误率、延迟、依赖健康、Kafka Lag、短信异常和资源饱和为基础。

需要回答“谁对什么业务对象做了什么”的记录见
[业务审计模块](../modules/02-business-audit.md)。
