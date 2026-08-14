# 前端规划

前端 Plan 只维护“接口消费方案”和“页面功能设计”。消费方案描述会话、缓存、错误处理和页面刷新，
页面文档描述路由、显示内容与交互；两者都不是接口事实源，不复制完整请求、响应字段或后端表结构。

可调用路径和字段以 `contracts/api/` 为准，阅读版使用
[`docs/api/openapi.json`](../../docs/api/openapi.json) 或 Swagger UI。

## 接口消费方案

- [认证、会话与身份接口](./api/01-authentication-and-session.md)
- [部门目录与医生管理接口](./api/02-department-directory-and-management.md)
- [超级管理员用户接口](./api/03-admin-user-management.md)
- [Appointment 检查目录、房间与周配置接口](./api/04-appointment-resources.md)
- [检查执行与报告接口](./api/05-examination-reports.md)

## 页面设计

- [应用外壳与身份版本](./pages/01-app-shell-and-variants.md)
- [登录入口与个人中心](./pages/02-entry-and-profile.md)
- [本人就诊信息](./pages/03-self-patient.md)
- [医生名录与人员管理](./pages/04-department-management.md)
- [超级管理员用户管理](./pages/05-admin-user-management.md)
- [工作人员检查项目与预约资源管理](./pages/06-admin-appointment-resource-management.md)
- [患者检查项目展示与预约占位](./pages/07-patient-examination-item-browser.md)
- [检查执行与报告页面](./pages/08-examination-reports.md)

## 当前基线

- 唯一客户端形态：微信小程序；不规划 Web/H5 管理后台或另一套原生 App；
- 工程：`apps/miniapp`，uni-app + Vue 3 + TypeScript + Vite；
- 一级页面：首页、医生名录（工作人员为人员管理）、消息、我的，使用标准 TabBar；
- 主认证：手机号 + 阿里云 PNVS 短信验证码；
- 会话：Hospital Access Token + Refresh Token，前端统一刷新和撤销；
- 授权或身份变化使旧会话失效时，统一清理状态、重置到登录页并允许立即重新登录；
- 应用版本：患者端与工作人员端复用四个 Tab 页面，工作人员可主动切回患者端；
- 工作人员：医生与超级管理员共享页面框架，具体入口按 permissions 区分；
- 第二页：患者端名称为“医生名录”，工作人员端名称为“人员管理”；只承载科室医生目录与用户管理入口；
- 组织范围：医院根节点只读；患者和医生切换有效院区；超级管理员管理院区和科室；首期无子科室；
- 超级管理员：从人员管理进入独立用户管理页，底栏仍保持四项；
- 管理员选择科室时按“园区 / 科室”展示，允许不同园区存在同名科室，写操作只使用稳定科室 ID；
- Appointment 管理端检查项目、房间、项目关系和独立周配置，以及患者端本周预约均已接入真实 API；
- 患者端项目浏览和预约与管理端分开实现，不再使用 Appointment Mock；
- 检查执行与报告后端主链路已经完成，前端页面待报告模板、患者搜索和可读名称契约评审后实施。

新增前端需求时，接口契约先进入 `contracts/`；`plan/frontend/api/` 只补客户端消费策略，页面交互进入
`pages/`。不再创建混合接口字段、后端实现和页面截图的超长文档。
