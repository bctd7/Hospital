# Appointment 阶段 2：检查房间、项目关系与独立周配置

> 状态：业务范围已确认，可以进入实现。
>
> 上位文档：[Appointment 预约检查服务总 Plan](../01-appointment-and-examination-booking.md)。
>
> 本阶段扩展第一阶段检查项目目录，交付检查房间、房间与项目关系、房间周开放配置、项目周预约配置及
> 管理端/患者端查询方向。患者预约、容量占用和核销状态机仍在下一阶段实现，但其已经确认的约束在本文固定。

## 1. 已确认的业务模型

一个检查项目只属于一个科室；一个检查房间也只属于一个科室。南区放射科和北区放射科即使同名，也是
Identity 中两个不同的科室 ID，各自维护项目和房间。

```text
科室
├─ N 个检查项目
└─ N 个检查房间

检查房间 N <-> N 检查项目
```

房间与项目的多对多关系只通过一张关系表表达。一个房间可以加入多个本部门项目，一个项目也可以加入多个
本部门房间；不拆成项目—科室、项目—院区、项目—地点三套关系，也不关联具体医生。

管理员和患者读取同一份关系，但查询方向相反：

```text
管理员：科室 -> 房间 -> 加入、移除和查看检查项目
患者：  科室 -> 检查项目 -> 选择可以执行该项目的房间
```

医生和管理员都是有权限的操作人员。医生仍只通过 Identity 隶属于科室；房间、项目关系、时间配置和未来
预约均不保存 `doctor_id`，不实现医生排班，也不让患者选择具体医生。

## 2. 数据所有权

Appointment 直接拥有“检查房间”这个预约资源，不依赖独立 Navigation Service：

```text
Appointment
  -> 房间所属科室、结构化院区位置、退役时间和版本
  -> 房间开放时间和共享活动容量
  -> 房间可以执行哪些检查项目

后续检查顺序能力
  -> 可选的院区、楼栋、楼层、坐标和移动时间
```

院区由 Identity 提供稳定 `campus_id`，楼栋、楼层和房间号由 Appointment 强约束保存。后续地图与检查顺序
直接引用 `room_id` 和这些结构化字段，不再从自由文本名称中解析位置。Appointment 不拥有 Identity 的科室；
数据库只保存稳定 `department_id` 和 `campus_id`，不建立跨数据库外键。

## 3. 检查房间

### 3.1 字段

```text
appointment_rooms
  room_id
  department_id
  campus_id
  building
  floor_number
  room_number
  retired_at             NULL 表示仍在使用
  version
  created_at
  updated_at
```

约束：

- `room_id` 是稳定 UUID；
- 一个房间只能属于一个科室；
- `campus_id` 必须是稳定 UUID，楼栋必填，楼层只能为 `-9..-1` 或 `1..99`，房间号只允许字母、数字、`-` 和 `_`；
- `(campus_id, building, floor_number, room_number)` 全局唯一，禁止同一物理房间重复登记；
- `display_name` 由服务端根据结构化字段生成，不持久化、不允许管理员自由填写；
- 房间创建后不能改变所属科室；配置错误时修改位置，分配错误时将旧房间标记为不再使用并在目标科室创建；
- 房间没有 active/disabled 切换。“不再使用”只写入 `retired_at`，管理列表和患者查询均不展示，也不提供恢复入口；
- 普通工作人员只能管理当前科室房间，`super_admin` 可以跨科室管理。

### 3.2 管理能力

```text
CreateRoom
GetRoom
ListRooms
UpdateRoom
RetireRoom
```

房间列表只返回仍在使用的房间并支持科室和稳定分页。更新允许修改院区、楼栋、楼层和房间号；不再使用后
立即禁止新预约，但不会物理删除房间、关系、配置或预约历史。

## 4. 房间与检查项目关系

```text
appointment_room_examination_items
  relation_id
  room_id
  item_id
  status                 active | disabled
  version
  created_at
  updated_at
```

约束：

- `UNIQUE(room_id, item_id)`；
- 房间和项目必须存在且属于同一个 `department_id`；
- 激活关系时，房间必须仍在使用且项目必须为 `active`；
- 关系只表达“这个房间可以执行这个项目”，不保存时间、容量、医生或重复的科室 ID；
- 移除项目使用 `disabled`，恢复使用 `active`，不物理删除；
- 房间不再使用或项目停用后，关系记录保持不变，但患者可用查询必须 fail-closed。

管理端以房间为中心提供：

```text
ListRoomExaminationItems
AddRoomExaminationItem
RemoveRoomExaminationItem
```

患者侧以项目为中心提供：

```text
ListAvailableRoomsByExaminationItem
```

患者查询只返回同科室、仍在使用且项目和关系均为 `active` 的房间。

## 5. 两条独立周配置

房间时间与项目时间是两条独立配置线：

