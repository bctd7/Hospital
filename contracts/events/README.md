# 事件契约

Kafka 事件采用统一信封，业务载荷放在 `payload` 中。当前只定义通用信封，不提前创建没有真实消费者的 Topic。

事件命名建议：

```text
<domain>.<entity>.<action>.v<major>
```

例如：

```text
planning.plan.created.v1
```

生产者和消费者必须共同约定：

- Topic 和分区 Key；
- Schema 版本；
- 幂等键；
- 重试和死信策略；
- 敏感字段和保留周期。
