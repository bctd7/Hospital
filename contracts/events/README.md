# 事件契约

Kafka 事件采用统一信封，业务载荷放在 `payload` 中。当前已经落地 Identity 授权版本事件：

| 项目 | 当前约定 |
|---|---|
| Topic / Event Type | `identity.authorization.changed.v1` |
| Kafka Key | `account_id`，保证同账号事件进入同一分区 |
| Consumer Group | `identity-authorization-version-projection-v1` |
| Payload Schema | `identity-authorization-changed-v1.payload.schema.json` |
| Go 表示 | `Envelope` 与 `IdentityAuthorizationChangedV1Payload` |

Payload 包含 `operation_id`、`action` 和 `authorization_version`。生产者从 MySQL Outbox 构建统一信封；消费者
校验信封、Key 与 Payload 后，只允许 Redis 授权版本增加，成功投影后才提交 Kafka Offset。

事件命名建议：

```text
<domain>.<entity>.<action>.v<major>
```

例如：

```text
identity.authorization.changed.v1
planning.plan.created.v1
```

生产者和消费者必须共同约定：

- Topic 和分区 Key；
- Schema 版本；
- 幂等键；
- 重试和死信策略；
- 敏感字段和保留周期。