```text
房间周配置
  -> 房间什么时候开放
  -> 每个房间窗口的共享活动容量

项目周配置
  -> 项目什么时候允许预约
  -> 项目什么时候停止新增预约
```

房间与项目关系本身不拥有时间和容量。

### 5.1 共同规则

- 一周固定为周一至周日；
- 每天最多有一个 `morning` 和一个 `afternoon`；
- 每个上午或下午是一个连续、自定义起止时间的大窗口；
- 允许 `09:00—12:00` 或 `08:00—11:00`，不支持把上午拆成 `08:00—10:00`、`10:00—12:00`；
- 某天某个 session 没有配置表示不开放或不可预约；
- 没有修改时持续沿用上一周配置；
- 修改从当前自然周立即生效，后续周继续沿用修改后的配置；
- 不实现节假日、单日例外、未来周预排或医生排班；
- 周配置使用乐观版本和审计，不每周复制一套模板数据。

自然周和业务时间统一使用医院配置时区；当前部署按 `Asia/Shanghai`，周一 `00:00:00` 至周日
`23:59:59.999...` 为同一自然周。

### 5.2 房间周开放配置

```text
appointment_room_weekly_windows
  window_id
  room_id
  weekday                1..7
  session                morning | afternoon
  open_time
  close_time
  active_capacity
  version
  created_at
  updated_at
```

约束：

```text
open_time < close_time
active_capacity > 0
UNIQUE(room_id, weekday, session)
```

容量属于“房间＋星期＋上午/下午窗口”，不是项目，也不是整个科室。同一房间窗口下所有项目共享活动容量。

### 5.3 项目周预约配置

```text
appointment_item_weekly_windows
  window_id
  item_id
  weekday                1..7
  session                morning | afternoon
  start_time
  booking_cutoff_time
  end_time
  version
  created_at
  updated_at
```

约束：

```text
start_time <= booking_cutoff_time < end_time
UNIQUE(item_id, weekday, session)
```

停止新增时间属于项目窗口，不属于房间窗口。超过 `booking_cutoff_time` 后，即使房间仍开放且有容量，也不能
继续创建该项目预约。

### 5.4 完整包含与严格修改

项目加入房间时，该项目每个已配置的星期/session 窗口必须完整落在房间对应开放窗口内：

```text
room.open_time <= item.start_time
item.end_time <= room.close_time
```

不自动裁剪，不取部分交集。如果房间缺少对应窗口，或不能完整包含项目窗口，则不能激活房间—项目关系。

修改房间时间时必须重新验证房间下所有 active 项目；修改项目时间时必须重新验证该项目关联的所有 active
房间。不满足完整包含时拒绝修改，并返回冲突房间或项目，不允许配置成功后在患者侧静默消失。

## 6. 已确认的下一阶段预约约束

本节固定后续预约实现必须遵守的规则，第二阶段不提前提供占号接口。

### 6.1 只预约本周

- 患者只能预约当前自然周内的日期；
- 不能预约下周或更远日期；
- 进入新一周后继续使用当前周配置；
- 上周的实际预约和占用数量不会带入新一周。

### 6.2 患者主动选择房间

患者流程固定为：

```text
选择科室
  -> 选择检查项目
  -> 选择能执行该项目的房间
  -> 选择本周日期及 morning/afternoon
```

后端不自动分配或优化科室、房间和时间，也不因后续检查顺序计算修改患者选择。预约保存项目、科室、房间、
日期、项目窗口和房间窗口快照。

### 6.3 房间共享活动容量

每条预约统一占用一个容量单位，不设置项目权重：

```text
available_capacity = active_capacity - active_occupancy
```

同一房间、具体服务日期和 session 下的所有项目共同占用该房间窗口容量。`confirmed`、`checked_in`、
`queued`、`called`、`in_progress` 占用容量；患者取消、医院取消、确认失约或医生完成检查最多释放一次容量。

容量真实占用按具体日期隔离。例如连续两个周一都沿用容量上限 20，但各自的实际预约数独立计算；这属于
系统运行数据，不要求管理员每天创建配置。

项目窗口结束后仍处于 `confirmed`、没有报到核销的预约自动进入 `no_show` 并释放容量。已经报到、排队、
叫号或检查中的预约不按未核销清理。

### 6.4 配置修改和医院取消

房间、项目或关系停用，以及本周配置缩短后，不物理删除已有预约。无法继续执行的预约进入
`cancelled_by_hospital`，不计患者责任，释放一次容量并保留历史、原因和审计。

房间容量调低到当前占用以下时，按以下稳定顺序取消足够数量的最新 `confirmed` 预约：

```text
created_at DESC, appointment_id DESC
```

已经报到、排队、叫号或检查中的预约不能自动取消。如果取消全部 `confirmed` 后占用仍高于新容量，则拒绝
容量调整。

## 7. 权限、幂等和审计

第二阶段沿用已存在权限：

- 读取使用 `appointment.read`；
- 创建房间、关系和周配置使用 `appointment.create`；
- 更新、停用、恢复和替换周配置使用 `appointment.update`。

