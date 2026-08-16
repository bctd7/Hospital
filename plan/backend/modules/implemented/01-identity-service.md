# Identity 当前范围已实现归档

## 服务职责

Identity 是账号、手机号凭据、Access/Refresh 会话、RBAC、医院组织和医生档案的唯一数据拥有者。患者医疗
资料、预约、容量和报告不进入 Identity。

## 已实现能力

- 阿里云 PNVS 手机验证码发送与登录，本地环境可显式启用固定验证码实现；
- 首次手机号登录自动创建患者账号；
- Ed25519 Access Token、Redis Refresh Session、轮换、重放检测和注销；
- 账号、角色、权限或科室变化递增授权版本，使旧 Token 失效；
- 医院、院区、科室目录和按科室查询有效医生；
- 组织单元创建、修改、停用与恢复；
- 管理员账号分页、详情、手机号精确搜索、停用与恢复；
- 开通医生、编辑公开资料、调科、撤销医生身份；
- 本人展示昵称读取与修改；
- `operation_id` 幂等、`management_version`/组织 `version` 乐观锁、授权与组织审计。

账号只表达系统身份。手机号登录不等同于医疗实名，就诊人、身份证、医保和病历不写入 Identity 表。

## 代码组织

```text
internal/
├─ authentication/
│  ├─ manager/             手机验证码认证流程编排
│  └─ sms/                 阿里云与本地短信实现
├─ account/
│  ├─ model.go / store.go  账号、医生和本人资料模型与端口
│  └─ manager/             账号、医生和资料操作
├─ organization/
│  ├─ model.go / store.go  组织模型与端口
│  └─ manager/             公共目录与组织管理
├─ session/                Access/Refresh 会话生命周期
├─ authorization/version/  授权事件与 Kafka -> Redis Consumer
├─ messaging/
│  ├─ kafka/               纯传输 Reader/Writer
│  └─ outbox/              通用 MySQL Outbox Publisher
├─ repository/             MySQL、Redis 适配器
├─ logic/                  RPC 协议适配
└─ svc/                    资源、安全组件、Manager 和 Worker 装配
```

`authentication/manager` 决定认证流程；`authentication/sms` 只发送和校验验证码；`session` 负责签发 Token
和 Refresh Session。普通请求由 RPC Interceptor 校验 Token 与授权版本并注入 Principal，不再主动查询一份
授权上下文。

## 授权版本链路

```text
Account Manager
  -> MySQL 主数据 + 审计 + Outbox（同一事务）
  -> 通用 Outbox Publisher
  -> Kafka Writer
  -> Authorization Version Consumer
  -> Redis 单调更新
  -> 提交 Kafka Offset
```

Kafka 包只提供读写能力；业务 Consumer 属于 `authorization/version`。Redis 更新失败时不提交 Offset，重复
消费依靠授权版本单调写保持幂等。仓库中没有 `projection` 包。

## 权限模型

- 所有有效账号都拥有患者基础能力；
- 医生在患者能力上增加固定科室范围内的工作人员权限；
- 超级管理员拥有跨科室管理权限；
- 前端应用版本只决定页面与调用方式，最终权限由 App API 与 RPC 再次校验。

当前模型是一名医生只隶属于一个科室，不包含多科室任职、排班或实际检查资源绑定。

## 数据与验证

当前初始迁移为 `migrations/identity/000001_identity_initial_schema`，共 12 张业务表。契约事实源是
`contracts/api/identity-*.api` 与 `contracts/proto/identity/v1/identity.proto`。

```powershell
go test ./service/identity/rpc/...
go test ./service/app/api/...
```

## 当前不包含

- 多医院租户；
- 多科室任职和医生排班；
- 医疗实名、就诊人或患者主索引；
- 多设备会话管理界面；
- 复杂审批、临时授权、账号合并和注销工作流。
