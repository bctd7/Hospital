# 部门目录与医生管理接口

> 状态：待新增后端契约并联调

## 1. 数据来源约束

页面不内置科室或医生数据。公共科室、医生列表和数量由 `app-api` 返回，前端只保留短期查询缓存和
当前选中 ID。新增、编辑、停用或调整科室成功后，必须使相关缓存失效并重新请求。

当前后端尚不具备本页面所需的部门目录、按部门查询医生和部门维护完整契约。因此以下接口均属于
待实现规划，不能在前端假定已可用。
后端数据模型、事务和审计规则见
[Identity 组织、科室、医生与用户管理](../../backend/modules/04-organization-staff-and-seed-data.md)。

## 2. 公共科室与医生目录

科室和医生是预约前必须展示的公共业务目录。患者、医生、超级管理员和未登录访客使用相同的只读
接口，不复用会返回管理字段和停用数据的管理员列表：

```http
GET /api/v1/directory/departments
GET /api/v1/directory/departments/:departmentId/doctors?page=1&page_size=20
```

两个接口不要求工作人员权限，默认只返回有效科室和有效医生，并在网关实施公共接口限流。建议响应
最小化为：

```text
DepartmentSummary
  department_id, code, name, doctor_count, version

DoctorSummary
  doctor_id, display_name, department_id, avatar_url?, description?, version
```

不得返回完整手机号、验证码、OpenID、Token、资格审核材料或其他页面不需要的敏感字段。分页响应
应包含 `items`、`page`、`page_size`、`total`，排序规则由后端稳定定义。

`department_id` 是科室的唯一标识，名称修改后 ID 不变。前端列表 key、医生查询和预约请求都使用
`department_id`，不能使用科室名称判断是否为同一个科室。

## 3. 超级管理员部门管理接口

部门维护沿用现有权限 `identity.department.manage`：

```http
GET    /api/v1/admin/identity/organization-units?unit_type=department&status=all
GET    /api/v1/admin/identity/organization-units/:unitId
POST   /api/v1/admin/identity/organization-units
PUT    /api/v1/admin/identity/organization-units/:unitId
DELETE /api/v1/admin/identity/organization-units/:unitId
POST   /api/v1/admin/identity/organization-units/:unitId/enable
```

页面只提交和展示 `unit_type=department` 的组织单元；当前只有一个医院，存在多个院区时同时提交
合法的 `parent_id`，但不把完整组织树混入左右列表布局。

默认左右列表继续使用公共目录，只展示有效科室。超级管理员进入“已停用科室”管理视图时，使用
管理员查询获取停用数据并执行恢复；公共目录不能承担恢复查询。

删除部门采用停用语义，不做物理删除。部门写接口携带 `operation_id` 和当前组织单元 `version`，
用于幂等与乐观锁；前端不在成功响应前擅自修改权威数据。

开通医生、调岗、撤销医生身份和禁用账号统一由独立的
[超级管理员用户接口](./03-admin-user-management.md)承载，不在部门目录组件中直接调用。

## 4. 权限与错误处理

| 场景 | 建议状态/错误码 | 前端行为 |
|---|---|---|
| 会话过期 | `401` | 进入统一刷新流程，最多重试一次 |
| 公共目录触发限流 | `429` | 延迟后重试，不循环请求 |
| 管理权限不足 | `403` | 撤下管理入口，不重试写请求 |
| 部门不存在或已停用 | `404` / 业务码 | 刷新部门列表并重新选择 |
| 部门仍有子节点或医生 | `409` / 业务码 | 展示阻塞原因，不移除列表项 |
| 版本冲突 | `409` | 重新拉取详情后要求管理员确认 |
| 重复 operation_id | 幂等返回原结果 | 按成功结果刷新相关列表 |

错误消息由前端根据稳定业务码映射，不能把数据库错误或内部 RPC 文本直接展示给用户。

## 5. 缓存与刷新

- 科室列表使用短期内存缓存；应用版本变化或管理写入成功时清空；
- 医生列表按 `departmentId + page` 缓存，不预加载全部部门；
- 切换部门可先显示有效缓存，同时触发后台校验；
- 新增或停用部门后使部门列表缓存失效；
- 用户管理页开通、调岗或撤销医生后，使受影响科室的公共医生列表缓存失效；
- 使用请求序号防止迟到响应覆盖当前选择；
- `onShow` 只在缓存过期或收到失效标记时请求，避免 Tab 切换产生重复请求和卡顿。

## 6. 联调清单

- 未登录访客、普通用户、医生和超级管理员可以获取相同的公共目录结构；
- 科室改名后 `department_id` 保持不变，已有选择和关联不会依赖名称失效；
- 医生调用任一写接口均返回 `403`；
- 超级管理员权限缺失时同样不能写入；
- 部门及医生由数据库变化驱动，刷新后页面自动反映新增、停用和任职调整；
- 快速切换部门、分页、并发管理员修改和 Token 刷新均不会串数据；
- 响应中不包含完整手机号、OpenID、Token 或其他无关敏感字段。
