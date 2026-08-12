# Appointment 阶段 1：检查项目描述 CRUD 与服务骨架

> 状态：业务范围已确认，供人工实现和学习验证。
>
> 上位文档：[Appointment 预约检查服务总 Plan](../01-appointment-and-examination-booking.md)。
>
> 本阶段只保存自然语言描述，不实现 AI 提取、结构化时间约束、预约窗口、容量、预约或排序。

## 1. 阶段目标

建立第一个可独立运行的 Appointment RPC 服务，并打通：

```text
Proto 契约
  -> 生成 RPC 基础框架
  -> Appointment Service
  -> Repository
  -> hospital_appointment MySQL
  -> 检查项目描述 CRUD
```

本阶段完成后，管理员可以创建、查询、分页列出、修改、停用和恢复检查项目。患者预约前需要了解的禁食、
禁水、携带资料、准备动作和时间间隔等内容，全部原样保存在 `description` 字符串中。

这属于业务规则较少的 CRUD，但仍遵守仓库已有的契约优先、Manager/Repository 分层、软停用、乐观锁、
幂等、权限和测试基线。生成代码只提供框架，业务判断不能堆入生成的 Server 或数据库 Store。

## 2. 已确认的数据口径

### 2.1 检查项目

一条检查项目至少表达：

- 稳定内部 `item_id`；
- 所属管理科室 `owner_department_id`；
- 科室内稳定且唯一的 `code`；
- 患者可见 `name`；
- 患者可见 `description`；
- `active/disabled` 状态；
- 从 1 开始递增的 `version`；
- 创建和更新时间。

`owner_department_id` 表示当前由哪个科室维护该项目，不等于该检查未来只能在一个地点执行。后续项目与
院区、执行科室和地点的多对多关系使用独立关系表增加，不把执行地点提前塞入本表。

### 2.2 描述字段

- Proto 使用 `string`，MySQL 使用可容纳长文本的 `TEXT`；
- `description` 保存完整自然语言，不拆成禁食、禁水、时长等固定列；
- 第一阶段不解析描述、不检查医学语义，也不根据描述计算检查顺序；
- 更新描述与更新名称一样属于项目更新，成功后 `version + 1`；
- 读取接口完整返回描述，日志、审计摘要和错误信息不得复制描述正文；
- 字符长度上限在 Proto/Manager 验证中统一设置，数据库类型不是无限输入授权。

## 3. 为后续结构化提取保留的边界

当前只保留稳定 `item_id` 和 `version`，不提前创建空 JSON 字段或未使用的规则表。以后接入 AI 时通过增量
迁移增加提取记录和结构化规则，至少关联：

```text
item_id
source_item_version
schema_version
model_version
prompt_version
extraction_status
review_status
```

未来流程为：

```text
读取指定 item_id + version 的 description
  -> 受 Schema 约束的 AI 提取
  -> 结构校验
  -> 人工审核
  -> 发布固定语义规则版本
```

AI 结果只能先成为草稿，不能直接成为患者侧硬约束。描述更新后，旧提取结果保留用于历史复现，但因
`source_item_version` 不匹配而不能自动用于新版本；新版本需要重新提取和审核。

## 4. Proto 优先

先新增 `contracts/proto/appointment/v1/appointment.proto`，包名和生成包遵守现有版本规范。第一阶段 RPC：

```text
CreateExaminationItem
GetExaminationItem
ListExaminationItems
UpdateExaminationItem
DisableExaminationItem
EnableExaminationItem
```

契约原则：

- 写请求携带 `operation_id`；更新、停用和恢复携带 `expected_version`；
- 操作者账号、角色、科室和 permissions 从认证上下文取得，不能由请求伪造；
- 列表支持游标或稳定分页、科室过滤和状态过滤，不一次返回全库；
- `ExaminationItem` 返回稳定业务字段，不暴露数据库列名或审计内部数据；
- 已发布字段号不能复用，后续新增结构化状态时只追加新字段；
- 错误至少稳定区分参数非法、无权限、不存在、编码冲突、版本冲突和幂等冲突。

Proto 评审通过后再生成：

- `contracts/gen/appointment/v1/`；
- `service/appointment/rpc/appointmentservice/`；
- RPC Server/Logic 基础文件。

所有标记为生成代码的文件只通过工具重新生成，不手改。

## 5. Service 基础框架

新增结构遵循 Identity 已验证的分层：

```text
service/appointment/rpc/
  appointment.go
  etc/appointment-rpc.yaml
  appointmentservice/          生成的 RPC Client
  internal/
    config/
    logic/
    server/
    svc/
    catalog/                    检查目录 Manager 与领域模型
    repository/
      mysqlstore/
```

职责边界：

- Server：把生成接口交给 Logic，不写业务判断；
- Logic：转换 Proto 请求、读取认证主体、调用 Manager、映射稳定 gRPC 错误；
- Catalog Manager：校验字段、permission、科室范围、状态、版本和幂等；
- Repository：声明读取和事务写入端口；
- MySQL Store：实现 SQL、唯一冲突映射和事务；
- ServiceContext：创建数据库连接和 Manager，并在关闭时释放资源。

