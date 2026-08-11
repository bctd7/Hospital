# Identity 组织、科室、医生与用户管理

> 文档状态：组织、医生与账号 CRUD、公共目录、昵称同步和小程序联调已落地；local seed 工具仍按本文后续阶段实施
>
> 目标：补齐 Identity Service 当前缺少的组织、医生与管理员用户管理能力，为预约、科室数据权限和
> 超级管理员部门/用户页面提供稳定接口。
>
> 本文是后端实施基线；页面与前端调用约束见
> [工作人员部门管理](../../frontend/pages/04-department-management.md)、
> [超级管理员用户管理](../../frontend/pages/05-admin-user-management.md)及其 API 文档。

当前已完成的组织阶段包括：组织表与审计/Outbox 迁移、Repository、Manager、公共目录 RPC、管理员
组织 CRUD RPC、app-api HTTP 入口、稳定错误映射以及 MySQL 集成测试。小程序已移除运行时 Mock，
先读取医院与有效院区，由用户显式选择当前院区后再加载直属科室和医生；超级管理员可以完整维护院区、
科室、医生身份和账号状态。本人昵称已同步到 Identity，管理员用户列表、详情和手机号精确查询也已接通。
严格受限的 local SMS Provider 已实现，生产环境误选 `local` 时服务会拒绝启动。当前只剩独立的 local
seed 工具未实施；它不属于生产 CRUD，也不影响本轮真实数据联调。

## 1. 为什么现在实现

手机号登录、Token、角色和医生开通主链路已经存在，但当前组织模型只能表达一张简单的科室树，
工作人员当前保存一个科室，但组织模型还不能明确表达医院和院区。预约、检查项目、号源和报告如果
直接引用这套临时结构，后续补充完整组织层级时会产生较大的迁移成本。

本阶段先建立以下业务基础：

1. 单医院下多个院区及其直属科室的稳定组织层级；
2. 超级管理员新增、修改、停用和恢复院区、科室；医院根节点只读展示；
3. 超级管理员将已注册账号开通为医生、维护任职科室和撤销医生身份；
4. 为所有用户提供同源的公共科室和医生目录，为超级管理员提供维护接口；
5. 为本地开发提供可重复导入、不会进入生产环境的 mock 数据。

它们仍然属于 Identity Service。当前不新建“科室服务”“医生服务”或“基础信息服务”，也不把
组织 Model 放进 `common`。

## 2. 首期业务范围

### 2.1 本期实现

- 一个医院、多个院区和院区直属科室组成的三层组织；
- 医院基本信息公共只读展示，院区和科室支持列表、详情、新增、修改、停用和恢复；
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
- 多医院切换、创建第二个医院、在小程序中修改医院根节点；
- 子科室以及任意深度组织树；
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

实施时将当前 `identity_departments` 演进为 `identity_organization_units`：

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

首期只允许存在一个医院根节点，不支持多医院或租户切换。医院下面可以建立多个院区，科室必须
直接挂在某个院区下；首期明确禁止科室下再建子科室，也不支持任意层级移动。医院根节点由部署初始化，
小程序只读取并展示其名称等基本信息，不提供创建、修改、停用或删除医院根节点的管理能力。

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
- 超级管理员小程序首版不录入编码。创建 `campus` 或 `department` 时由服务端生成稳定且同一父节点下唯一的
  `code`，并在响应中返回；后续接入人工编码或 HIS 编码前，不得要求当前页面补传该字段；
- 名称允许以后修改，业务引用必须使用 ID；
- 请求携带 `operation_id`，重复提交返回同一个操作结果。

### 4.2 修改

首期允许修改院区和科室名称；科室可以显式调整到另一个有效院区。以下变更需要限制：

- 已被引用的稳定 ID 永远不能修改；
- 已进入业务使用的组织编码原则上不修改；
- 院区不允许移动父节点，科室目标父节点必须是有效院区；
- 当前只有一个医院，不提供跨医院移动和医院切换；
- 修改名称不反向覆盖历史预约中的名称快照。

