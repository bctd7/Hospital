# Appointment 阶段 3：本周预约 CRUD 与容量占用

> 状态：已实现并完成本地验收（2026-08-13）。
>
> 上位文档：[Appointment 预约检查服务总 Plan](../01-appointment-and-examination-booking.md)。
>
> 前置阶段：[检查房间、项目关系与独立周配置](./02-room-project-weekly-configuration.md)。

## 1. 阶段目标

本阶段只建立“患者预约一个检查项目”的最小闭环。主体是 CRUD，不提前实现检查顺序、叫号、医生排班、
检查单、报告、支付、处罚或黑名单。

```text
患者：查询本周可约资源 -> 创建预约 -> 查看本人预约 -> 删除本人预约
工作人员：查询本科室预约 -> 查看预约 -> 核销到院 -> 删除预约
系统：占用/释放具体日期容量 -> 配置变化联动删除 -> 清理当天未核销预约
```

一次预约只包含一个检查项目、一个患者主动选择的房间、一个本周日期和一个
`morning/afternoon` session。医生和管理员只是授权操作者，不是预约资源，预约不保存 `doctor_id`。

## 2. 已确认的业务规则

- 患者只能预约医院时区下的当前自然周，不能预约过去日期、下周或更远日期；
- 患者选择方向固定为“科室 -> 检查项目 -> 房间 -> 日期/session”；
- 患者能力是身份基础能力；医生和管理员切换到患者端后可使用全部患者预约接口，并在此基础上额外拥有工作人员管理权限；
- 后端不自动分配科室、房间或医生；
- 项目和房间必须属于同一科室，且 active 房间—项目关系必须存在；
- 项目时间必须完整落在房间同星期、同 session 的开放窗口内；
- 到达项目 `booking_cutoff_time` 后不能再创建该日期的预约；
- 同一房间、具体日期、session 下的所有项目共享容量；
- 每条有效预约只占一个容量单位，不设置项目权重；
- 患者占用按“具体日期 + morning/afternoon”判断，不比较项目的精确起止时间；
- 上午/下午以 12:00 为唯一分界：上午窗口不得晚于 12:00，下午窗口不得早于 12:00，窗口不可跨越中午；
- 同一患者在同一天同一 session 只能保留一条 `confirmed` 预约，不能通过选择其他项目或房间重复占位；
- 工作人员核销后释放患者的 session 占位，患者可以立即预约该日期同一 session 的下一个项目；房间容量仍按原规则独立统计；
- 不同具体日期的占用完全隔离，两个不同周一不能共用计数；
- 周配置没有修改时持续沿用，修改从本周生效；
- 关闭窗口或其他资源变化使预约失效时，从有效预约中删除；
- 容量调低时，按照 `created_at DESC, appointment_id DESC` 删除最后创建的预约；
- 当天结束后，仍未核销的预约由系统清理；
- 不提供改签接口；改签等价于先创建新预约，再由患者自行删除旧预约。

## 3. 数据模型

### 3.1 预约

新增 `appointment_bookings`：

```text
id
patient_account_id
department_id
item_id
room_id
service_date
session                 morning | afternoon
status                  confirmed | checked_in
room_open_time_snapshot
room_close_time_snapshot
item_start_time_snapshot
item_end_time_snapshot
item_booking_cutoff_snapshot
created_at
updated_at
checked_in_at           nullable
checked_in_by           nullable
version
```

说明：

- `department_id`、项目、房间和时间快照用于稳定展示及后续检查流程，不依赖配置被修改后反查旧语义；
- 首期只有 `confirmed` 和 `checked_in` 两个活动状态；完成检查、排队、叫号在后续阶段扩展；
- 预约不允许修改项目、房间、日期或 session；
- 患者 ID 从认证上下文取得，请求不能替患者伪造；
- 工作人员核销保存操作者账号，但不把操作者变成预约资源。

索引至少覆盖：

```text
(patient_account_id, service_date, created_at)
(department_id, service_date, session, created_at)
(room_id, service_date, session, status, created_at)
```

### 3.2 具体日期容量

新增 `appointment_room_date_capacity`：

```text
id
room_id
service_date
session
total_capacity
occupied_capacity
room_window_id
room_window_version
created_at
updated_at
version
```

