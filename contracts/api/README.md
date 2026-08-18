# HTTP 契约

`app.api` 是 go-zero 对外 HTTP 服务入口，只负责导入已经按产品能力拆开的契约文件。

```text
identity-auth.api                手机号登录
identity-session.api             Access/Refresh 会话
identity-profile.api             当前账号本人展示资料
identity-doctor-directory.api    公开科室医生目录
identity-account-admin.api       管理员账号与医生身份维护
identity-organization.api        公开组织目录与组织维护
appointment-catalog.api          检查项目和报告模板
appointment-resources.api        房间、项目关联与周窗口
appointment-bookings.api         预约选项、预约与现场检查流程
appointment-messages.api         患者本人消息与科室消息
appointment-reports.api          报告草稿、发布、更正和只读查询
guidance-rules.api               项目直接先后规则
guidance-configuration.api       三步项目完整配置与说明解析
guidance-planning.api            智能预约与当日检查顺序
guidance-routing.api             地点检索与两点路线
```

文件按调用者看见的业务能力拆分，不按 Handler、数据库表或 Manager 拆分。某项 HTTP 能力即使组合多个内部 RPC，
也只在最贴近用户操作的一个文件中定义。

```powershell
goctl api validate -api contracts/api/app.api
```