### 4.3 删除、停用和恢复

超级管理员小程序页面的“删除院区/科室”首期采用停用语义，不物理删除业务主数据：

```text
DELETE 组织单元
  -> 检查是否存在有效子节点
  -> 检查是否存在有效工作人员任职
  -> 存在则拒绝并返回具体原因
  -> 不存在则 status = disabled
  -> 写审计、Outbox 并提升受影响授权版本
```

- 院区有有效直属科室时必须先处理科室；
- 科室有有效医生时必须先调岗或撤销任职；
- 被预约、报告等历史业务引用不阻止停用，但禁止物理删除；
- 停用后不能创建新的业务关联，历史查询仍能识别原院区或科室；
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

账号曾经是医生、后来被撤销医生身份时，原医生档案以 `revoked` 状态保留。再次开通不能插入第二条
医生档案，而是在同一事务中重新激活原档案、覆盖为本次确认的唯一科室和公开资料、重新授予医生角色，
并递增管理版本和授权版本。账号仍处于 `disabled` 时必须先恢复账号，不能通过开通医生顺便绕过禁用。

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

统一拦截器通过 Redis 授权版本校验使已经签发的旧 Access Token 在下一次请求时失效。普通角色撤销
或调岗返回 `401` 后，客户端优先通过全局 `refreshOnce()` 无感取得权限收缩后的新 Token，并将原请求
最多重试一次；账号禁用、Refresh Session 过期或显式撤销会导致刷新失败，此时客户端清理本地会话并
回到登录流程。撤销医生身份是否同时撤销全部 Refresh Session 按安全策略决定；一旦撤销，就明确放弃
无感刷新并要求重新登录。

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

首期复用当前权限目录，不在本模块另造与权限契约不一致的权限码：

| 能力 | 授权规则 |
|---|---|
| 查看医院、有效院区、有效科室及科室内医生 | 所有人可访问公共只读接口，不要求医生或管理员权限 |
| 新增、修改、停用和恢复院区、科室 | `identity.department.manage` |
| 查询管理员用户列表和详情 | `identity.authorization.manage` 或 `identity.account.manage` |
| 开通、调岗和撤销医生身份 | `identity.authorization.manage` |
| 禁用和恢复普通账号 | `identity.account.manage`，不属于部门目录操作 |

患者、医生、超级管理员以及未登录访客调用同一组公共只读目录接口，获得相同的医院、有效院区、
有效科室和有效医生数据。公共查询不授予任何写权限；所有新增、修改、停用、恢复、调岗和撤销操作仍只允许超级
管理员按 permission 执行。公共接口需要限流和缓存，但不要求 `department_doctor` 或
`super_admin` 身份。

`app-api` 和 Identity RPC 的统一认证拦截器先完成 JWT 验签，再通过 Redis 授权版本校验器确认 Token
中的 `authorization_version` 仍是当前版本。Logic 必须从 Context 取得已经验证的完整
`authn.Principal`，不能相信客户端传入的操作者 ID；Manager 使用 `common/authz` 判断 permission，
不再由每个业务模块分别查询操作者的角色和权限。Identity Repository 仍负责目标账号、组织和医生的
最新业务事实及事务锁定。

Redis 授权版本读取和校验不得放在 `service/identity/rpc/internal` 后再要求其他服务复制。公共接口、
Validator、HTTP/gRPC 拦截器位于 `common/authn`，可复用 Redis Reader 位于
`common/authn/versionredis`；每个服务仅在自身 `ServiceContext` 注册和注入这组公共组件。Identity 是
授权版本唯一写入者，Appointment、Planning、Report、Navigation 等业务服务只能读取共享版本并使用
Context 中的 Principal 完成本地业务授权。

## 7. 前端已确认的 HTTP 契约

