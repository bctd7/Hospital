# 后端模块设计

- [01-identity-and-access-control.md](./01-identity-and-access-control.md)：账号、认证、角色、权限、部门和工作人员开通。
- [02-examination-planning.md](./02-examination-planning.md)：检查顺序规则、计算、版本和解释。
- [03-business-audit.md](./03-business-audit.md)：关键业务操作、授权变化和敏感访问审计。
- [04-organization-staff-and-seed-data.md](./04-organization-staff-and-seed-data.md)：单医院、院区、科室、医生资料、管理员用户管理、公共目录和本地 mock 数据。

模块文档至少说明业务流程、状态机/约束、数据归属、接口或事件、具体技术方案（包括适用算法）和
验收标准。Appointment、Patient、Navigation、Report、Message 等模块在规则进入开发前再建立
独立文档，不把总体功能清单误当成模块实现设计。
