# Identity 组织、科室、医生与用户管理

> 文档状态：待审核方案
>
> 目标：补齐 Identity Service 当前缺少的组织、医生与管理员用户管理能力，为预约、科室数据权限和前端预留稳定接口。
>
> 当前阶段只编写方案，不修改数据库、RPC、HTTP 接口或前端代码。

## 1. 为什么现在实现

手机号登录、Token、角色和医生开通主链路已经存在，但当前组织模型只能表达一张简单的科室树，
工作人员当前保存一个科室，但组织模型还不能明确表达医院和院区。预约、检查项目、号源和报告如果
直接引用这套临时结构，后续补充完整组织层级时会产生较大的迁移成本。

本阶段先建立以下业务基础：

1. 单医院下院区、科室的稳定组织层级；
2. 超级管理员新增、修改、停用和恢复组织单元；
3. 超级管理员将已注册账号开通为医生、维护任职科室和撤销医生身份；
4. 为所有用户提供同源的公共科室和医生目录，为超级管理员提供维护接口；
5. 为本地开发提供可重复导入、不会进入生产环境的 mock 数据。

它们仍然属于 Identity Service。当前不新建“科室服务”“医生服务”或“基础信息服务”，也不把
组织 Model 放进 `common`。

## 2. 首期业务范围

### 2.1 本期实现

- 一个医院及其院区、科室组成的树状组织；
- 组织单元列表、详情、新增、修改、停用和恢复；
- 所有用户都可以动态查询科室目录和科室下的医生列表；
- 超级管理员可以分页查询用户并查看最小必要的账号、身份和科室信息；
- 按完整手机号精确查找已经注册的账号；
- 用户可同步可选账号昵称，超级管理员可按昵称或医生显示名称分页查询候选；
- 手机号精确查询和昵称候选查询统一进入用户详情，再执行身份和账号操作；
- 将普通账号开通为医生；
- 医生基础身份资料和公开展示资料；
- 一个医生只能属于一个当前科室；
- 医生调岗时整体变更其当前科室；
- 撤销医生身份，但保留其普通用户账号和历史业务记录；
- 组织和医生变更的授权版本、审计记录和 Outbox 事件；
- 本地组织、科室和医生 mock 数据；
- 患者预约端、医生端和管理操作所需的 HTTP 接口。

### 2.2 本期不实现

- 医生执业证在线核验和完整审批流；
- 人力资源、工资、考勤和劳动关系管理；
- 结构化专业标签、出诊排班和预约容量；
- 从医院 HR、HIS 或统一身份系统自动同步；
- 批量导入真实生产数据；
- 科室合并、拆分和历史数据迁移工具；
- 超级管理员任意修改手机号、昵称或物理删除账号的通用账号 CRUD。

首版规模下不单独建立医疗人员目录。医生姓名、头像 URL、`description` 等变化频率较低的展示字段
直接保存在 Identity 医生档案中。这里的 `description` 只是一段“该医生擅长哪些内容”的固定
纯文本。出诊时间、可预约时段和剩余容量会参与预约事务，仍由 Appointment Service 管理。

## 3. 业务对象和数据归属

### 3.1 组织单元

系统当前只服务一个医院，使用统一组织单元表达其内部层级：

```text
演示医院（hospital）
└── 本部院区（campus）
    ├── 放射科（department）
    ├── 超声医学科（department）
    └── 检验科（department）
```

建议将当前 `identity_departments` 演进为 `identity_organization_units`：

| 字段 | 作用 |
|---|---|
| `id` | 稳定 UUID，供其他服务引用 |
| `parent_id` | 上级组织单元；医院根节点为空 |
| `unit_type` | `hospital`、`campus` 或 `department` |
| `code` | 稳定业务编码，不使用名称充当标识 |
| `name` | 当前展示名称 |
| `status` | `active` 或 `disabled` |
| `version` | 管理修改使用的乐观锁版本 |
| `created_at`、`updated_at` | 创建和修改时间 |

首期只允许存在一个医院根节点，不支持多医院或租户切换。医院下面建立院区，院区下面建立科室。
科室如确有上下级需要，可以允许科室下面建立子科室，但不能出现循环引用，也不能将医院挂在科室
下面。医院根节点由部署初始化，管理员可以修改基本信息，但不能创建第二个医院或删除医院根节点。