本节是 `app-api` 和 Identity RPC 的实施依据，字段名与前端 API 文档保持一致。所有 JSON 字段使用
`snake_case`；列表统一使用 `items`，分页列表额外返回 `page`、`page_size` 和 `total`。单条查询和
写操作直接返回资源对象，不再套 `data`。公共只读目录位于 `/api/v1/directory`；管理员接口位于
`/api/v1/admin/identity`，必须使用 Hospital Access Token 并按 permission 鉴权。

现有 `POST /accounts/search-by-phone` 和 `POST /doctors/promote` 只返回授权上下文，不能满足新页面。
实施本模块时必须在 `.api`、Proto、生成代码和 Handler 中同步升级为本节响应，不允许让前端同时兼容
旧、新两套响应。

### 7.1 共用响应对象

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
  status: active | disabled
  version: integer

DepartmentSummary
  department_id: string
  campus_id: string
  campus_name: string
  code: string
  name: string
  doctor_count: integer
  status: active | disabled
  version: integer

DoctorSummary
  doctor_id: string
  display_name: string
  department_id: string
  avatar_url?: string
  description?: string
  version: integer

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

AdminAccountSummary
  account_id: string
  nickname?: string
  display_name?: string
  avatar_url?: string
  masked_phone?: string
  account_status: active | disabled
  identity_type: patient | doctor | super_admin
  department_id?: string
  department_name?: string
  management_version: integer

AdminAccountDetail extends AdminAccountSummary
  phone_verification_status?: string
  staff_no?: string
  description?: string
  roles: string[]
  authorization_version: integer
  available_actions: AdminAccountAction[]
  created_at: RFC3339 string
  updated_at: RFC3339 string
```

`doctor_id` 首期直接使用稳定 `account_id`。`CampusSummary.status` 和 `DepartmentSummary.status` 在
公共接口中固定为 `active`，仍
必须返回，以便公共目录和管理员列表复用同一前端类型。可选字符串无值时应省略，不得用空字符串或
`null` 伪造已填写数据；数组必须返回空数组而不是 `null`。

`identity_type` 是展示字段，不落冗余列，计算优先级固定为 `super_admin -> doctor -> patient`。账号
被禁用但医生档案和角色仍有效时仍返回 `doctor`，同时 `account_status=disabled`；撤销医生身份后才
返回 `patient`。`management_version` 与 Token 使用的 `authorization_version` 含义不同。

### 7.2 本人账号展示资料

```http
GET /api/v1/auth/me/display-profile
PUT /api/v1/auth/me/display-profile
```

接口要求 Access Token。`PUT` 首期只接受 `nickname`，完成裁剪、长度和控制字符校验后写入
`identity_account_profiles`；昵称修改提升 `management_version`，但不提升 `authorization_version`
或使 Token 失效。返回值不得把昵称描述为真实姓名。

### 7.3 公共医院、院区、科室与医生目录

```http
GET /api/v1/directory/organization-context
GET /api/v1/directory/departments?campus_id=:campusId
GET /api/v1/directory/departments/:departmentId/doctors?page=1&page_size=20
```

- 组织上下文响应为 `{ "hospital": HospitalSummary, "campuses": CampusSummary[] }`，医院为唯一根节点，
  院区只包含有效 `campus`；如果医院根节点不存在或不可用，返回服务端配置错误而不是伪造名称；
- 科室响应为 `{ "items": DepartmentSummary[] }`，`campus_id` 必填且只返回该有效院区下的有效
  `department`；缺失或非法 `campus_id` 返回 `400`，院区不存在或已停用返回 `404`；
- 医生响应为 `{ "items": DoctorSummary[], "page": 1, "page_size": 20, "total": 0 }`，只包含账号正常、
  医生档案有效且属于目标科室的医生；
- 三个接口允许所有人查询，不要求工作人员或管理员权限；
- `doctor_count` 必须与相同过滤条件下医生分页接口的 `total` 一致；
- 科室 `version` 来自组织单元版本，医生 `version` 来自账号 `management_version`；
- `page` 从 1 开始，默认 `page_size=20`，服务端限制最大值；非法分页参数返回 `400`；
- `department_count` 只统计院区下的有效科室；部门数量使用聚合查询，医生列表使用
  `department_id` 分页索引，禁止逐院区、逐部门 N+1 查询；
- 排序必须稳定：院区、部门按编码和 ID，医生按显示名称、工号和账号 ID 兜底。

前端按部门懒加载医生，不要求后端一次返回全部医生。开发 seed 必须先写入本地数据库再通过同一接口
返回。接口不得返回手机号、OpenID、Token、角色、账号状态、审计信息或其他管理员字段。

### 7.4 超级管理员院区与科室接口

```http
GET    /api/v1/admin/identity/organization-units?unit_type=campus|department&parent_id=&status=all
GET    /api/v1/admin/identity/organization-units/:unitId
POST   /api/v1/admin/identity/organization-units
PUT    /api/v1/admin/identity/organization-units/:unitId
DELETE /api/v1/admin/identity/organization-units/:unitId
POST   /api/v1/admin/identity/organization-units/:unitId/enable
```

列表返回平铺的 `{ "items": AdminOrganizationUnit[] }`，不返回整棵树。`unit_type` 必须明确为
`campus` 或 `department`，不能查询或管理医院根节点；查询科室时 `parent_id` 必须是院区 ID。
`status` 支持 `active`、`disabled`、`all`，默认 `active`。院区的 `child_count` 统计有效直属科室，
科室的 `doctor_count` 统计有效医生；不适用的计数字段返回 `0`。详情及所有写操作返回最新
`AdminOrganizationUnit`。

请求体固定为：

```text
POST /organization-units
  unit_type: campus | department
  parent_id: string
  name: string
  operation_id: UUID

