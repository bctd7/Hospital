# Appointment 阶段 5：预约消息与提醒

> 状态：已实施。消息由预约和报告事实动态生成，只持久化账号独立的已读状态。
>
> 前置阶段：[检查执行与报告版本](./04-examination-report-versioning.md)。

## 1. 范围

本阶段为现有预约、检查执行和报告事实生成患者与工作人员消息。消息不改变预约状态，也不引入医生预约资源关系。

## 2. 患者消息

患者只接收本人预约产生的以下四类消息：

1. **预约成功**：预约创建成功后生成，包含检查项目、科室、院区、严格房间地址、预约日期、时间窗口和最晚到院时间；
2. **到院提醒**：以最晚到院时间为基准，提前 1 小时和提前 30 分钟各提醒一次；生成提醒时预约必须仍为 `confirmed`，已经进入检查中、已经取消或已经成为未到场时不再提醒；
3. **报告发布**：检查完成并发布正式报告后生成，包含检查项目、检查日期和执行科室；
4. **报告更正**：正式报告产生更正版本后生成，包含报告、更正版本和更正时间信息。

患者从预约成功和到院提醒进入本人预约详情，从报告发布和报告更正进入本人报告详情。

如果预约创建时已经进入最晚到院时间前 1 小时范围，立即生成不足 1 小时提醒；预约在提前 30 分钟节点仍为
`confirmed` 时，继续生成提前 30 分钟提醒。

## 3. 工作人员消息

医生和管理员接收当前科室范围内的以下三类消息：

1. **新预约**：患者预约成功后生成，包含患者展示信息、检查项目、日期、时间窗口和严格房间地址；
2. **预约取消或未到场**：患者取消预约，或超过最晚到院时间仍未开始检查时生成；
3. **检查中但报告未完成**：预约在检查窗口结束时仍为 `in_progress` 时提醒一次，窗口结束 1 小时后仍为
   `in_progress` 时再次提醒；完成并发布报告后不再提醒。

工作人员消息属于科室，不绑定具体医生。工作人员从消息进入现有科室预约详情或报告填写页面。

## 4. 科室范围

- 医生固定使用登录身份中的所属科室，不允许切换到其他科室；
- 管理员按当前选择的科室查看消息，不默认展示全院全部消息；
- 管理员优先沿用工作人员管理页面当前选择的科室，其次使用上一次选择的科室；
- 管理员从未选择过科室时，第一次进入消息页必须先选择，不自动选择列表中的第一个科室；
- 管理员选择科室时按“院区 / 科室”识别范围，业务查询使用稳定 `department_id`。

## 5. 预约取消与消息来源

取消预约不再物理删除 `appointment_bookings`，而是把 `status` 更新为 `canceled`。取消仍然释放房间容量和患者时段，
且不归还患者当周预约额度；取消后的预约不进入患者未完成预约列表。现有 `appointment_booking_operations` 继续记录
取消操作者、原因和幂等结果。

消息不建立独立事实表，按稳定规则从现有数据动态生成：

- `appointment_bookings`：预约成功、到院提醒、取消、未到场和报告待完成；
- `appointment_examination_reports`：报告发布；
- `appointment_examination_report_versions`：报告更正；
- `appointment_booking_operations`：取消操作信息。

每条动态消息生成稳定 `message_key`，例如：

```text
booking:{booking_id}:patient:created
booking:{booking_id}:patient:arrival-60m
booking:{booking_id}:patient:arrival-30m
booking:{booking_id}:department:created
booking:{booking_id}:department:canceled
booking:{booking_id}:department:no-show
booking:{booking_id}:department:report-due
booking:{booking_id}:department:report-overdue
report:{report_id}:patient:published
report-version:{version_id}:patient:corrected
```

到院和报告待完成提醒根据预约时间、`started_at`、`completed_at` 和最终状态判断在对应时点是否成立；已经开始检查、
已经取消或已经未到场的预约不产生后续到院提醒，已经完成的预约不产生后续报告待完成提醒。

## 6. 已读状态

只新增一张个人阅读状态表，不新增 `appointment_messages`：

```text
appointment_message_reads
├── account_id
├── message_key
└── read_at
```

`(account_id, message_key)` 为联合主键。同一条科室消息由不同工作人员分别记录已读状态；读取其他账号的消息不会
改变当前账号的未读数量。写入已读前必须验证动态消息真实存在并属于当前患者或当前工作人员可访问的科室。

## 7. 接口

患者端：

```text
GET /api/v1/appointment/messages?page=&page_size=
PUT /api/v1/appointment/messages/read
```

工作人员端：

```text
GET /api/v1/admin/appointment/messages?department_id=&page=&page_size=
PUT /api/v1/admin/appointment/messages/read
```

已读请求只提交 `message_key`。列表项复用现有预约摘要，并增加消息自身字段：

```text
message_key
message_type
occurred_at
read_at
booking
report_id
report_version_id
report_version_no
```

`booking` 使用现有预约响应结构；报告消息只返回跳转和版本标识，不在消息列表返回报告正文。列表响应保留分页字段，
同时返回当前账号的 `unread_count`；工作人员响应另外返回按 `department_id` 汇总的科室未读数量，支持管理员科室
选择和底部消息角标。

## 8. 尚未确认

以下内容不在本 Plan 中预设，实施前继续讨论：

- 消息页面的具体视觉样式；
- 消息卡片的具体排版和动效；
- 微信订阅消息等站外推送渠道。

## 9. 验收标准

- 患者只收到本人预约对应的四类消息；
- 提前 1 小时和提前 30 分钟提醒均以最晚到院时间为基准，并在生成时重新检查预约状态；
- 工作人员只收到已确认的三类科室消息；
- 已经完成并发布报告的检查不再产生报告未完成提醒；
- 医生不能切换科室，管理员必须在明确的当前科室范围内查看消息；
- 取消预约保留为 `canceled`，释放容量但不归还当周预约额度；
- 消息从现有预约和报告数据动态生成，只新增个人阅读状态表；
- 同一科室消息的已读状态按工作人员账号分别记录；
- 不新增未经确认的消息类型、医生资源关系或页面功能。