其他服务只保存组织单元 ID 和必要历史快照，不直接读写 Identity 数据库。例如预约可以保存
`department_id` 以及预约时的科室名称快照，但科室主数据只有 Identity 可以修改。

### 3.2 账号展示资料

为管理员昵称备选搜索增加服务端账号展示资料 `identity_account_profiles`，它与患者实名资料、医生档案
分开：

| 字段 | 作用 |
|---|---|
| `account_id` | 唯一对应 Hospital 账号 |
| `nickname` | 用户自行确认的可选昵称，允许重名，不作为登录或实名依据 |
| `created_at`、`updated_at` | 创建和修改时间 |

当前小程序 `hospital:display-profile` 只保存在设备 Storage，不能作为管理员在另一部手机登录同一
小程序时的搜索数据源。用户在就诊人管理页面确认昵称后，通过受保护接口同步到 Identity。昵称去除
首尾空白、限制长度并拒绝控制字符；管理员查询采用规范化前缀匹配和分页，不允许把昵称设为唯一键。

手机号仍是首选的精确定位方式。昵称只返回候选账号，超级管理员必须结合脱敏手机号、身份、状态和
科室核对详情；任何写操作最终都使用不可变 `account_id`。

### 3.3 医生档案

当前不区分“工作人员”和“医生”两套业务对象：存在有效医生档案的工作人员就是医生，公共目录也
只返回医生。现有数据库表 `identity_staff_profiles` 继续保存医生身份事实，不再额外建立
`staff_directory` 或 `doctor_directory` 表：

| 字段 | 作用 |
|---|---|
| `account_id` | 对应已经存在的账号 |
| `staff_no` | 可选工号，真实规则确认前不强制 |
| `display_name` | 医生显示名称 |
| `department_id` | 当前唯一所属科室 |
| `avatar_url` | 医生头像的 HTTP(S) 地址，未来可指向 OSS |
| `description` | 固定展示医生擅长内容的纯文本，例如“擅长腹部超声检查” |
| `staff_status` | `active` 或 `revoked` |
| `created_at`、`updated_at` | 创建和修改时间 |

姓名是否需要实名核验留待真实医院规则确认；首版 `display_name` 只作为工作人员管理展示数据，不能
自动证明医生资格。`avatar_url` 只保存地址，Identity 不下载、转存或代理图片；当前可使用普通
HTTP(S) 链接，接入 OSS 后更新为对象访问地址。`description` 只保存固定的擅长内容纯文本，不接受
HTML，也不承载排班、号源和实时状态。

排班和容量后续使用 Appointment Service 的独立表表达，而不是继续给工作人员档案加字段。概念上
至少会区分资源、排班规则、具体时段和预约占用：

```text
appointment_resources          医生、设备或检查室等可预约资源
appointment_schedule_rules     周一上午等周期性可用规则
appointment_slots              某一天某个具体时间段及总容量
appointment_slot_reservations  预约对具体时段容量的占用
```

最终表名和字段在 Appointment 模块计划中确定；Identity 只向 Appointment 提供稳定的医生
`account_id` 和当前 `department_id`。

### 3.4 管理并发版本

账号的 `authorization_version` 只服务 Token 权限变更，不应因为修改头像或擅长描述而失效。为用户
管理页面增加独立的 `management_version`：

- 账号昵称、医生公开资料、科室、医生身份或账号状态发生管理变更时递增；
- 管理员账号列表和详情明确返回 `management_version`；
- 账号、医生写请求携带当前 `management_version`，版本不一致返回 `409`；
- 涉及角色、科室或账号状态的变更同时递增 `authorization_version`；
- 只修改头像、显示名称或擅长描述时不递增 `authorization_version`，避免无意义地刷新 Token。

组织单元使用自己的 `version`，不能拿账号版本更新科室，也不能用时间字符串代替并发控制。

### 3.5 科室归属

当前业务规则明确限制一个医生只能属于一个科室，因此不新增医生—科室多对多任职表，也不引入
“主科室”概念。`department_id` 继续保存在 `identity_staff_profiles` 中，并且必须指向类型为
`department` 的有效组织单元。

约束：

- 开通医生时必须选择一个有效科室；
- 一个有效医生必须且只能有一个 `department_id`；
- 调岗使用一次原子更新，将原科室整体替换为新科室；
- 不能通过清空 `department_id` 表示离开科室，需要明确执行“撤销医生身份”；
- 停用的科室不能接收新医生；
- 调岗前后的科室 ID 进入授权审计，历史预约等业务记录不随调岗修改。