PUT /organization-units/:unitId
  name?: string
  parent_id?: string
  version: integer
  operation_id: UUID

DELETE /organization-units/:unitId
POST /organization-units/:unitId/enable
  version: integer
  operation_id: UUID
```

首版页面不录入 `code`，服务端创建时生成并返回同一父节点下稳定唯一的编码；编辑接口不得修改 ID、
`unit_type` 或 `code`。创建院区时 `parent_id` 必须是唯一医院根 ID；创建科室时 `parent_id` 必须是
有效院区 ID，禁止省略、禁止由服务端随机或默认选择院区。`PUT` 只修改实际提交的字段；科室可通过
显式提交新的有效院区 `parent_id` 调整归属，院区不得移动到其他父节点。JSON 中缺失 `parent_id`
表示保持不变，不能解释为清空父级。

`DELETE` 只停用，不物理删除。院区存在有效科室时返回
`409 ORGANIZATION_HAS_ACTIVE_CHILDREN`；科室存在有效医生时返回
`409 DEPARTMENT_HAS_ACTIVE_DOCTORS`。恢复时父节点必须有效，禁止级联停用或恢复。公共目录只包含
有效院区和有效科室；“已停用院区/科室”视图必须调用本组管理员列表。

### 7.5 超级管理员用户查询

```http
GET  /api/v1/admin/identity/accounts?page=1&page_size=20&nickname=&identity_type=&status=&department_id=
GET  /api/v1/admin/identity/accounts/:accountId
POST /api/v1/admin/identity/accounts/search-by-phone
```

列表返回 `AdminAccountSummary` 分页对象，详情返回 `AdminAccountDetail`。`nickname` 同时对账号昵称和医生
`display_name` 做规范化前缀匹配，允许重名；`identity_type`、`status` 和 `department_id` 是可选精确
筛选。查询必须在数据库分页并限制最大页大小，不得读出全部账号后在内存中过滤。

完整手机号精确查询沿用现有包装，响应升级为：

```text
SearchAccountByPhoneResponse
  identity: AdminAccountSummary
  phone:
    phone_masked: string
    verification_status: string
    verification_source: string
