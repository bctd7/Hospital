# Appointment 实施子计划

本目录把 [Appointment 预约检查服务总 Plan](../01-appointment-and-examination-booking.md) 拆成可以独立验收的
工程阶段。子 Plan 不能改变总 Plan 的业务边界；发现冲突时先更新总 Plan，再继续实现。

- [01 检查项目描述 CRUD 与服务骨架](./01-examination-item-description-crud.md)：Proto 优先建立 Appointment
  RPC，连接独立 MySQL，并完成检查项目及自然语言描述的基础 CRUD。
- [02 检查房间、项目关系与独立周配置](./02-room-project-weekly-configuration.md)：管理科室房间、
  房间可执行项目、房间开放窗口和项目预约窗口，并为热点患者查询建立缓存；本阶段不创建患者预约。
