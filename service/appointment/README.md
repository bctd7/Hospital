# Appointment 服务

Appointment 是检查项目、房间、开放窗口、预约容量、患者预约、现场检查、检查报告和预约消息的数据拥有者。
它回答“能否预约、预约处于什么状态、现场下一位是谁、检查和报告是否完成”，不负责账号组织主数据，也不负责
智能推荐检查顺序。完整业务规则见
[Appointment 当前范围已实现归档](../../plan/backend/modules/implemented/02-appointment-service.md)，本 README 说明这些
规则在当前代码中的承载方式、运行边界和验证入口，不重复维护逐接口字段。

## 业务模型

### 项目、房间与时间

- 一个检查项目只属于一个科室；同名项目如果属于不同院区的不同科室，仍然是不同项目事实；
- 一个房间只属于一个科室，严格地址由院区、楼栋、楼层和房间号组成；
- 一个房间可以执行多个本科室项目，一个项目也可以在多个房间执行；
- 房间开放窗口只表达房间何时开放，项目预约窗口只表达项目何时可以预约，两套时间独立维护；
- 患者从项目出发选择可执行房间，医生和管理员是授权操作者，不是预约资源，不进入窗口和容量模型；
- 项目预计检查时长用于页面展示和 Guidance 规划参考，不占用或切分房间容量。

项目基本事实由 Appointment 保存。工作人员通过 Guidance 的完整三步配置流程创建或更新项目时，Appointment
参与 TCC，确保项目事实与 Guidance 规则整体确认或整体取消；房间、房间—项目关系和两套周窗口仍由 Appointment
独立维护。

### 预约与容量

- 容量按“房间 + 具体日期 + 上午/下午”分别统计，同一天同一大窗内该房间的所有项目共用容量；
- 每次预约只占一个容量单位，最终扣减必须在 MySQL 事务内锁定并复核；
- 患者只预约本周，每周最多创建 10 次预约；取消不会归还本周额度；
- 同一患者、同一项目在 `confirmed`、`queued`、`called` 或 `in_progress` 时不能通过换房间重复占用；
- 检查结束进入 `report_pending` 后已经释放容量和活动占用，可以再次预约同一项目；
- 智能导诊整组确认由 Appointment 在一个本地事务内完成，结果只能全部成功或全部失败。

配置修改同样以预约事实为准：停用关系、关闭窗口或降低容量时，由工作人员业务规则决定受影响预约，不能只改
配置而留下超出新限制的活动预约。

## 患者与工作人员流程

### 患者侧

患者可以读取项目与可预约房间、查看预约候选、创建和取消本人预约、检查报到、查看候检状态、读取本人消息，
并在报告发布后查看只读检查报告。工作人员切换到患者视图后仍以本人患者身份使用这些能力，不会把科室范围
混入“我的预约”。

### 工作人员侧

管理员和授权科室工作人员可以维护本科室检查资源，查看本科室预约与检查记录，按房间叫下一位，确认患者开始
检查、结束实际检查，随后保存报告草稿、发布正式报告或新增更正版本。工作人员权限高于患者能力，但科室操作仍
受当前 Principal 的科室范围约束。

## 状态机与现场队列

```text
confirmed（待报到）
  -> queued（已报到，候检中）
     -> called（正在叫号）
        -> in_progress（工作人员确认到场并开始检查）
           -> report_pending（实际检查结束，报告待完成）
              -> completed（正式报告已发布）

called    -> queued（1 分钟未响应，顺延 3 个实际叫号位置）
confirmed -> no_show（停止报到后仍未报到）
called    -> no_show（项目最终结束后叫号超时）
confirmed -> canceled（报到前取消）
```

候检队列以房间为单位，按照有效报到顺序和过号轮次选择下一位，不使用链表持久化整条队列。一个房间同一时刻
只允许存在一个 `called` 或 `in_progress` 现场占位。后台清理任务持续处理停止报到后的待到院预约和叫号超时，
但所有迁移仍在带锁事务内完成；页面刷新不是状态变化的触发条件。

结束检查立即释放容量和患者项目活动占用并进入 `report_pending`，不会等待医生写完报告。这样既保留真实检查
已经完成的事实，也不会因为报告晚写阻塞新预约。

## 报告与消息

- 报告模板属于检查项目；
- `in_progress` 和 `report_pending` 可以保存草稿，只有 `report_pending` 可以发布首版正式报告；
- 已发布报告不得覆盖原版，只能新增带原因的更正版本；患者始终读取已发布版本；
- 报告保存患者、科室、项目、房间严格地址和执行人员显示名等检查时快照，后续主数据变化不改写历史报告；
- 患者消息和科室消息由预约事实及时间条件生成，取消、叫号、未到场和报告提醒不会依赖前端临时拼接；
- 已读状态按账号单独保存，同一条科室消息可以被不同工作人员分别读取。