```

其中 `identity` 不再是旧的 `CurrentIdentityResponse`。未找到返回 `404 ACCOUNT_NOT_FOUND`；响应、审计
和普通日志均不得记录输入的完整手机号。手机号搜索、列表和详情的权限规则一致，均为
`identity.authorization.manage` 或 `identity.account.manage`。

`available_actions` 只能从以下值中返回：`promote_doctor`、`edit_doctor`、`change_department`、
`revoke_doctor`、`disable_account`、`enable_account`。服务端按最新操作者权限和目标状态计算：

| 目标状态 | 可返回动作 |
|---|---|
| 正常普通账号 | 有授权管理权限时 `promote_doctor`；有账号管理权限时 `disable_account` |
| 正常医生账号 | 有授权管理权限时返回三个医生操作；有账号管理权限时 `disable_account` |
| 已禁用普通账号或医生账号 | 有账号管理权限时仅 `enable_account` |
| 任意超级管理员账号 | 空数组，首版只读 |

前端还会与当前 Principal permission 取交集，但写接口必须再次鉴权。完整手机号仅用于定位候选；所有
写操作只能使用稳定 `account_id`。

### 7.6 超级管理员医生和账号写接口

```http
POST   /api/v1/admin/identity/doctors/promote
PUT    /api/v1/admin/identity/doctors/:accountId
PUT    /api/v1/admin/identity/doctors/:accountId/department
DELETE /api/v1/admin/identity/doctors/:accountId
POST   /api/v1/admin/identity/accounts/:accountId/disable
POST   /api/v1/admin/identity/accounts/:accountId/enable
```

请求体固定为：

```text
POST /doctors/promote
  account_id: string
  department_id: string
  display_name: string
  staff_no?: string
  avatar_url?: string
  description?: string
  management_version: integer
  offline_verified: true
  operation_id: UUID

PUT /doctors/:accountId
  display_name?: string
  staff_no?: string
  avatar_url?: string
  description?: string
  management_version: integer
  operation_id: UUID

PUT /doctors/:accountId/department
  department_id: string
  management_version: integer
  operation_id: UUID

DELETE /doctors/:accountId
POST /accounts/:accountId/disable
POST /accounts/:accountId/enable
  management_version: integer
  operation_id: UUID
