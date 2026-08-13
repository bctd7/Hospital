# Appointment RPC 内部分层

请求链路如下：

```text
server -> logic -> catalog/resource Manager -> mysqlstore
```

- `catalog/`：检查项目目录、科室归属、状态和乐观锁；
- `resource/`：房间、房间项目关系、两套独立周窗口、严格包含校验和 Redis Cache Aside；
- `repository/mysqlstore/`：MySQL 查询、事务锁、幂等操作与审计；
- `logic/`：Proto 转换、Principal 获取和 gRPC 错误映射；
- `svc/`：装配 MySQL、鉴权 Redis、业务 Redis 和 Manager。

所有写操作以 MySQL 事务结果为准；Redis 只保存高频读取副本，不参与业务授权或未来的容量扣减判断。
