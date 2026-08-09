# 患者端与医生工作台模式切换

> 文档状态：基础切换能力进入实现
>
> 适用范围：同时具备医生角色和本人患者需求的账号

## 1. 核心决策

不在数据库账号表增加 `current_role`、`current_identity` 或 `active_mode` 字段。数据库保存的是
长期身份事实，小程序保存的是当前设备上的界面偏好：

```text
Identity 数据库
  -> account_type = staff
  -> role = department_doctor
  -> department_id = 当前科室

小程序本地会话
  -> active_mode = patient / doctor
```

医生切换到患者端不会删除医生角色，也不会把 `account_type` 改回 `patient`。切回医生工作台时
不需要管理员重新授权。

## 2. 为什么不保存数据库字段

- 当前模式不是权限事实，不应触发授权版本和 Token 重签；
- 同一个医生可能同时在工作电脑使用医生端、在手机使用患者端，数据库单值会互相覆盖；
- 页面偏好发生异常时应直接回退患者端，不能影响账号角色；
- 如果患者篡改本地 Storage 为 `doctor`，后端仍会根据真实 Token 角色拒绝医生接口。

若以后确实需要跨设备同步“默认进入哪个端”，可以新增账号偏好或设备会话偏好，但它仍然不能
参与后端授权判断。

## 3. 可用模式计算

```text
普通患者账号
  -> available_modes = [patient]

包含 department_doctor 角色的账号
  -> available_modes = [patient, doctor]
```

`super_admin` 是否同时显示医生工作台取决于其是否还具有医生角色，不因为是超级管理员自动视为
医生。后续支持多角色时仍按明确角色计算。

登录、恢复 Session 或刷新 Token 后都要重新校验模式。如果医生角色已被撤销，当前模式立即回退
`patient`。

## 4. 前端行为

首期继续复用同一组 TabBar 页面路径，只根据模式切换文案和页面内容骨架：

| 页面路径 | 患者模式 | 医生模式 |
|---|---|---|
| `pages/home/index` | 首页 | 工作台 |
| `pages/registration/index` | 挂号 | 预约管理 |
| `pages/messages/index` | 消息 | 消息 |
| `pages/profile/index` | 我的 | 我的 |

在“我的”页面中，只有医生角色可见模式切换按钮：

```text
患者模式 -> 切换到医生工作台
医生模式 -> 切换到患者端
```

医生业务尚未实现时，医生工作台只显示真实的建设中状态，不伪造排班、患者或预约数据。

## 5. 会话状态

`active_mode` 与 TokenPair、Principal 一起保存在现有 `hospital:session` 本地记录中：

```text
status
principal
tokens
active_mode
```

切换模式不请求后端、不刷新 Token。退出登录时清除模式；下一次全新登录默认进入患者端。Storage
中缺少模式或模式值非法时也默认患者端。

## 6. 后端边界

后端不信任 `active_mode`：

- 医生业务接口继续校验 `department_doctor` 和具体 permission；
- 患者业务接口按当前 `account_id -> patient_profile_id` 校验本人数据；
- `account_type=staff` 不应阻止医生建立自己的本人 patient profile；
- 当前模式不能通过请求参数扩大权限，也不写入 Access Token；
- 操作审计记录真实 `account_id`、角色、科室和业务动作，不把界面模式当成身份依据。

## 7. 后续扩展

医生和患者页面差异扩大后，可以把页面内容拆成独立组件或医生页面树，但仍复用同一个登录会话
和角色事实。只有标准 TabBar 无法满足两套导航结构时，才评估自定义 TabBar 或医生工作台独立
页面，不因为模式切换新增 Identity 数据库字段。

## 8. 验收标准

- 普通患者看不到医生模式入口；
- 医生可以在患者端和医生工作台之间来回切换；
- 切换后刷新或重开小程序仍保留当前设备模式；
- 医生角色被撤销后自动回退患者模式；
- 修改本地模式不能让普通患者调用医生接口；
- 切换模式不修改账号类型、角色、科室和授权版本；
- 医生在患者模式下可以使用自己的本人档案，不会操作其他患者资料。
