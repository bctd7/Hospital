# 后端架构与数据边界

## 当前调用关系

```text
微信小程序
  -> App API :8888
       -> Identity RPC :8080
       -> Appointment RPC :8081
       -> Guidance RPC :8082
            -> Appointment RPC（项目事实、配置 TCC、整组预约）
            -> 高德 Web 服务

Identity MySQL Outbox -> Kafka -> Identity 授权版本 Consumer -> Redis
Appointment -> MySQL（业务事实与并发锁）
Appointment -> Redis（热点读取副本）
Guidance -> MySQL（项目规则、配置事务与短期规划方案）
```

小程序只访问 App API。App API 负责 HTTP 鉴权、协议转换和少量页面聚合，不拥有 Identity 或 Appointment
业务表。RPC 服务使用同一 Access Token 重新鉴权，前端隐藏按钮不能替代服务端权限检查。

## 数据所有权

| 数据 | 唯一拥有者 | 其他服务如何使用 |
|---|---|---|
| 账号、手机号、角色、权限、院区、科室、医生档案 | Identity | Token、Identity RPC 或经评审的事件 |
| 检查项目、房间、周窗口、预约、容量、报告、消息已读状态 | Appointment | Appointment RPC |
| 医学顺序规则、准备条件、配置事务、短期方案和推荐逻辑 | Guidance | Guidance RPC；项目与预约事实仍通过 Appointment RPC |
| HTTP 会话展示和页面组合 | App API | 不建立业务事实表 |

Appointment 可以保存 `account_id`、`campus_id`、`department_id` 等稳定外部 ID 和预约时的必要快照，但不
建立跨数据库外键，也不直接查询 Identity 数据库。检查报告属于一次预约的后续结果，当前继续由 Appointment
拥有，不另建 Report 服务。

## Guidance 服务边界

Guidance 已作为独立服务加入，而不是放入 Appointment 或 App API：

```text
微信小程序
  -> App API
       -> Identity RPC
       -> Appointment RPC
       -> Guidance RPC
            -> 外部地图能力
```

Guidance 拥有医学顺序规则、准备条件、项目完整配置协调、短期规划方案和推荐逻辑，但不拥有项目、房间、容量或
预约。它通过 Appointment 的窄用途 RPC 读取项目和预约事实、协调项目配置 TCC，并在患者确认方案后请求
Appointment 原子创建普通预约，不跨库读取主数据。高德地点检索和两点路线由 Guidance 封装，小程序只使用微信
原生地图绘制结果。医院楼栋入口资料库、跨院区组合和可靠现场负载重排仍是后续扩展。
完整功能关系见
[Guidance 当前范围归档](./modules/implemented/03-guidance-service.md)。

## 同步与异步

- 页面立即需要的查询和写结果使用 HTTP -> gRPC；
- Identity 授权失效使用 MySQL Outbox -> Kafka -> Redis；
- Kafka Reader/Writer 只负责传输，业务 Consumer 负责解释事件、更新 Redis 和决定何时提交 Offset；
- 当前 Appointment 组织信息仍由 App API 组合 Identity 与 Appointment 查询，后续只读副本见独立提案。

## 一致性与缓存

MySQL 是业务事实和并发裁决者。Appointment Redis 只缓存项目、房间、关系、周窗口和短时预约候选；写事务
提交后递增科室 generation，使后续请求切换到新缓存键。缓存失败必须回源 MySQL，容量扣减、重复预约和
状态迁移必须在 MySQL 事务内加锁复核，因此缓存陈旧不会造成超卖。

Identity Redis 保存 Refresh Session 与授权版本。授权主数据仍在 MySQL；Redis 版本只用于快速拒绝旧 Token。

## 安全边界

- 日志不得记录验证码、Token、完整手机号、报告正文或请求体；
- 患者读取只能命中本人数据，工作人员读取必须再次校验 permission 与科室范围；
- 医生和管理员在患者端具有患者基础能力，但工作人员权限不会反向授予普通患者；
- 生产环境禁止本地固定验证码和默认 Secret；
- 跨服务不得直接读写对方数据库。
