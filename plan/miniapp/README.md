# 微信小程序规划包

本目录集中保存微信小程序的信息架构、交互外壳、页面规划和后续前后端联调边界，避免把设计讨论散落在源码目录或后端服务文档中。

## 当前决策

- 技术栈：uni-app + Vue 3 + TypeScript + Vite；
- 首版发布目标：微信小程序；
- 一级导航：首页、挂号、消息、我的；
- 第一阶段：只建立可切换的空白页面外壳，不依赖后端业务接口；
- 导航实现：优先使用标准 TabBar，不提前实现自定义导航；
- 接口原则：业务规则确认后，先修改 `contracts/api/` 中的契约，再实现前后端联调。

## 文档索引

1. [01-frontend-shell-design.md](./01-frontend-shell-design.md)：四个一级页面、标准 TabBar、工程目录、后端边界和第一阶段验收标准。

后续页面和业务规划按顺序增加，例如：

```text
02-login-and-session.md
03-registration-flow.md
04-message-center.md
05-profile-and-patients.md
```

只有当相应业务范围和交互已经确认时才创建文档，不预先生成空规划文件。