检查报告继续属于 Appointment，不因为页面上独立展示就拆成单独 Report 服务。消息同样服务于预约闭环，当前不
扩展成通用站内信平台。

## 服务边界与调用关系

```text
Miniapp
  -> App API
      -> Appointment RPC

Guidance RPC
  -> Appointment RPC（项目事实、窗口和容量候选、配置 TCC、整组预约）
```

- App API 负责 HTTP、Access Token 中间件、协议转换和错误映射，不拥有 Appointment 数据；
- Appointment 从经过校验的 Principal 获取账号、角色和科室范围，不直接查询 Identity 数据库；
- Identity 是账号、角色和组织的唯一事实来源，Appointment 只保存业务发生时必要的患者、科室和地址快照；
- Guidance 读取 Appointment 提供的事实并计算推荐，但 Appointment 始终是容量、重复预约、额度和预约成功的最终
  裁决者；
- Appointment 不访问 Guidance 数据库，也不把规则计算交给小程序；
- 医生只作为操作人员和报告签署人存在，不与房间、项目窗口或容量建立预约资源关系。

## 代码结构

```text
service/appointment/rpc/
├─ appointment.go                 RPC 启动、服务注册和鉴权拦截器
├─ etc/                           本地配置
├─ appointmentservice/            生成的 RPC Client 包装
└─ internal/
   ├─ manager/
   │  ├─ shared/                  患者与工作人员共同读取的项目和房间能力
   │  ├─ patient/                 本人预约、报到、本人报告和消息
   │  ├─ staff/                   资源管理、科室预约、队列、检查、报告和消息
   │  └─ common/                  领域数据、错误、缓存和事务内原子契约
   ├─ repository/mysqlstore/      MySQL Store、事务锁、审计和幂等实现
   ├─ logic/                      gRPC 协议适配、Principal 读取和错误转换
   ├─ server/                     gRPC Server 方法
   └─ svc/                        MySQL、两类 Redis、Manager 和清理 Worker 装配
```

第一层 Manager 按能力使用者划分，而不是继续使用含义重叠的 `catalog`、`resource`。更细的文件职责和 Store 约束见
[Appointment RPC 内部结构](./rpc/internal/README.md)。读取一条请求时遵循：
`server → logic → shared/patient/staff Manager → Store → mysqlstore`。

## 一致性、幂等与缓存

- MySQL 是项目、预约、队列、报告和已读状态的唯一事实来源；
- 容量扣减、患者项目占用、每周额度、候检轮次、状态迁移和整组预约均在事务中复核；
- 写操作使用 `operation_id` 和操作记录保证重试幂等，关键变更同时写入审计；
- Redis 业务缓存采用 Cache Aside：常规资源约 5 分钟并带随机抖动，预约候选约 20 秒；
- 同实例合并并发回源，写后递增科室 generation，使旧 Key 自动失效；
- Redis 失败时读取回源 MySQL，缓存从不参与授权、容量扣减或叫号裁决，因此不会造成超卖或重复叫号；
- 授权版本使用独立 Redis 前缀，只负责让权限变更后的旧 Token 失效，不与业务缓存混用。

## 配置、迁移与验证

Appointment 需要 MySQL、授权版本 Redis、业务缓存 Redis 和 Identity 签发 Token 的 Ed25519 公钥。变量名与本地
默认装配见 `rpc/etc/appointment.yaml`，真实密钥只存在于本地 `.env` 或部署 Secret，不写入 README。

```powershell
.\scripts\database\migrate.ps1 -Service appointment -Direction up
.\scripts\database\seed-comprehensive-test-data.ps1 -Reset
.\scripts\development\start-backend.ps1 -Restart
go test ./service/appointment/...
go test ./service/app/api/...
goctl api validate -api contracts/api/app.api
.\scripts\quality\verify-repository.ps1
```

综合测试数据覆盖多科室、严格地址房间、共享容量、八种预约状态、候检叫号、过号顺延、未到场、草稿、正式与
更正报告，以及患者/科室消息。具体体验账号和场景见
[`scripts/database/README.md`](../../scripts/database/README.md)。

## 当前不包含

- 医生选择、医生排班或医生容量；
- 精确候检时间预测；
- 跨医院租户和跨院区智能组合；
- 通用电子病历、住院病案、收费、医保或电子票据；
- 独立的通用消息中心；
- 由 Appointment 计算智能检查顺序或道路路线。

这些能力必须先在 `plan/backend` 明确业务拥有者和边界，不能从 Handler 或数据表直接扩展。
