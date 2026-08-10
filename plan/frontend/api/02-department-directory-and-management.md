# 医院、院区、科室与医生目录接口

> 状态：医院、院区和科室契约及后端已落地，小程序已接入院区选择、院区新增和科室管理；医生目录及院区完整管理入口待后续联调

## 1. 范围和数据来源

首期组织范围固定为“唯一医院—多个院区—院区直属科室”。医院、院区、科室和医生全部由
`app-api` 动态返回；页面不得硬编码医院名称、院区或科室，也不得根据名称推断层级。

- 小程序展示唯一医院，但不提供医院切换或医院根节点管理；
- 患者、医生、超级管理员和未登录访客读取相同的有效医院目录；
- 超级管理员使用独立管理员接口维护院区和科室；
- 科室必须直接属于院区，首期不支持子科室；
- 小程序不再内置运行时 Mock；未实现的医生或管理员接口必须显示真实错误，不能回退到演示数据。

后端数据模型、事务、审计及停用规则见
[Identity 组织、科室、医生与用户管理](../../backend/modules/04-organization-staff-and-seed-data.md)。

## 2. 公共目录

```http
GET /api/v1/directory/organization-context
GET /api/v1/directory/departments?campus_id=:campusId
GET /api/v1/directory/departments/:departmentId/doctors?page=1&page_size=20
```

公共接口不要求工作人员权限，但必须在网关限流。数据结构固定为：

```text
HospitalSummary
  hospital_id: string
  code: string
  name: string
  version: integer

CampusSummary
  campus_id: string
  hospital_id: string
  code: string
  name: string
  department_count: integer
  status: active
  version: integer

DepartmentSummary
  department_id: string
  campus_id: string
  campus_name: string
  code: string
  name: string
  doctor_count: integer
  status: active
  version: integer

DoctorSummary
  doctor_id: string
  display_name: string
  department_id: string
  avatar_url?: string
  description?: string
  version: integer
```

### 2.1 组织上下文

`GET /organization-context` 返回：

```json
{
  "hospital": {
    "hospital_id": "hospital-id",
    "code": "HOSPITAL",
    "name": "演示医院",
    "version": 1
  },
  "campuses": []
}
```

- `hospital` 必须来自唯一有效医院根节点，不能由前端或网关伪造默认名称；
- `campuses` 只返回该医院下的有效院区；
- `department_count` 只统计院区下的有效直属科室；
- 院区按 `code + campus_id` 稳定排序；
- 唯一医院不存在或不可用属于服务配置错误，不返回空医院对象。

### 2.2 按院区查询科室

`GET /departments` 必须携带 `campus_id`。响应为：

```json
{ "items": [] }
```

- 只返回指定有效院区下的有效直属科室；
- 缺少或格式错误的 `campus_id` 返回 `400`；
- 院区不存在或已停用返回 `404`；
- `campus_id`、`campus_name` 必须随科室返回，不能让前端根据当前选择自行补齐权威数据；
- `doctor_count` 必须和该科室医生分页接口相同过滤条件下的 `total` 一致；
- 科室按 `code + department_id` 稳定排序。

### 2.3 按科室查询医生

医生响应为标准分页对象：

```json
{ "items": [], "page": 1, "page_size": 20, "total": 0 }
```

只返回账号正常、医生档案有效且属于目标科室的医生。不得返回完整手机号、OpenID、Token、角色、
账号状态、资格审核材料或管理员字段。

所有前端 key、筛选、缓存和预约提交必须使用稳定 ID；医院、院区、科室或医生名称只能展示，不能
作为关联条件。

## 3. 超级管理员院区与科室接口

院区和科室共用 `identity.department.manage` 权限：

```http
GET    /api/v1/admin/identity/organization-units?unit_type=campus|department&parent_id=&status=active|disabled|all
GET    /api/v1/admin/identity/organization-units/:unitId
POST   /api/v1/admin/identity/organization-units
PUT    /api/v1/admin/identity/organization-units/:unitId
DELETE /api/v1/admin/identity/organization-units/:unitId
POST   /api/v1/admin/identity/organization-units/:unitId/enable
```

管理员列表返回平铺结构，不返回整棵组织树：

```text
AdminOrganizationUnit
  unit_id: string
  parent_id?: string
  unit_type: campus | department
  code: string
  name: string
  status: active | disabled
  child_count: integer
  doctor_count: integer
  version: integer
```

