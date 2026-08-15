# Appointment 实施子计划

本目录把 [Appointment 预约检查服务总 Plan](../01-appointment-and-examination-booking.md) 拆成可以独立验收的
工程阶段。子 Plan 不能改变总 Plan 的业务边界；发现冲突时先更新总 Plan，再继续实现。

- [01 检查项目描述 CRUD 与服务骨架](./01-examination-item-description-crud.md)：Proto 优先建立 Appointment
  RPC，连接独立 MySQL，并完成检查项目及自然语言描述的基础 CRUD。
- [02 检查房间、项目关系与独立周配置](./02-room-project-weekly-configuration.md)：管理科室房间、
  房间可执行项目、房间开放窗口和项目预约窗口，并为热点患者查询建立缓存；本阶段不创建患者预约。
- [03 本周预约 CRUD 与容量占用](./03-current-week-booking-crud.md)：在既有项目、房间和双周配置之上完成
  患者本周候选查询、预约创建/查看/删除、工作人员开始检查、具体日期容量占用、每周预约额度、配置变更联动清理和超时未到场处理。
- [04 检查执行与报告版本](./04-examination-report-versioning.md)：补齐检查开始/完成状态、首版报告原子发布、
  正式报告不可覆盖的更正版本，以及工作人员和患者两套读取接口。
- [05 预约消息与提醒](./05-appointment-messages.md)：在现有预约、检查执行和报告事实之上生成患者本人消息与
  当前科室工作人员提醒；当前只记录已确认的消息类型、触发条件和科室范围，具体数据结构与接口待实施前评审。

未排期优化提案：

- [组织目录投影与预约资源树](./optimization-organization-projection-and-resource-tree.md)：把当前临时的前端跨服务
  聚合收回 App API/Appointment，通过 Identity 组织事件、必要的同步 RPC 和资源树接口支撑院区切换及两套
  三级视图；该提案不属于当前第一阶段交付。