```

六个写接口成功后都直接返回最新 `AdminAccountDetail`，使前端可以用同一详情状态刷新页面。现有
`POST /doctors/promote` 返回 `CurrentIdentityResponse` 的契约在本模块实施时废止。调岗响应中的最新详情
只能表示目标科室；RPC/Outbox/审计还必须同时记录原科室和目标科室 ID，供服务端事件消费者准确失效
缓存。

开通医生仅允许正常普通账号；曾被撤销的档案执行重新激活而不是插入第二条。编辑、调岗和撤销只
允许当前医生；禁用账号保留医生角色、档案和科室，恢复后仍为医生；撤销医生身份后再恢复账号不会
自动重新开通医生。所有接口拒绝变更 `super_admin`，也拒绝操作者修改自己的账号或身份。

### 7.7 版本、幂等和错误响应

- 组织写使用 `version`，账号和医生写使用 `management_version`；缺失返回 `400`，不匹配返回
  `409 VERSION_CONFLICT` 并附最新版本；
- `operation_id` 全局唯一；同一操作者对同一目标重放同一动作时返回原结果，用于其他目标或动作时返回
  `409 OPERATION_ID_REUSED`；
- 所有写请求必须先校验最新权限、目标状态和版本，再在单个 Identity 事务中修改主数据、写审计和
  Outbox；
- 身份、科室或账号状态变化递增 `management_version` 和 `authorization_version`；仅编辑展示资料只
  递增 `management_version`；
- `401` 表示会话无效，`403` 表示权限不足，`404` 表示目标不存在，`409` 表示状态或并发冲突，
  `422` 表示字段语义不合法；错误响应必须包含稳定 `code` 和可展示的安全 `message`；
- 用户详情读取及所有身份、账号写操作记录操作者、目标账号、request ID、operation ID 和结果，不得
  记录完整手机号、Token、验证码或数据库内部错误。

这些接口供超级管理员专属“用户管理”二级页面使用。部门管理页展示医院、院区、科室和公共医生
目录，但不排列调岗、撤销或禁用按钮。

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

只导入医生资料并不等于测试医生能够登录。真实环境短信链路使用阿里云 PNVS，虚假手机号无法接收
验证码。本地联调可选择已经实现的固定码短信提供者：

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

组织写请求必须携带组织单元 `version`，账号和医生写请求必须携带 `management_version`，不能再用
`updated_at` 字符串代替乐观锁。版本冲突返回 `409`，要求前端重新读取后确认。所有失败响应和日志
继续使用脱敏手机号、`request_id`、`operation_id`、操作者 ID 和目标 ID，不能记录完整手机号或
Token。

## 11. 实施顺序

计划审核通过后，从最新 `main` 创建新的业务分支，按以下顺序实现：

1. 增加新的 Identity 迁移，演进组织单元、账号展示资料和医生档案；
2. 更新 permission 契约、角色初始化和授权测试；
3. 更新 Identity Proto 并生成 RPC 骨架；
4. 在 `common/authn` 增加可注入的授权版本校验抽象和 HTTP/gRPC 统一拦截器，在
   `common/authn/versionredis` 实现一份跨服务复用的 Redis Reader；各服务只完成配置与
   `ServiceContext` 注入，不得复制校验代码；
5. 重构 `authorization.Manager`：操作者使用已验证 Principal，保留目标账号业务查询和变更事务；
6. 实现 Identity Repository、事务、审计、Outbox、授权版本唯一写入以及 Redis 未命中的
   Identity/MySQL 回源，并补充旧 Token 拒绝、跨服务读取和缓存故障 fail-closed 测试；
7. 更新 `app-api` 的 `.api` 契约并生成 HTTP 骨架；
8. 实现本人昵称同步、公共科室和医生目录以及管理员组织、用户、医生和账号接口；
9. 实现 local seed 工具与环境保护；
10. 如联调需要，启用已经实现且严格受限的 local SMS Provider；
11. 补充单元测试、MySQL/Redis 集成测试和 API 权限测试；
12. 分别使用未登录访客、普通用户、医生和超级管理员验证公共目录；
13. 使用超级管理员完成一次组织维护、医生开通、调岗、撤销和普通用户回退的联调回归。

前端页面与调用契约已经确定；本阶段按第 7 节一次性落地 `.api`、Proto、RPC、数据库迁移、错误码和
mock 数据。后端不得仅实现路径占位或返回旧授权上下文后要求前端临时兼容。

## 12. 验收标准

- 可以建立“唯一医院—多个院区—院区直属科室”三层结构，子科室、第二个医院和非法层级被拒绝；
- 未登录访客、普通用户、医生和超级管理员都可通过同一公共接口获取医院、有效院区、选中院区的
  有效科室及选中科室的医生；
- 部门或医生数据库数据发生变化后，接口结果随之变化，不依赖前端硬编码清单；
- 超级管理员可以新增、修改、停用和恢复院区、科室，但不能通过小程序修改医院根节点；
- 有有效科室的院区、有有效医生的科室不能直接停用；
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

## 13. 已确定的实施决定

1. 使用统一组织单元表表达唯一医院、多个院区和院区直属科室；首期不允许子科室；
2. 小程序动态展示医院但不管理医院根节点；超级管理员管理院区和科室；
3. “删除院区/科室”统一采用停用语义，不提供生产物理删除或级联停用；
4. 撤销医生身份后保留普通用户身份和全部历史记录；
5. 首版工作人员资料为显示名称、可选工号、唯一所属科室、头像 URL 和擅长描述；
6. local SMS Provider 不是部门/用户页面接口的前置条件；其实现只允许在 `local/test` 环境按第 9.3 节显式启用。