- 查询院区时 `parent_id` 为唯一医院 ID；
- 查询科室时 `parent_id` 为选中院区 ID；
- 院区的 `child_count` 为有效直属科室数，科室的 `doctor_count` 为有效医生数；
- 不适用的计数字段返回 `0`；
- 本组接口拒绝 `unit_type=hospital`，小程序不能修改医院根节点。

### 3.1 新增

```text
POST /organization-units
  unit_type: campus | department
  parent_id: string
  name: string
  operation_id: UUID
```

- 新增院区时 `parent_id` 必须是唯一医院 ID；
- 新增科室时 `parent_id` 必须是用户明确选中的有效院区 ID；
- `parent_id` 不得省略，服务端不得随机或默认选择院区；
- 首版页面不录入 `code`，由服务端生成同一父节点下稳定唯一编码。

### 3.2 编辑和调整科室归属

```text
PUT /organization-units/:unitId
  name?: string
  parent_id?: string
  version: integer
  operation_id: UUID
```

- 院区只能修改名称，不允许修改 `parent_id`；
- 科室可以显式提交另一有效院区 ID 完成归属调整；
- 缺失 `parent_id` 表示保持原归属，不能解释为清空父节点；
- 不允许修改 ID、`unit_type` 或 `code`；
- 成功后返回最新 `AdminOrganizationUnit`。

### 3.3 停用和恢复

删除统一表示停用，不物理删除，也不级联：

- 院区存在有效科室时拒绝停用，返回 `409 ORGANIZATION_HAS_ACTIVE_CHILDREN`；
- 科室存在有效医生时拒绝停用，返回 `409 DEPARTMENT_HAS_ACTIVE_DOCTORS`；
- 恢复时父节点必须有效；
- 公共目录不返回停用院区或科室；管理员通过 `status=disabled|all` 查询并恢复；
- 写请求携带当前 `version` 和全局唯一 `operation_id`。

开通医生、调岗、撤销医生身份和禁用账号仍由
[超级管理员用户接口](./03-admin-user-management.md)承载，不放入目录列表行。

## 4. 错误处理

| 场景 | 状态/业务码 | 前端行为 |
|---|---|---|
| 会话过期 | `401` | 进入统一刷新流程，最多重试一次 |
| 公共目录限流 | `429` | 延迟后重试，不循环请求 |
| 管理权限不足 | `403` | 撤下管理入口，不重试写请求 |
| 缺少院区 ID | `400` | 阻止请求并恢复到有效院区选择 |
| 院区/科室不存在或已停用 | `404` | 刷新组织上下文并重新选择 |
| 层级不合法 | `422 INVALID_ORGANIZATION_HIERARCHY` | 保留表单并提示正确父级 |
| 院区仍有科室 | `409 ORGANIZATION_HAS_ACTIVE_CHILDREN` | 提示先处理科室，不移除院区 |
| 科室仍有医生 | `409 DEPARTMENT_HAS_ACTIVE_DOCTORS` | 提示先调岗或撤销身份 |
| 版本冲突 | `409` | 重新读取详情后要求管理员确认 |
| 重复 operation_id | 幂等返回原结果 | 按成功结果刷新缓存 |

错误消息由前端根据稳定业务码映射，不能直接展示数据库或 RPC 内部错误。

## 5. 缓存与失效

- 组织上下文按应用版本短期缓存，医院或院区管理成功后整体失效；
- 科室列表按 `campus_id` 缓存，不能使用一份全局科室数组；
- 医生列表按 `department_id + page` 缓存，不预加载全部院区、科室的医生；
- 切换院区时恢复该院区上次选中的有效科室，否则选择第一个有效科室；
- 新增、移动、停用或恢复科室后，使原院区和目标院区的科室缓存失效；
- 开通、调岗或撤销医生后，使受影响科室医生缓存和院区科室计数失效；
- 使用请求序号丢弃院区、科室快速切换产生的迟到响应；
- `onShow` 只在缓存过期或收到失效标记时重新请求。

## 6. 联调清单

- 所有身份看到相同的医院名称、有效院区、有效科室和有效医生；
- 多个院区可以存在同名科室，但通过不同 `campus_id`、`department_id` 正确区分；
- 切换院区后只显示该院区的直属科室，不串用上一院区缓存；
- 超级管理员可以管理院区和科室，但任何身份都不能通过小程序修改医院根节点；
- 新增科室必须明确提交院区 ID，不能依赖默认院区；
- 医生调用管理员写接口返回 `403`；
- 院区、科室或医生数据库变化后，刷新页面能够动态反映；
- 快速切换、分页、并发修改和 Token 刷新不会串数据；
- 响应不包含完整手机号、OpenID、Token 或无关敏感字段。