### 3.6 角色、任职和账号的区别

三个对象不能混为一个删除操作：

| 操作 | 结果 |
|---|---|
| 调整所属科室 | 将医生从原科室整体调到新科室，不能同时保留两个科室 |
| 撤销医生身份 | 撤销医生档案并移除医生角色，账号仍可作为普通用户登录 |
| 禁用账号 | 整个账号不可继续登录，属于更高风险的账号管理操作 |

前端即使都显示“删除”按钮，也必须根据对象调用不同接口并显示准确的确认提示。

## 4. 组织管理规则

### 4.1 新增

- 只有具有 `identity.department.manage` 权限的超级管理员可以新增；
- 部署初始化时必须已经存在唯一医院根节点，管理接口不允许创建第二个医院；
- 新增接口首期只接受 `campus` 和 `department`；
- `unit_type` 必须与父节点类型匹配；
- 同一父节点下编码不得重复；
- 名称允许以后修改，业务引用必须使用 ID；
- 请求携带 `operation_id`，重复提交返回同一个操作结果。

### 4.2 修改

首期允许修改名称和展示属性。以下变更需要限制：

- 已被引用的稳定 ID 永远不能修改；
- 已进入业务使用的组织编码原则上不修改；
- 不允许移动后造成层级循环；
- 当前只有一个医院，不提供跨医院移动和医院切换；
- 修改名称不反向覆盖历史预约中的名称快照。

### 4.3 删除、停用和恢复

超级管理员小程序页面的“删除科室”首期采用停用语义，不物理删除业务主数据：

```text
DELETE 组织单元
  -> 检查是否存在有效子节点
  -> 检查是否存在有效工作人员任职
  -> 存在则拒绝并返回具体原因
  -> 不存在则 status = disabled
  -> 写审计、Outbox 并提升受影响授权版本
```

- 有有效子科室时必须先处理子节点；
- 有有效医生时必须先调岗或撤销任职；
- 被预约、报告等历史业务引用不阻止停用，但禁止物理删除；
- 停用后不能创建新的业务关联，历史查询仍能识别原科室；
- 恢复时父级必须仍然有效。

仅本地 mock 数据可以通过专用的本地数据重置流程清理，生产管理接口不提供级联物理删除。

## 5. 医生管理流程

### 5.1 开通医生

复用当前“先注册、再按手机号查找、最后开通”的方案：

```text
超级管理员输入完整手机号
  -> Identity 计算手机号 HMAC 指纹
  -> 精确查找已注册账号
  -> 管理员线下核对人员身份
  -> 填写显示名称、可选工号、任职科室、头像 URL 和擅长描述
  -> 确认线下已核验
  -> 原子创建医生档案、唯一所属科室和医生角色
  -> authorization_version + 1
  -> 写审计和 Outbox
```

如果手机号尚未注册，只返回“未找到可开通账号”，不由管理员代替用户创建手机号账号。完整手机号、
HMAC Key 和验证信息不得进入普通日志。

### 5.2 修改医生

允许超级管理员：

- 修改显示名称、可选工号、头像 URL 和擅长描述；
- 将当前所属科室整体调整为另一个有效科室；
- 修改工作人员状态。

调岗必须一次提交新科室并原子更新，不能先删除旧科室、再添加新科室，否则会产生医生暂时没有科室
或只完成一半的状态。

### 5.3 撤销医生身份

撤销不是删除账号：

```text
确认目标不是最后一个可用超级管理员
  -> staff_status = revoked
  -> 移除 department_doctor 角色
  -> 没有其他工作人员身份时恢复为 patient 账号类型
  -> authorization_version + 1
  -> 撤销 Refresh Session
  -> 写审计和 Outbox
```

已经签发的短期 Access Token 最迟在过期后失效；高风险环境如需要立即失效，应结合 Redis 授权版本
或撤销记录校验。首期至少撤销所有 Refresh Session，禁止继续刷新旧权限。

医生创建过的预约处理记录、检查记录和审计记录不能随身份撤销而删除。

### 5.4 统一用户定位和账号禁用

开通医生、调岗、撤销医生身份、禁用和恢复账号统一从管理员用户搜索开始：

