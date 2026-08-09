# 微信小程序规划包

本目录集中保存微信小程序的信息架构、交互外壳、页面规划和后续前后端联调边界，避免把设计讨论散落在源码目录或后端服务文档中。

## 当前决策

- 技术栈：uni-app + Vue 3 + TypeScript + Vite；
- 首版发布目标：微信小程序；
- 一级导航：首页、挂号、消息、我的；
- 已实现基线：可切换的页面外壳和旧版静态“我的”页面；
- 最新确认需求：入口页使用手机号验证码注册/登录，头像和昵称仅作为可选展示资料；
- 登录标识：阿里云 PNVS 验证后的手机号是唯一登录标识，内部 UUID 仍是数据库主键；
- 就诊人首版：一个账号只维护本人，已验证手机号 + 本人资料作为基础身份完成条件；
- 接口分层：小程序只调用 `service/app/api` 提供的 HTTP JSON 接口，`service/identity/rpc` 只用于后端内部 gRPC；
- 导航实现：优先使用标准 TabBar，不提前实现自定义导航；
- 接口原则：业务规则确认后，先修改 `contracts/api/` 中的契约，再实现前后端联调。

## 文档索引

1. [01-frontend-shell-design.md](./01-frontend-shell-design.md)：四个一级页面、标准 TabBar、工程目录、后端边界和第一阶段验收标准。
2. [03-entry-and-profile-requirements.md](./03-entry-and-profile-requirements.md)：历史微信资料入口需求；认证入口以计划 07 为准。
3. [04-login-and-identity-implementation.md](./04-login-and-identity-implementation.md)：历史微信登录调用链，保留用于理解兼容代码。
4. [05-self-patient-basic-identity.md](./05-self-patient-basic-identity.md)：首版仅本人、已验证手机号、数据归属和医疗信息访问边界。
5. [06-account-mode-switching.md](./06-account-mode-switching.md)：医生角色保持不变，通过本地会话在患者端和医生工作台之间切换。
6. [07-phone-primary-authentication.md](./07-phone-primary-authentication.md)：当前认证基线；使用阿里云 PNVS 短信认证，以手机号作为唯一登录标识。

后续页面和业务规划按顺序增加，例如：

```text
08-message-center.md
09-appointment-flow.md
```

只有当相应业务范围和交互已经确认时才创建文档，不预先生成空规划文件。
