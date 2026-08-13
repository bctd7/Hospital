# common/authz

`authz` 在 `authn` 已经建立有效 Principal 之后回答“这个身份能做什么”。

```text
authorizer.go       permission 与科室范围判断
version/            authorization_version 校验边界
version/redisstore  Redis 中的当前授权版本
```

MySQL 是角色、权限和授权版本的主数据；Redis 版本只用于拦截器快速拒绝旧
Token。Identity 的授权版本 Consumer 负责 Kafka 到 Redis 的同步。
