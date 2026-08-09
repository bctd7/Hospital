# 可观测性

本目录保存跨服务复用的可观测性基础代码。

## 功能日志

Logic 使用稳定的事件名称记录重要业务结果：

```go
projectlog.Info(
	l.ctx,
	"planning.plan.created",
	logx.Field("plan_id", planID),
)
```

失败事件使用 `Error`，内部错误只进入服务日志，不直接作为客户端响应：

```go
projectlog.Error(
	l.ctx,
	"planning.plan.creation_failed",
	err,
	logx.Field("plan_id", planID),
)
```

事件名称采用 `<domain>.<entity>.<action>`。不要记录 Token、Cookie、微信临时 code、请求体、完整患者资料或检查报告内容。

## HTTP 访问日志

`httpaccess.Middleware` 负责：

- 生成或透传合法的 `X-Request-ID`；
- 将 `request_id` 注入上下文日志；
- 记录方法、路径、状态码和耗时；
- 2xx/3xx 使用 `info`，4xx/5xx 使用 `error`；
- 不读取 Header、Query、请求体和响应体。

业务 Handler 不需要重复记录访问日志。
