# Appointment 阶段 4：检查执行与报告版本

> 状态：已实现并通过临时 MySQL 完整链路验收（2026-08-14）。
>
> 前置阶段：[本周预约 CRUD 与容量占用](./03-current-week-booking-crud.md)。

## 1. 范围

本阶段在 Appointment 内完成一次预约的一次检查和一份报告，不建立独立 Report 服务，也不引入医生预约资源关系。

```text
confirmed -> in_progress -> completed
         \-> no_show
                                  └-> 完成检查 + 发布首版报告 + 释放房间容量（同一事务）
```

工作人员能力位于 `manager/staff`，患者读取位于 `manager/patient`。报告领域数据和事务端口放在
`manager/common`，MySQL 实现位于 `repository/mysqlstore`。

## 2. 已确认规则

- 一次预约对应一次检查和唯一一份报告主体；
- 本科室授权医生或管理员可以开始检查、保存草稿、完成并发布；
- `started_by` 作为可获得的实际执行人员，`completed_by`、`authored_by`、`published_by` 分别记录实际动作；
- 草稿允许原地编辑，发布时“检查所见”和“检查结论”必填；
- 正式报告绝不原地更新；更正必须填写原因并新增版本；
- 历史正式版本保留为 `superseded`，患者只读取当前 `published` 版本；
- 预约创建时保存患者展示名和脱敏手机号快照；报告发布时保存项目、患者、科室、院区、实际操作人员和房间严格地址快照，目录或地址后续变化不改写历史报告；
- 检查完成、报告首版发布和容量释放必须全部成功或全部回滚；
- 患者只能读取本人预约的正式报告，不能读取草稿或其他患者报告。

## 3. 数据结构

- `appointment_examination_reports`：一次预约唯一的报告主体、检查元数据快照、当前版本指针；
- `appointment_examination_report_versions`：草稿、当前正式版和被更正替代的历史正式版；
- `appointment_examination_items`：保存项目级四字段报告模板及独立模板版本；
- `appointment_bookings`：增加 `in_progress/completed`、开始人与完成人及时间；
- `appointment_booking_operations`：保存开始检查、草稿保存、完成发布和更正的幂等结果。

固定正文包含：

- `objective_findings`：检查所见；
- `impression`：检查结论；
- `recommendation`：处理建议，可选；
- `notes`：备注，可选。

## 4. 接口

工作人员：

- `GET/PUT /admin/appointment/examination-items/:itemId/report-template`
- `POST /admin/appointment/bookings/:bookingId/start-examination`
- `PUT /admin/appointment/bookings/:bookingId/report/draft`
- `POST /admin/appointment/bookings/:bookingId/report/complete-and-publish`
- `GET /admin/appointment/bookings/:bookingId/report`
- `POST /admin/appointment/reports/:reportId/corrections`
- `GET /admin/appointment/reports/:reportId/versions`

患者：

- `GET /appointment/reports`
- `GET /appointment/bookings/:bookingId/report`

## 5. 暂不包含

- 报告附件、影像和 OSS；
- PDF 生成、下载和批量导出；
- 多人审核、电子签章和撤回审批；
- 报告发布通知。通知由后续消息阶段消费已提交的业务事实，不进入本阶段核心事务。