唯一约束：

```text
UNIQUE(room_id, service_date, session)
0 <= occupied_capacity <= total_capacity
```

该表是系统运行数据，不要求管理员按日期创建。第一次查询或创建预约时，根据当前有效周配置惰性建立；
进入新一周后自然生成新的日期行，绝不复制上一周占用数量。

### 3.3 幂等与最小审计

预约写操作继续使用 UUID `operation_id`。创建、患者删除、工作人员删除和核销都必须幂等；相同
`operation_id` 携带不同请求返回冲突。即使业务要求从预约主表物理删除，操作记录仍保留最小必要的预约 ID、
操作者、动作、原因和时间，避免重试导致重复释放容量。

### 3.4 患者 session 占位

`appointment_patient_session_claims` 使用
`PRIMARY KEY(patient_account_id, service_date, session)` 保证并发请求最多成功一个。创建预约时在同一事务内
取得占位；患者删除、工作人员删除、配置联动删除和系统清理时释放占位；核销时只释放患者占位，不删除预约历史。
该表是事务约束，不是管理员配置，也不进入热点缓存。

## 4. API 范围

### 4.1 患者接口

```text
GET    /api/v1/appointment/examination-items/:itemId/booking-options
POST   /api/v1/appointment/bookings
GET    /api/v1/appointment/bookings
GET    /api/v1/appointment/bookings/:bookingId
DELETE /api/v1/appointment/bookings/:bookingId
```

`booking-options` 按患者已选择的项目返回本周可选房间、日期、session、窗口时间、总容量和剩余容量。
它只负责展示候选，最终创建必须重新验证数据库事实。

### 4.2 工作人员接口

```text
GET    /api/v1/admin/appointment/bookings
GET    /api/v1/admin/appointment/bookings/:bookingId
POST   /api/v1/admin/appointment/bookings/:bookingId/check-in
DELETE /api/v1/admin/appointment/bookings/:bookingId
```

普通工作人员只能操作认证上下文中的科室，`super_admin` 可以显式指定科室。列表支持日期、session、项目、
房间和状态过滤以及稳定分页。

### 4.3 RPC

Proto 先增加与 HTTP 一一对应的 RPC：

```text
ListBookingOptions
CreateBooking
GetMyBooking
ListMyBookings
DeleteMyBooking
GetBooking
ListBookings
CheckInBooking
DeleteBooking
```

App API 只负责认证上下文传递、HTTP 类型转换和错误映射；预约规则、容量和事务全部在 Appointment Service。

## 5. 创建与删除事务

### 5.1 创建预约

同一 MySQL 事务内：

1. 校验 `operation_id` 幂等；
2. 从认证上下文取得 `patient_account_id`；
3. 按医院时区校验日期属于当前自然周且没有过去；
4. 锁定并校验项目、房间、关系和两条周窗口；
5. 校验当前时间没有超过目标日期的项目停止新增时间；
6. 创建或锁定 `(room_id, service_date, session)` 容量行；
7. 原子执行 `occupied_capacity + 1 <= total_capacity`；
8. 插入 `confirmed` 预约及时间快照；
9. 写入幂等操作记录；
10. 提交后失效本周候选和本人预约缓存。

最后一个容量单位发生并发竞争时，只允许一个事务成功。Redis 剩余容量不参与授权，因此缓存陈旧不会超卖。

### 5.2 删除预约

患者删除、工作人员删除、配置联动删除和系统清理共用一个事务原语：

1. 锁定预约；
2. 验证权限、当前状态和幂等记录；
3. 锁定对应具体日期容量行；
4. 将 `occupied_capacity` 减一且不得为负数；
5. 删除预约主记录；
6. 写入操作记录；
7. 提交后失效候选、患者列表和工作人员列表缓存。

批量删除按容量行分组并使用稳定锁顺序，避免与创建预约、调低容量产生死锁。

## 6. 配置修改联动

阶段 2 的写操作必须补上本周预约联动：