服务复用 `common/authn` 的 Access Token 与授权版本校验。普通科室工作人员只能维护本人当前科室项目；
`super_admin` 可以跨科室维护。第一阶段使用 `appointment.catalog.manage`，若 Identity 中尚无该权限，必须
通过新的 Identity 增量迁移加入，不能修改已经执行过的历史迁移。

## 6. MySQL 接入

新增：

- `APPOINTMENT_MYSQL_USER`、`APPOINTMENT_MYSQL_PASSWORD`、`APPOINTMENT_MYSQL_DSN`；
- 本地 `hospital_appointment` 数据库和最小权限账号；
- `migrations/appointment/000001_appointment_catalog.*.sql`；
- `scripts/migrate.ps1` 中 `appointment` 白名单和迁移目录映射；
- Appointment RPC 配置中的 MySQL DataSource。

第一版主表概念结构：

```text
examination_items
  id
  owner_department_id
  code
  name
  description            TEXT
  status                 active | disabled
  version
  created_at
  updated_at
```

约束：

- `id` 为稳定 UUID；
- `(owner_department_id, code)` 唯一；
- `version >= 1`；
- 不建立指向 Identity 数据库的外键；
- 不物理删除业务记录；
- 非法 DSN 或数据库不可用时服务启动失败，不以空 Store 静默运行。

为了让写操作可重试和可追溯，同一迁移阶段加入最小的操作幂等/审计存储；一次写操作的项目变更、操作记录
和必要审计必须处于同一个本地事务。当前没有下游消费者时不为形式完整提前发布空 Kafka 事件；确有事件
消费方后再以增量迁移加入 Appointment Outbox。

## 7. CRUD 规则

### 创建

- 校验科室、编码、名称和描述；
- 相同 `operation_id`、相同请求返回第一次结果；
- 相同 `operation_id`、不同请求返回幂等冲突；
- 同一科室下编码重复返回编码冲突；
- 创建结果为 `active`、`version = 1`。

### 查询与列表

- ID 查询返回 active 或 disabled 项目；
- 管理列表可以按科室和状态过滤；
- 未来患者目录只能读取 active 项目，但患者 HTTP 查询不在本阶段；
- 列表顺序必须稳定，翻页不能依赖未指定顺序。

### 更新

- 允许修改名称和描述；编码是否允许修改在 Proto 评审时固定，默认创建后不修改；
- `expected_version` 不等于当前版本时返回版本冲突；
- 成功更新后版本递增；
- disabled 项目默认不能普通更新，需先恢复；
- 描述更新不触发 AI 或结构化规则计算。

### 停用与恢复

- “删除”使用停用语义，不提供物理删除 RPC；
- 停用和恢复都是带版本的幂等写操作；
- 重复执行同一 `operation_id` 返回原结果；
- 状态变化不修改历史审计。

## 8. 实施顺序

1. 编写并人工检查 Appointment Proto；
2. 生成共享 Proto 类型、RPC Client、Server 和 Logic 基础框架；
3. 增加 Appointment 配置、进程入口和认证拦截器；
4. 增加独立数据库配置、本地建库和空库迁移；
5. 先完成 MySQL 连接与启动失败测试；
6. 定义 Catalog Manager 和 Repository 端口；
7. 手写 MySQL Store 与 CRUD Logic；
8. 补齐权限、幂等、乐观锁、审计和错误映射；
9. 更新服务、迁移和契约 README；
10. 运行统一检查。

这一顺序故意先把 Proto 和可启动的服务骨架搭好，再手写领域与持久化内容，适合作为当前学习成果验证。
本阶段不修改 App API 和小程序；RPC 与数据库集成测试通过后，再单独计划患者/管理员 HTTP 纵向接入。

## 9. 测试与验收

至少覆盖：

- Proto 生成结果可编译；
- 正确 DSN 可以启动，错误 DSN 或数据库不可达时明确失败；
- 空库迁移 `up` 成功，最近版本 `down -> up` 成功；
- 创建、按 ID 查询、分页列表、更新、停用和恢复；
- 描述包含长文本、换行和中文时可以无损往返；
- 编码冲突、版本冲突、非法状态转换和无权限；
- 同操作同请求幂等、同操作不同请求冲突；
- 普通工作人员不能跨科室维护，超级管理员可以；
- 日志与审计摘要不包含完整 `description`；
- `go test ./...`、`go vet ./...` 和仓库 `scripts/check.ps1` 通过。

## 10. 完成边界

本阶段完成只表示“检查项目描述 CRUD 与 Appointment RPC/MySQL 基础设施”可用，不表示预约服务完成。
以下内容明确留给后续 Plan：

- AI 提取、结构化时间约束、规则审核和发布；
- 准备动作节点、时间区间求值和排序求解器；
- 项目与执行科室、院区、地点的关系；
- 上午/下午窗口、活动容量和并发占用；
- 患者档案、预约、限制方案 A、取消和历史；
- 报到、队列、叫号、检查执行和报告；
- App API、OpenAPI 和小程序页面接入。