```text
完整手机号
  -> 规范化并计算 HMAC 指纹
  -> 精确查找绑定账号

昵称或医生显示名称
  -> 服务端前缀匹配并分页返回候选
  -> 管理员结合脱敏手机号、身份、科室和状态核对

选中账号
  -> 读取 account_id、management_version、available_actions
  -> 进入详情执行对应身份或账号操作
```

手机号和昵称只用于定位账号，禁用、恢复、开通医生、调岗和撤销等写接口始终接收 `account_id`。
手机号可能换绑，昵称也可能重复或修改，不能直接作为写操作主键。

禁用账号后的短信登录流程：

```text
发送验证码
  -> 维持统一响应，不提前暴露账号状态
验证码校验成功
  -> 手机号指纹定位到已有账号
  -> status = disabled
  -> 返回 ACCOUNT_DISABLED，不签发 Access Token / Refresh Token
```

禁用账号不等同于禁止发送验证码；短信轰炸、异常号码和费用控制属于独立风控状态及手机号/IP 限流。
实现时必须将 `ErrInactiveAccount` 映射为稳定的账号禁用业务错误，不能返回笼统 `500`。

## 6. 权限设计

首期复用当前权限目录，不在页面规划阶段发明与代码不一致的权限码：

| 能力 | 授权规则 |
|---|---|
| 查看有效科室及科室内医生 | 所有人可访问公共只读接口，不要求医生或管理员权限 |
| 新增、修改、停用和恢复部门 | `identity.department.manage` |
| 查询管理员用户列表和详情 | `identity.authorization.manage` 或 `identity.account.manage` |
| 开通、调岗和撤销医生身份 | `identity.authorization.manage` |
| 禁用和恢复普通账号 | `identity.account.manage`，不属于部门目录操作 |

患者、医生、超级管理员以及未登录访客调用同一组公共只读目录接口，获得相同结构的有效科室和有效
医生数据。公共查询不授予任何写权限；所有新增、修改、停用、恢复、调岗和撤销操作仍只允许超级
管理员按 permission 执行。公共接口需要限流和缓存，但不要求 `department_doctor` 或
`super_admin` 身份。

`app-api` 使用 Access Token 中的权限提前拒绝无权请求；Identity RPC 对高风险写操作再次查询最新
账号和权限事实，不能仅相信客户端传入的操作者 ID。

## 7. 为前端预留的 HTTP 接口

所有数据由 `app-api` 从 Identity Service 动态查询，前端不得内置科室或医生清单。公共只读目录
位于 `/api/v1/directory`，小程序内的患者预约页、医生页和超级管理员页共用；写接口位于
`/api/v1/admin/identity`，仅超级管理员按 permission 调用。HTTP 层负责参数转换、限流、管理接口
鉴权和响应映射，最终读写由 Identity RPC 完成。

### 7.1 本人账号展示资料

```http
GET /api/v1/auth/me/display-profile
PUT /api/v1/auth/me/display-profile
```

接口要求 Access Token。`PUT` 首期只接受昵称，完成裁剪、长度和控制字符校验后写入
`identity_account_profiles`；昵称修改提升 `management_version`，但不提升 `authorization_version`
或使 Token 失效。返回值不得把昵称描述为真实姓名。

### 7.2 公共科室与医生目录

```http
GET /api/v1/directory/departments
GET /api/v1/directory/departments/:departmentId/doctors?page=1&page_size=20
```

- 两个接口允许所有人查询，不要求工作人员或管理员权限；
- 科室列表只返回有效的 `department`，包含稳定 `department_id`、编码、名称、医生数量和版本；
- 医生列表按选中部门分页查询，只返回医生 ID、显示名称、部门 ID、头像 URL、擅长描述和版本；
- 对外字段使用 `doctor_id`；首期它直接取内部稳定 `account_id`，不为同一个人再创建第二套 ID；
- 不返回完整手机号、OpenID、Token、内部审计信息或管理员专用字段；
- 部门数量和医生列表由数据库实时事实生成，不能由 API 或前端硬编码；
- 部门人数使用聚合查询，医生列表使用按 `department_id` 的分页索引，禁止逐部门 N+1 查询；
- 排序规则稳定，建议部门按展示顺序和编码，医生按显示名称、工号及账号 ID 兜底排序。

前端按部门懒加载医生列表，不要求后端一次返回全部部门的全部医生。开发环境的 seed 数据也必须
先写入本地数据库再经过同一接口返回，不能形成另一套前端静态数据源。