普通工作人员只能操作当前 `department_id` 的房间、项目和关系，`super_admin` 可以跨科室。权限和科室范围
从认证上下文取得，不能由请求伪造。

所有写操作：

- 携带 UUID `operation_id`；
- 更新携带 `expected_version`；
- 相同操作和相同请求返回第一次结果；
- 相同操作携带不同请求返回幂等冲突；
- 主数据、幂等结果和不含完整项目描述的审计摘要处于同一 MySQL 事务；
- 稳定区分参数非法、无权限、不存在、唯一冲突、版本冲突、状态冲突和幂等冲突。

## 8. 热点缓存设计

房间、项目、关系和周配置会被管理端和患者端高频读取，Appointment 使用现有业务 Redis 做 Cache Aside：

```text
MySQL                         最终事实源
Redis                         热点读副本
进程内 singleflight           合并同 key 回源
```

### 8.1 缓存内容

- 科室下 active 房间列表；
- 房间下 active 项目列表；
- 项目下 active 房间列表；
- 房间周开放配置；
- 项目周预约配置；
- 本周患者候选房间/窗口的短期查询结果；
- 不存在的资源使用很短的负缓存，防止穿透。

缓存 key 必须包含领域、科室/资源 ID、当前周起始日期和数据版本，不能缓存任意 SQL 查询结果。列表采用科室
namespace generation；写事务提交后递增对应 generation 并删除实体 key，旧列表 key 通过 TTL 自动淘汰。

### 8.2 TTL 和防击穿

- 房间、关系和周配置：基础 TTL 5 分钟并加入随机抖动；
- 本周候选结果：TTL 30 秒；
- 负缓存：TTL 10 秒；
- 同一进程使用 singleflight 合并并发回源；
- Redis 不可用时降级读取 MySQL，不允许以空结果伪装成功；
- 缓存命中、未命中、回源、失效失败和降级必须有指标，日志不得包含项目完整描述或患者信息。

### 8.3 容量不能只依赖缓存

Redis 中的剩余容量和候选窗口只能用于快速展示，不能作为预约成功依据。创建预约必须在 MySQL 本地事务中
重新检查本周限制、项目停止新增时间、房间开放状态和真实活动容量，并原子增加占用。缓存即使短暂陈旧，
最多影响展示，不能造成超卖；预约事务提交后再失效对应房间和项目的本周候选缓存。

## 9. 第二阶段接口范围

Proto 优先新增：

```text
Room CRUD
RoomExaminationItem 关系管理及双向列表
RoomWeeklyWindow CRUD
ItemWeeklyWindow CRUD
管理员房间视角聚合查询
患者项目视角的可用房间/本周配置查询
```

App API 同步提供管理员 HTTP 接口和只读患者查询接口，并生成 OpenAPI。第二阶段患者接口只返回配置层面的
候选，不创建预约，不修改容量。

## 10. 数据库迁移

通过 `000002` 增量迁移增加：

```text
appointment_rooms
appointment_room_examination_items
appointment_room_weekly_windows
appointment_item_weekly_windows
appointment_resource_operations
appointment_resource_audit
```

只在 Appointment 数据库内建立外键；不修改已经执行的 `000001`，不建立跨 Identity 数据库外键，不为未来
预约或地图建立空表。

## 11. 测试与验收

至少覆盖：

- 房间 CRUD、物理位置唯一、结构化字段校验、单向退役、乐观锁、幂等和科室权限；
- 一个房间加入多个项目、一个项目加入多个房间；
- 跨科室关系被拒绝，关系中不存在 `doctor_id`；
- 管理员按房间查项目、患者按项目查房间得到同一关系的相反视图；
- 周一至周日、上午/下午唯一，窗口连续且时间合法；
- 房间时间和项目时间独立保存；
- 项目窗口不能被房间完整包含时，新增关系或修改时间被拒绝并返回冲突；
- 房间窗口容量属于房间/session，不属于项目；
- 停用任一资源后患者查询 fail-closed，历史数据不物理删除；
- 缓存命中、失效、TTL、负缓存、singleflight 和 Redis 故障降级；
- 缓存陈旧时写路径仍以 MySQL 校验为准；
- 空库 `up`、最近版本 `down -> up` 和真实 MySQL 集成测试；
- `go test ./...`、`go vet ./...`、API 契约校验和仓库统一检查通过。

## 12. 完成边界

本阶段完成表示管理员可以管理科室房间、为房间加入检查项目，并独立配置房间和项目的周窗口；患者侧可以
按“科室—项目—房间”读取本周候选配置。以下内容留给下一阶段：

- 患者档案和本人 ID；
- 创建、查询和医院/患者取消预约；
- 具体日期活动占用的原子增加和最多一次释放；
- 窗口结束后的自动失约任务；
- 方案 A 额度；
- 报到、排队、叫号、开始和完成；
- 检查顺序、移动时间和报告。
