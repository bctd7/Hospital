# 本人就诊信息与基础身份方案

> 文档状态：首版折中方案已确认
>
> 适用范围：小程序“就诊人管理”首版、本人资料和业务访问边界
>
> 优先级：本文件对首版就诊人范围的约定高于旧文档中的家庭成员描述

## 1. 首版决策

首版只管理当前账号本人，不实现父母、子女或其他家庭成员：

```text
一个微信 OpenID
  -> 一个 Hospital account
  -> 一个本人 patient profile
  -> 一个自报手机号
```

用户完成微信登录，并登记唯一、格式合法的手机号和必要的本人资料后，产品状态记为
`basic_verified`（基础身份完成），可进入首版本人业务。

这是个人主体小程序在没有微信手机号快捷验证、短信供应商和医院患者主索引接口时采用的折中
判断，不是法律或医疗意义上的实名认证。页面文案使用“基础认证完成”或“本人资料已完成”，
不使用“实名认证成功”。

## 2. 两个认证步骤

### 2.1 第一步：微信账号认证

```text
uni.login
  -> 微信 code
  -> Hospital 后端 code2Session
  -> OpenID
  -> Identity account
  -> Hospital Access Token + Refresh Token
```

这一步证明后端后续识别到的是同一个微信账号，但不证明用户真实姓名或手机号归属。

### 2.2 第二步：本人基础资料

登录成功后，用户在“就诊人管理”中填写：

- 本人姓名；
- 中国大陆手机号；
- 预约业务实际需要的其他最少资料。

手机号继续使用 Identity 已实现的 `PUT /api/v1/auth/me/phone` 保存：数据库只保存手机号 HMAC
指纹、脱敏值和 `self_reported` 来源。同一手机号指纹只能绑定一个账号。

当以下条件同时满足时，前端业务状态可显示为 `basic_verified`：

```text
Hospital Session 有效
AND account_type = patient
AND 本人 patient profile 已完成
AND Identity 中存在唯一的 self_reported 手机号绑定
```

该状态只表示首版产品允许继续办理本人业务，不得把 Identity 表里的手机号状态改成
`verified`，也不能声称运营商、微信或医院已经确认手机号归属。

## 3. 解决“填别人手机号就看到别人资料”的规则

手机号只作为当前账号的联系方式和基础资料，不作为医疗数据检索凭证。严格禁止以下查询：

```text
手机号 -> 查找医院患者 -> 自动返回预约、病历或报告
姓名 + 手机号 -> 模糊匹配患者 -> 返回候选人资料
```

首版所有业务数据从创建时就绑定内部 ID：

```text
account_id
  -> patient_profile_id
      -> appointment_id
          -> examination_id
              -> report_id
```

查看数据时必须沿已经建立的内部关系校验：

```text
当前 Token.account_id
  -> 当前账号唯一 patient_profile_id
  -> 目标预约或报告的 patient_profile_id
  -> 完全相同才允许访问
```

因此即使用户误填或恶意填写了别人的手机号，他也只能看到自己这个账号以后创建的业务数据，
不会看到医院里另一个真实患者的历史资料。

## 4. 首版功能边界

| 功能 | 仅微信登录 | 基础身份完成 |
|---|---:|---:|
| 浏览公告、科室、检查项目等公开信息 | 是 | 是 |
| 填写和修改本人资料 | 是 | 是 |
| 创建本人预约 | 否 | 是 |
| 查看当前账号创建的本人预约 | 否 | 是 |
| 查看由这些预约产生并绑定到本人档案的报告 | 否 | 是 |
| 按手机号查询医院既有历史预约或报告 | 否 | 否 |
| 自动绑定医院既有患者档案 | 否 | 否 |
| 添加或切换家庭成员 | 否 | 否 |

“检查报告查询”首版只开放本系统业务链路内产生、且已经通过 `patient_profile_id` 建立归属的
报告。医院历史报告接入必须等到以后具备患者主索引匹配、线下核验或其他更强身份能力。

## 5. 页面调整

现有 `pages/profile/patients/index.vue` 仍然展示“默认就诊人”和“家庭就诊人”，实施时调整为
单一本人资料页：

