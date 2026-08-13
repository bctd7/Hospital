# 患者检查项目展示与预约占位

> 状态：患者展示流程与独立 Mock Adapter 已实现；真实患者项目目录及预约接口待后端补充
>
> 适用身份：患者视图，以及工作人员主动切换到患者视图

## 1. 与管理端的差异

患者页面不是管理端列表去掉按钮后的版本。它只帮助患者理解检查项目并逐步选择：

```text
院区 -> 科室 -> 检查项目 -> 可选房间 -> 本周时间 -> 确认预约（后续）
```

患者不查看 disabled 资源、版本、更新时间、配置容量、operation ID 或管理状态，也不选择医生。

截图首页中的入口按应用版本区分：

| 应用版本 | 标题 | 副标题 | 行为 |
|---|---|---|---|
| 患者端 | 检查项目 | 查看项目说明与可检查地点 | 进入患者项目浏览 |
| 工作人员端 | 检查项目管理 | 维护项目、房间与开放时间 | 进入管理页面 |

首页卡片可以复用视觉组件，但标题、权限和目标路由必须由 `appVariant` 分开配置。

## 2. 近期页面范围

患者端近期可以完成：

- 院区和科室选择；
- 检查项目卡片、说明摘要和详情；
- 按项目展示可选择房间；
- 使用 Mock 展示本周日期和 morning/afternoon；
- 明确提示“演示数据，不会创建真实预约”。

暂不完成：

- 精确剩余容量；
- 预约创建、取消或改签；
- 预约成功页；
- 医生、诊室最优推荐或自动分配；
- 下周及更远日期选择。

## 3. 页面结构

```text
PatientExaminationBrowser
├── CampusSelector                 Identity 公共目录
├── DepartmentSelector             Identity 公共目录
├── ExaminationItemList            Patient Appointment Adapter
├── ExaminationItemDetail
├── AvailableRoomList               已知 item_id 后可接真实接口
└── MockCurrentWeekWindowList        后续替换为候选窗口接口
```

项目卡片展示名称和短说明，点击后先展示完整患者说明，再进入房间选择。房间只展示患者可理解的名称及院区/科室
上下文，不展示共享容量配置。

## 4. Mock 隔离规则

允许当前预约流程使用 Mock，但必须满足：

- 文件与实例明确命名为 `patientAppointmentMockAdapter`；
- 只在患者预约页面依赖注入，不进入管理端或通用 API Adapter；
- 页面顶部或确认区持续显示“Mock/演示”标识；
- 确认按钮只弹出演示说明，不生成本地预约记录，不显示真实成功态；
- Mock 项目、房间和窗口不写入 Pinia 持久化或 Storage；
- 接真实接口时保留相同 ViewModel，由 Adapter 完成字段转换。

现有 Mock 中“检查项目 -> 承接科室 -> 医生 -> 日期”的模型需要调整为：

```text
院区/科室 -> 检查项目 -> 房间 -> 本周 morning/afternoon
```

医生数组、医生选择步骤和“号源选项”文案全部移除。

## 5. Identity 和 Appointment 数据拼接

Identity 提供医院、院区和科室；Appointment 提供项目和房间。页面容器以稳定 ID 拼接：

```text
campus_id -> Identity 科室列表
department_id -> Patient Appointment 项目列表
item_id -> Appointment 可选房间
```

同名科室必须连同院区显示。项目描述来自 Appointment，不在 Identity 中维护；房间也不从 Identity 读取。

## 6. 真实接入前置条件

患者端从 Mock 切真实数据前需要：

1. 患者安全的按科室 active 项目列表接口；
2. 能组合房间开放与项目预约窗口的本周候选接口；
3. 创建预约接口在 MySQL 事务中复核关系、时间、停止新增时间和容量；
4. 稳定的满额、截止、配置变化和幂等错误码。

只有第 1 项完成时，可以先把项目列表和房间列表切真实，日期与确认仍保持明确的预约占位。

## 7. 验收标准

- 患者入口不再显示“检查项目管理”；
- 患者流程中没有医生选择；
- 管理端真实数据不会混入患者 Mock，患者 Mock 也不会写入后端；
- 患者只看到 active 项目和 active 房间；
- 项目、房间、日期和预约成功的真实程度在 UI 上表达清楚；
- 后续替换 Adapter 时不需要重写页面组件和选择状态机。