`department_id` 是识别同一个科室的唯一技术标识。它由后端生成并且永不复用；科室改名、停用或
恢复时 ID 都不改变。`name` 只负责展示，不能用作数据库关联、前端列表 key 或预约请求参数；`code`
用于后台录入和数据导入，也不能替代 ID。调用流程固定为：先查询科室得到 `department_id`，再使用
该 ID 查询医生或创建预约。

### 7.3 管理员组织接口

```http
GET    /api/v1/admin/identity/organization-units
GET    /api/v1/admin/identity/organization-units/:unitId
POST   /api/v1/admin/identity/organization-units
PATCH  /api/v1/admin/identity/organization-units/:unitId
DELETE /api/v1/admin/identity/organization-units/:unitId
POST   /api/v1/admin/identity/organization-units/:unitId/enable
```

列表支持 `parent_id`、`unit_type`、`status` 查询，并可返回树形结果。`DELETE` 的业务语义是停用，
不是从数据库物理删除。

公共目录只包含有效科室。超级管理员查看或恢复停用科室时，调用管理员列表并指定
`status=disabled` 或 `status=all`；不能要求前端从公共目录恢复已经被过滤的数据。预约页面不能直接
复用包含停用数据和管理字段的管理员响应。

### 7.4 管理员用户、医生与账号接口

保留现有接口：

```http
POST /api/v1/admin/identity/accounts/search-by-phone
POST /api/v1/admin/identity/doctors/promote
```

本期补充用户列表和详情：

```http
GET /api/v1/admin/identity/accounts?page=1&page_size=20&nickname=&identity_type=&status=&department_id=
GET /api/v1/admin/identity/accounts/:accountId
```

- `identity_type` 是根据账号、角色和有效医生档案计算出的 `patient`、`doctor` 或 `super_admin`，
  不新增一列重复保存；
- 支持按身份、账号状态和科室筛选；`nickname` 查询账号昵称和医生 `display_name`，采用前缀匹配；
- 普通账号昵称可以为空，前端使用脱敏手机号作为展示兜底，不允许 Identity 跨库连接未来的 Patient
  数据库完成分页；
- 完整手机号继续使用现有精确查询接口，分页和详情只返回脱敏手机号；
- 列表返回稳定账号 ID、可选昵称、可选医生显示名称、可选头像、脱敏手机号、身份、可选科室、账号状态和
  `management_version`；
- `management_version` 与 Token 使用的 `authorization_version` 分开；
- 详情补充手机号验证状态、创建/更新时间、角色、可选工号、医生公开资料、医生状态、
  `authorization_version` 及后端计算的 `available_actions`；
- 查询必须数据库分页并限制最大页大小，不允许读出全部账号后在内存中过滤。

手机号搜索通过现有 `POST .../search-by-phone` 精确返回账号；昵称允许重名，只返回分页候选。两条搜索
链路最终都进入相同用户详情，并使用详情中的 `account_id` 执行开通医生、调岗、撤销、禁用或恢复。

分页列表和详情不回显完整手机号：当前数据库只保存手机号 HMAC 指纹和脱敏值，没有可恢复明文。
若以后确需回显，必须单独设计加密存储、密钥轮换和访问审计，不能把手机号改为数据库明文。

`available_actions` 根据操作者最新权限和目标状态计算，可能包含 `promote_doctor`、`edit_doctor`、
`change_department`、`revoke_doctor`、`disable_account`、`enable_account`。它帮助前端生成操作区，但
不能代替写接口再次鉴权。

本期确定的写接口：

```http
PATCH  /api/v1/admin/identity/doctors/:accountId
PUT    /api/v1/admin/identity/doctors/:accountId/department
DELETE /api/v1/admin/identity/doctors/:accountId
POST   /api/v1/admin/identity/accounts/:accountId/disable
POST   /api/v1/admin/identity/accounts/:accountId/enable
```

- `PUT .../department` 原子替换医生当前唯一所属科室；
- `DELETE .../doctors/:accountId` 表示撤销医生身份，不删除账号；
- `PATCH .../doctors/:accountId` 可以维护显示名称、工号、头像 URL 和擅长描述；
- `POST .../disable` 禁用账号所有登录身份，`POST .../enable` 只恢复登录能力；
- 首版拒绝通过用户管理接口禁用、恢复或变更任何 `super_admin`；
- 写接口必须携带 `operation_id`，避免网络重试造成重复变更。