- 房间退役、项目停用、房间—项目关系停用：删除受影响的本周 `confirmed` 预约并释放容量；
- 房间窗口关闭或缩短：删除不再满足完整包含的本周 `confirmed` 预约；
- 项目窗口关闭或缩短：删除不再满足新项目窗口的本周 `confirmed` 预约；
- 房间容量调低：锁定具体日期容量行，按创建倒序删除足够数量的 `confirmed` 预约；
- 配置写入、预约删除、容量修改、审计和幂等结果必须在同一事务提交；
- 批量影响数量和被删除预约 ID 返回管理端，不能静默处理。

## 7. 自动清理

增加 Appointment 内部定时任务，按医院时区清理已经结束日期中仍为 `confirmed` 的预约：

- 查询条件使用 `(service_date, status)` 索引；
- 小批量、可重试处理，不执行全表大事务；
- 每条预约复用幂等删除原语，最多释放一次容量；
- 服务重启后会补扫遗漏日期；
- `checked_in` 预约不属于“未核销”，不自动清理。

## 8. 缓存

继续使用 Cache Aside：

- 本周预约候选：15～30 秒 TTL，加随机抖动；
- 本人预约列表/详情和工作人员列表直接读取 MySQL；它们是身份隔离数据，不属于跨用户热点；
- 管理列表默认不缓存任意组合查询，只缓存高频的“科室＋日期＋session”首页；
- 具体剩余容量可以缓存展示，但创建、删除和配置修改全部以 MySQL 事务为准；
- 同 key 回源使用进程内 singleflight；Redis 不可用时回源 MySQL；
- 写事务提交后递增相关 namespace generation，旧 key 由 TTL 淘汰。

## 9. 代码组织

沿用当前按操作者划分的 manager：

```text
internal/manager/patient/
  booking.go             患者候选、创建、本人查询和删除
  booking_validation.go

internal/manager/staff/
  booking.go             科室预约查询、核销和删除
  booking_effects.go     配置变化联动

internal/repository/mysqlstore/
  booking_store.go
  booking_tx_store.go
  booking_cleanup_store.go
```

共享预约类型和错误仍放在 `internal/manager`，不新建含义模糊的 catalog/resource 业务包。

## 10. 实施顺序

1. 更新本 Plan、Proto 和 App API 契约；
2. 在当前未发布的 `000001` Appointment 初始 Schema 中加入预约、具体日期容量和操作记录；
3. 实现 MySQL Store、事务原语和并发测试；
4. 实现 patient manager 的候选、创建、读取和删除；
5. 实现 staff manager 的列表、详情、核销和删除；
6. 把资源配置写操作接入预约联动；
7. 实现自动清理任务；
8. 接入 App API、OpenAPI 和前端 API 类型；
9. 接入患者端预约页面和管理端预约查看入口；
10. 完成测试、迁移验证、缓存降级验证和后端健康检查。

## 11. 验收标准

- 患者只能看到并创建本周有效候选；
- 项目、房间、科室和窗口不一致时创建失败；
- 同一容量最后一个单位的并发请求只有一个成功；
- 不同日期的容量和占用互不影响；
- 重复创建/删除/核销不会重复占用或释放容量；
- 配置关闭和容量降低按照已确认规则删除预约；
- 当天未核销预约能在日期结束后被补偿清理；
- Redis 不可用时查询和预约仍正确，缓存陈旧不能造成超卖；
- 普通工作人员不能越权读取或操作其他科室预约；
- migration 空库 up/down、`go test ./...`、API 校验和健康检查全部通过。

本地验收同时覆盖微信开发者工具自动预览、正常登录后的真实 HTTP/RPC/MySQL 链路、缓存失效后的剩余容量刷新，
以及患者创建/查询、医生按科室查询/核销/删除的端到端操作。物理手机自动化需要开发者工具提供测试账号
或 test ticket；没有该通道时不得把模拟器或自动预览标记为物理真机通过。

## 12. 已确认的特殊业务点

1. **预约本人 ID**：首期直接使用当前登录人的 `account_id`，请求体不能指定患者身份；
2. **患者主动删除截止时间**：项目窗口开始前允许患者删除，开始后只能由工作人员处理；
3. **已经核销后遇到配置修改**：不自动删除 `checked_in` 预约，存在冲突时拒绝配置修改；
4. **固定预约额度**：本阶段不启用每周创建上限或同时有效上限，只使用房间具体日期容量。
