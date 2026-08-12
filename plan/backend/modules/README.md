# 后端模块设计

## 当前评审与实施入口

- [01 Appointment 预约检查服务](./01-appointment-and-examination-booking.md)：当前唯一进入人工评审的模块
  Plan；预约时间采用上午/下午大窗口模型（原时间方案 C），惩罚机制已选择方案 A“固定预约额度”。

评审通过后，先更新 HTTP、RPC、事件和迁移契约，再创建 Appointment Service。已实施归档和业务审计等
横切文档不会替代当前实施入口。

## 持续约束

- [02 业务审计](./02-business-audit.md)：所有业务服务持续遵守的跨模块约束；各领域随自身功能落地。

独立 Planning Service 已从当前路线删除：每个检查项目形成一条独立预约、执行和报告链路；Appointment
内部可以根据固态时间约束图、科室窗口和地图耗时计算建议顺序，但不会合并或自动修改预约。

## 已实施模块

- [implemented](./implemented/)：已经确认、实现并完成阶段验收的模块 Plan。

模块文档至少说明业务流程、状态机与约束、数据归属、权限、失败处理、本期边界和验收标准。Patient、
Navigation、Report、Message 等能力在真正进入开发前再建立独立 Plan，不把总体功能清单当成可执行设计。
