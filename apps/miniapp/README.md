# 微信小程序

本目录是 Hospital 唯一客户端，技术栈为 uni-app、Vue 3、TypeScript 和 Vite。页面业务边界见
`plan/frontend/`，HTTP 路径和字段以 `contracts/api/` 为准。

## 当前实现

- 首页、医生名录/人员管理、消息、我的四个 Tab；
- 手机号验证码登录、Access/Refresh Token、单任务刷新和退出；
- 患者端与工作人员端切换；
- 院区、科室、医生目录和超级管理员用户管理；
- 工作人员检查项目、房间、可执行项目及两套周配置管理；
- 患者本周预约、本人预约和取消；
- 工作人员科室预约、检查开始、报告草稿、发布和更正；
- 患者检验报告、工作人员检查记录；
- 患者和科室消息、未读徽标与逐账号已读状态。

Appointment 业务使用真实 HTTP。首页只是静态入口编排，不伪造预约、报告或消息数据；尚未实现的缴费、
医保、票据和就诊人业务只显示明确占位状态。

## 源码结构

```text
apps/miniapp/src/
├─ api/          HTTP Client 与按领域拆分的 Adapter
├─ services/     页面间业务编排、短缓存和失效
├─ components/   可复用组件
├─ pages/        页面与分包页面
├─ stores/       会话和应用版本状态
├─ types/        前端领域类型
└─ utils/        无状态规则与展示转换
```

页面不直接调用 `uni.request`。Token、错误转换、刷新与退出集中维护；服务层不弹窗、不跳转，页面层不按名称
反查业务 ID。

## 本地开发

```powershell
Set-Location apps/miniapp
npm install
npm run dev:mp-weixin
```

微信开发者工具导入：

```text
apps/miniapp/dist/dev/mp-weixin
```

开发工具模拟器可以访问电脑的 `127.0.0.1`；真机需要在 `.env.local` 配置电脑局域网地址。环境文件只保存
公开连接信息，不能放 AccessKey、JWT 私钥或手机号 HMAC Key。

## 构建与上传

```powershell
npm run test
npm run type-check
npm run build:all:mp-weixin
```

- 开发产物：`dist/dev/mp-weixin`；
- 发布产物：`dist/build/mp-weixin`；
- 正式构建固定读取 `release.config.json` 并校验 CloudBase AnyService 配置；
- 不要从项目根目录或 `dist/dev` 上传体验版。

终端上传：

```powershell
$miniappRoot = 'C:\Users\27902\GolandProjects\Hospital\apps\miniapp'
$wechatCli = 'C:\Program Files (x86)\Tencent\微信web开发者工具\cli.bat'
$releaseVersion = '0.3.1'
$releaseDescription = '完善检查预约、检查记录与检验报告功能，优化整体使用体验。'

Set-Location $miniappRoot
npm run test
npm run type-check
npm run build:mp-weixin

& $wechatCli upload `
  --project (Join-Path $miniappRoot 'dist\build\mp-weixin') `
  --version $releaseVersion `
  --desc $releaseDescription `
  --lang zh
```

终端输出 `√ upload` 才表示上传成功。若开发者工具正打开 `dist/build/mp-weixin`，先关闭该项目再构建，避免
重建目录时被误报为项目文件夹已删除。

## 登录联调

所有构建都必须通过手机号验证码取得真实 Hospital Token，再调用 `GET /api/v1/auth/me` 恢复角色、权限和
稳定 `department_id`。工作人员切换到患者端仍使用同一账号，只改变页面和业务入口。

```text
POST /api/v1/auth/phone/code
POST /api/v1/auth/phone/login
GET  /api/v1/auth/me
```

## 安全约束

- 不记录手机号、验证码、Token 和报告正文；
- 不在 Storage 长期保存不必要的医疗数据；
- 菜单隐藏不是权限校验；
- 不把开发服务直接暴露到公网；
- 不在未评估时强制升级可能破坏 DCloud 兼容性的依赖。