禁用医生账号时保留医生角色和医生档案，只阻止登录；重新启用后仍恢复原医生身份。只有先执行
“撤销医生身份”才会移除医生角色，之后再启用账号也不会自动重新开通医生。

这些接口供超级管理员专属“用户管理”二级页面使用。部门管理页只展示公共科室和医生目录，不直接
排列调岗、撤销或禁用按钮。用户详情读取以及所有身份和账号写操作均记录操作者、目标账号、请求 ID、
operation ID 和结果；不得在响应、审计或普通日志中记录完整手机号。

## 8. RPC 边界

Identity RPC 需要补充与 HTTP 对应的内部能力：

- `ListOrganizationUnits`、`GetOrganizationUnit`；
- `GetAccountDisplayProfile`、`UpdateAccountDisplayProfile`；
- `ListPublicDepartments`、`ListDoctorsByDepartment`；
- `ListAdminAccounts`、`GetAdminAccount`；
- `CreateOrganizationUnit`、`UpdateOrganizationUnit`；
- `DisableOrganizationUnit`、`EnableOrganizationUnit`；
- `UpdateDoctor`；
- `ChangeDoctorDepartment`、`RevokeDoctor`；
- `DisableAccount`、`EnableAccount`；
- 扩展现有 `PromoteToDepartmentDoctor`，补充工作人员资料和组织单元校验。

每个写 RPC 必须在 Identity 数据库的本地事务中同时完成：

1. 校验操作者最新权限；
2. 校验目标账号和组织状态；
3. 更新主数据；
4. 所有账号/医生管理写入提升 `management_version`；只有角色、科室或账号状态变化才同时提升
   `authorization_version`；
5. 写入 `identity_authorization_audit`；
6. 写入 `identity_outbox_events`；
7. 提交事务后处理 Refresh Session 撤销。

管理员用户详情属于敏感读取，需要记录操作者、目标账号、request ID 和结果；普通分页列表和公共
目录记录访问日志与指标，但不为每一行分别写业务审计。

不要由 `app-api` 直接连接 Identity 数据库，也不要把组织 Repository 提取到 `common` 供其他服务调用。

## 9. mock 数据设计

### 9.1 存放位置

本地开发数据与数据库迁移分开：

```text
seeds/
└── local/
    └── identity/
        ├── organizations.yaml
        └── doctors.yaml

tools/
└── dev-seed/
    └── main.go

scripts/
└── seed-local.ps1
```

- `organizations.yaml` 保存明确标记为演示数据的医院、院区和科室；
- `doctors.yaml` 保存可选的虚假医生资料、示例头像 URL 和简介，不保存真实个人信息；
- `dev-seed` 读取配置并通过 Identity 的业务写入能力幂等导入；
- `seed-local.ps1` 负责环境检查和统一调用。

mock 数据不写入 `migrations/identity/*.sql`，也不放进 Compose 的 MySQL 初始化目录。数据库迁移只负责
结构和系统固定权限数据，不能让生产迁移自动产生演示科室或测试医生。

### 9.2 安全约束

- 只允许 `APP_ENV=local` 或测试环境执行；
- 使用固定、明确的演示 UUID，重复执行不产生重复记录；
- 默认不清空已有数据库，也不覆盖人工维护的数据；
- 数据名称统一使用“演示医院”“测试医生”等明显标识；
- 不包含真实手机号、身份证、AccessKey、AppSecret、Token 或验证码；
- 生产环境检测到 local seed 模式必须拒绝执行；
- CI 测试使用独立 `testdata`，不依赖开发者本地 seed。

### 9.3 mock 医生登录

只导入医生资料并不等于测试医生能够登录。当前短信链路使用真实阿里云 PNVS，虚假手机号无法接收
验证码。若需要测试多个医生账号，可以增加本地短信提供者：

```text
APP_ENV=local
SMS_PROVIDER=local
LOCAL_SMS_CODE=<仅保存在本地环境>
```

本地提供者只能在 `local/test` 环境启用；其他环境选择 `local` 时服务必须启动失败。固定验证码不写入
仓库、不打印到日志，真实阿里云发送路径不受影响。测试账号导入作为显式可选项，不能默认创建。

## 10. 错误和并发处理

至少需要定义并映射以下业务错误：

