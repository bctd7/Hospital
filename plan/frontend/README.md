# 前端规划

前端 Plan 只维护“接口消费方案”和“页面功能设计”。消费方案描述会话、缓存、错误处理和页面刷新，
页面文档描述路由、显示内容与交互；两者都不是接口事实源，不复制完整请求、响应字段或后端表结构。

可调用路径和字段以 `contracts/api/` 为准，阅读版使用
[`docs/api/openapi.json`](../../docs/api/openapi.json) 或 Swagger UI。

## 接口消费方案

- [认证、会话与身份接口](./api/01-authentication-and-session.md)
- [部门目录与医生管理接口](./api/02-department-directory-and-management.md)
- [超级管理员用户接口](./api/03-admin-user-management.md)

## 页面设计

- [应用外壳与身份版本](./pages/01-app-shell-and-variants.md)
- [登录入口与个人中心](./pages/02-entry-and-profile.md)
- [本人就诊信息](./pages/03-self-patient.md)
- [工作人员部门管理](./pages/04-department-management.md)
- [超级管理员用户管理](./pages/05-admin-user-management.md)

## 当前基线

- 唯一客户端形态：微信小程序；不规划 Web/H5 管理后台或另一套原生 App；
- 工程：`apps/miniapp`，uni-app + Vue 3 + TypeScript + Vite；
- 一级页面：首页、挂号、消息、我的，使用标准 TabBar；
- 主认证：手机号 + 阿里云 PNVS 短信验证码；
- 会话：Hospital Access Token + Refresh Token，前端统一刷新和撤销；
- 应用版本：患者端与工作人员端复用四个 Tab 页面，工作人员可主动切回患者端；
- 工作人员：医生与超级管理员共享页面框架，具体入口按 permissions 区分；
- 工作人员第二页：名称为“部门管理”，动态展示唯一医院、多个院区、院区直属科室和医生；
- 组织范围：医院根节点只读；患者和医生切换有效院区；超级管理员管理院区和科室；首期无子科室；
- 超级管理员：从部门管理进入独立用户管理页，底栏仍保持四项；
- 预约、消息等未接入业务保持真实空状态；部门和用户管理已经统一使用真实 HTTP Adapter，运行时 Mock、
  Mock 缓存和演示账号数据已经删除。

新增前端需求时，接口契约先进入 `contracts/`；`plan/frontend/api/` 只补客户端消费策略，页面交互进入
`pages/`。不再创建混合接口字段、后端实现和页面截图的超长文档。