```text
就诊人管理
├── 本人资料状态
│   ├── 未完成
│   └── 基础认证完成
├── 姓名
├── 脱敏手机号
├── 编辑本人资料
└── 安全说明
```

需要删除：

- “默认就诊人”；
- “家庭就诊人”；
- “后续支持管理本人及家庭成员”；
- 家庭成员数量和切换入口。

页面必须明确说明：基础认证用于当前小程序本人业务，不代表实名核验，不能用于自动调取医院
既有病历。

## 6. 前端状态

登录状态与本人资料状态分开：

```text
session.status = idle / authenticating / authenticated / guest
patient.status = missing / basic_verified
```

不能因为 Identity 自动创建了 `account_type=patient` 就认为本人资料已经完成。

前端调用关系：

```text
完成微信登录
  -> GET 本人 patient profile
  -> missing：进入本人资料填写
  -> 提交手机号到 PUT /auth/me/phone
  -> 提交其他本人资料到 Patient API
  -> 后端确认两部分都存在
  -> patient.status = basic_verified
```

## 7. 后端影响

现有 Identity 登录部分不需要重做：

- OpenID 登录和自动创建账号已经实现；
- Access Token、Refresh Token 和 Redis Session 已经实现；
- `PUT /api/v1/auth/me/phone` 和手机号唯一指纹已经实现；
- 手机号继续保持 `self_reported`，不修改为强验证状态。

但当前后端没有保存“本人姓名和 patient profile”的数据结构，也没有本人资料查询接口。为了让
预约和报告拥有稳定的 `patient_profile_id`，仍需在后续业务开发中增加一个很小的 Patient 模块，
不需要拆成微服务：

```text
patient_profiles
├── id
├── account_id UNIQUE
├── name
├── status               # 首期 missing / basic_verified
├── created_at
└── updated_at

GET /api/v1/patients/me
PUT /api/v1/patients/me
```

Patient 模块保存本人业务资料，Identity 继续保存登录账号和手机号绑定。预约和报告只引用
`patient_profile_id`，不能通过手机号反向查找并授权。

如果首期确认除手机号外完全不需要姓名等资料，也可以暂时直接以 `account_id` 作为本人业务
主体，但在预约模块建立前必须固定这一决定；不能一部分表使用 `account_id`，另一部分又临时
创建 patient ID。

## 8. 实现顺序

1. 完成 `04-login-and-identity-implementation.md` 中的微信登录和 Session；
2. 小程序接入现有手机号登记接口；
3. 建立首期一对一本人 Patient 数据结构和 API；
4. 将就诊人页面从家庭列表改为本人资料页；
5. 由后端统一计算并返回 `basic_verified`，前端不能自行判定成功；
6. 预约创建时绑定 `account_id + patient_profile_id`；
7. 报告只能通过预约/检查链路关联到相同 patient profile；
8. 增加越权、错手机号和跨账号访问测试。

## 9. 验收标准

- 一个账号最多一个本人 patient profile；
- 一个手机号指纹最多绑定一个账号；
- 页面不再出现家庭成员管理；
- OpenID 和手机号两步都完成后才显示“基础认证完成”；
- 数据库中的手机号仍明确为 `self_reported`；
- 任何接口都不能凭手机号返回医院既有患者或报告；
- 账号只能查看绑定到自己 patient profile 的预约、检查和报告；
- 修改手机号后基础身份状态按业务规则重新计算；
- 日志不记录微信 code、OpenID、完整手机号或 Token；
- 后续接入医院历史数据前必须另行设计更强身份匹配方案。

## 10. 后续升级

以后接入短信验证码、微信手机号、医院患者主索引、线下审核或合规实名认证后，可以增加更高
等级：

```text
basic_verified       # OpenID + 自报手机号 + 本人资料
phone_verified       # 手机号控制权已验证
hospital_verified    # 已与医院患者主索引可靠绑定
```

家庭成员功能只能建立在独立授权关系上，不能让用户输入一个手机号或身份证号后直接读取对方
数据。