- 组织单元不存在、已停用或层级不合法；
- 组织编码冲突；
- 组织仍有有效子节点；
- 科室仍有有效工作人员；
- 目标账号不存在、已禁用或已经是医生；
- 任职科室不存在或不是科室类型；
- 医生没有科室或目标组织单元不是科室；
- 头像不是合法的 HTTP(S) URL，或擅长描述超过长度限制；
- 昵称为空、含控制字符或超过长度限制；
- 操作者权限不足；
- 重复 `operation_id`；
- 并发更新版本冲突。

组织和工作人员更新建议携带 `updated_at` 或显式 `version` 做乐观锁，防止两个管理员互相覆盖。所有
失败响应和日志继续使用脱敏手机号、`request_id`、`operation_id`、操作者 ID 和目标 ID，不能记录
完整手机号或 Token。

## 11. 实施顺序

计划审核通过后，从最新 `main` 创建新的业务分支，按以下顺序实现：

1. 增加新的 Identity 迁移，演进组织单元、账号展示资料和医生档案；
2. 更新 permission 契约、角色初始化和授权测试；
3. 更新 Identity Proto 并生成 RPC 骨架；
4. 实现 Repository、事务、审计、Outbox 和授权版本更新；
5. 更新 `app-api` 的 `.api` 契约并生成 HTTP 骨架；
6. 实现本人昵称同步、公共科室和医生目录以及管理员组织、用户、医生和账号接口；
7. 实现 local seed 工具与环境保护；
8. 如联调需要，再实现严格受限的 local SMS Provider；
9. 补充单元测试、MySQL/Redis 集成测试和 API 权限测试；
10. 分别使用未登录访客、普通用户、医生和超级管理员验证公共目录；
11. 使用超级管理员完成一次组织维护、医生开通、调岗、撤销和普通用户回退的联调回归。

本阶段完成页面与接口方案，不要求同步实现超级管理员小程序页面代码；接口、契约、错误码和 mock 数据准备好后，
前端再按独立任务开始开发。

## 12. 验收标准

- 可以建立“医院—院区—科室”层级，非法层级和循环关系被拒绝；
- 未登录访客、普通用户、医生和超级管理员都可通过同一公共接口获取有效科室及选中科室的医生；
- 部门或医生数据库数据发生变化后，接口结果随之变化，不依赖前端硬编码清单；
- 超级管理员可以新增、修改、停用和恢复组织单元；
- 有子节点或有效医生的科室不能直接停用；
- 已注册普通账号可以被开通为某一个科室的医生；
- 超级管理员可以分页查询用户并查看账号、身份、科室和可用操作；
- 用户昵称同步到后端后，可在不同手机登录同一微信小程序时读取；昵称允许重名且不被当作实名；
- 超级管理员可以用完整手机号精确定位账号，也可以按昵称或医生显示名称查询分页候选；
- 开通医生、调岗、撤销医生身份、禁用和恢复账号最终都按选中账号的 `account_id` 执行；
- 一个有效医生始终有且只有一个所属科室；
- 医生详情可以返回头像 URL 和纯文本擅长描述，修改后动态生效；
- 调岗、撤销医生和禁用账号具有不同结果；
- 禁用账号后所有身份不能登录，恢复账号不会自动恢复已撤销的医生身份；
- 禁用账号在验证码校验成功后返回 `ACCOUNT_DISABLED` 且不签发 Token；验证码发送阶段不暴露状态；
- 首版用户管理不能禁用、恢复或变更超级管理员身份；
- 撤销医生后账号仍可作为普通用户使用，旧 Refresh Session 不能继续获得医生权限；
- 普通用户和医生可以调用公共目录，但不能调用任何组织或医生管理接口；
- 管理操作均写入审计与 Outbox，重复请求不会重复变更；
- local seed 可重复执行且不会进入生产环境；
- 全部接口不泄露完整手机号、验证码、Token 和密钥；
- 数据迁移支持全新数据库升级，并有相应回滚或前向修复说明；
- `scripts/check.ps1` 和 CI 全部通过。

## 13. 需要审核确认的决定

进入代码实现前需要确认：

1. 是否接受使用统一组织单元表表达唯一医院、院区和科室；
2. “删除科室”是否接受统一采用停用语义；
3. 撤销医生身份后是否确认保留普通用户身份和全部历史记录；
4. 本地联调是否需要固定验证码的 local SMS Provider；
5. 首版工作人员资料是否确定为显示名称、可选工号、唯一所属科室、头像 URL 和擅长描述。
