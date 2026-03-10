# TradeMate 核心数据模型与 API 契约

**文档版本**：V1.0  
**对应文档**：`docs/active/v1-agent-platform/TradeMate Agent平台技术方案_V1.0.md`  
**适用范围**：平台共用底座 + V1 广告 Agent  
**目的**：提供字段级数据模型、接口契约和事件约定

## 1. 设计原则

1. 平台对象与 Agent 业务对象分层建模。
2. 建议、审批、任务、复盘使用显式状态和不可变事件补充历史。
3. 所有核心对象必须带 `agent_type` 或明确归属。
4. API 优先 REST，状态变更补充 WebSocket 事件。

## 2. 枚举定义

### 2.1 通用枚举

|枚举|取值|
|---|---|
|agent_type|`ad_agent` `profit_agent` `pricing_agent` `inventory_agent` `compliance_agent`|
|risk_level|`low` `medium` `high`|
|status|`active` `paused` `disabled`|
|site_code|`US` `CA` `UK` `DE` `FR` `IT` `ES` `JP`|

### 2.2 建议状态

`draft` `ready` `pending_approval` `approved` `rejected` `expired` `executed`

### 2.3 审批状态

`pending` `approved` `rejected` `cancelled`

### 2.4 任务状态

`draft` `pending_approval` `approved` `queued` `running` `succeeded` `failed` `cancelled`

### 2.5 复盘状态

`pending` `partial` `ready`

## 3. 核心数据模型

### 3.1 user

|字段|类型|必填|说明|
|---|---|---|---|
|id|string|是|用户 ID|
|email|string|否|邮箱|
|phone|string|否|手机号|
|display_name|string|是|展示名|
|status|string|是|`active` / `disabled`|
|created_at|datetime|是|创建时间|
|updated_at|datetime|是|更新时间|

### 3.2 role_assignment

|字段|类型|必填|说明|
|---|---|---|---|
|id|string|是|记录 ID|
|user_id|string|是|用户 ID|
|store_id|string|是|店铺 ID|
|role_code|string|是|`owner` `operator` `approver` `viewer`|
|created_at|datetime|是|创建时间|

### 3.3 store

|字段|类型|必填|说明|
|---|---|---|---|
|id|string|是|店铺 ID|
|site_code|string|是|站点编码|
|store_name|string|是|店铺名称|
|currency|string|是|币种|
|timezone|string|是|时区|
|status|string|是|`active` / `paused`|

### 3.4 ad_account

|字段|类型|必填|说明|
|---|---|---|---|
|id|string|是|广告账户 ID|
|store_id|string|是|所属店铺|
|account_name|string|是|账户名称|
|status|string|是|账户状态|
|last_sync_at|datetime|否|最近同步时间|

### 3.5 ad_goal

|字段|类型|必填|说明|
|---|---|---|---|
|id|string|是|目标配置 ID|
|agent_type|string|是|固定为 `ad_agent`|
|store_id|string|是|店铺 ID|
|site_code|string|是|站点|
|goal_name|string|是|目标名称|
|acos_target|decimal|否|目标 ACOS|
|daily_budget_cap|decimal|否|日预算上限|
|risk_profile|string|是|`conservative` `balanced` `aggressive`|
|auto_approve_enabled|boolean|是|是否启用自动放行|
|auto_approve_budget_delta_pct|decimal|否|预算阈值|
|auto_approve_bid_delta_pct|decimal|否|竞价阈值|
|status|string|是|`active` / `paused`|
|effective_from|datetime|是|生效时间|
|updated_by|string|是|更新人|

### 3.6 context_snapshot

|字段|类型|必填|说明|
|---|---|---|---|
|id|string|是|快照 ID|
|agent_type|string|是|Agent 类型|
|store_id|string|是|店铺 ID|
|target_type|string|是|`campaign` `ad_group` `keyword` `search_term`|
|target_id|string|是|目标对象 ID|
|snapshot_date|date|是|快照日期|
|metrics_json|json|是|指标快照|
|source_version|string|是|数据版本|

### 3.7 suggestion

|字段|类型|必填|说明|
|---|---|---|---|
|id|string|是|建议 ID|
|agent_type|string|是|固定 `ad_agent`|
|store_id|string|是|店铺 ID|
|site_code|string|是|站点|
|goal_id|string|是|目标配置 ID|
|target_type|string|是|对象类型|
|target_id|string|是|对象 ID|
|suggestion_type|string|是|建议类型|
|title|string|是|建议标题|
|reason_summary|string|是|原因摘要|
|risk_level|string|是|风险等级|
|impact_estimate_json|json|否|预估影响|
|action_payload_json|json|是|结构化动作|
|status|string|是|建议状态|
|expires_at|datetime|否|失效时间|
|created_at|datetime|是|创建时间|

### 3.8 approval

|字段|类型|必填|说明|
|---|---|---|---|
|id|string|是|审批单 ID|
|suggestion_id|string|是|建议 ID|
|store_id|string|是|店铺 ID|
|risk_level|string|是|风险等级|
|status|string|是|审批状态|
|requested_by|string|是|发起人|
|approved_by|string|否|审批人|
|decision_note|string|否|备注|
|decided_at|datetime|否|处理时间|
|created_at|datetime|是|创建时间|

### 3.9 task

|字段|类型|必填|说明|
|---|---|---|---|
|id|string|是|任务 ID|
|agent_type|string|是|固定 `ad_agent`|
|suggestion_id|string|是|建议 ID|
|approval_id|string|否|审批单 ID|
|task_type|string|是|动作类型|
|target_type|string|是|对象类型|
|target_id|string|是|对象 ID|
|risk_level|string|是|风险等级|
|payload_json|json|是|执行参数|
|status|string|是|任务状态|
|retry_count|int|是|已重试次数|
|failure_reason|string|否|失败原因|
|created_by|string|是|创建人|
|approved_by|string|否|审批人|
|executed_at|datetime|否|开始执行时间|
|finished_at|datetime|否|完成时间|

### 3.10 task_event

|字段|类型|必填|说明|
|---|---|---|---|
|id|string|是|事件 ID|
|task_id|string|是|任务 ID|
|from_status|string|否|原状态|
|to_status|string|是|目标状态|
|event_type|string|是|状态变更类型|
|event_payload_json|json|否|附加信息|
|created_at|datetime|是|事件时间|

### 3.11 review_snapshot

|字段|类型|必填|说明|
|---|---|---|---|
|id|string|是|复盘 ID|
|agent_type|string|是|Agent 类型|
|task_id|string|是|任务 ID|
|store_id|string|是|店铺 ID|
|status|string|是|复盘状态|
|before_metrics_json|json|是|执行前指标|
|after_metrics_json|json|否|执行后指标|
|summary|string|否|复盘摘要|
|generated_at|datetime|是|生成时间|

### 3.12 notification

|字段|类型|必填|说明|
|---|---|---|---|
|id|string|是|消息 ID|
|user_id|string|是|接收用户|
|agent_type|string|是|Agent 类型|
|message_type|string|是|消息类型|
|priority|string|是|`low` `medium` `high`|
|title|string|是|标题|
|body|string|是|正文|
|target_type|string|否|跳转对象类型|
|target_id|string|否|跳转对象 ID|
|is_read|boolean|是|是否已读|
|created_at|datetime|是|创建时间|

### 3.13 audit_log

|字段|类型|必填|说明|
|---|---|---|---|
|id|string|是|日志 ID|
|agent_type|string|是|Agent 类型|
|actor_id|string|是|操作人|
|action|string|是|动作名称|
|target_type|string|是|对象类型|
|target_id|string|是|对象 ID|
|result|string|是|成功或失败|
|metadata_json|json|否|附加信息|
|created_at|datetime|是|创建时间|

## 4. API 约定

### 4.1 通用规范

1. 所有接口前缀统一为 `/api/v1`
2. 认证方式：`Authorization: Bearer <token>`
3. 时间字段统一使用 ISO 8601
4. 金额字段统一使用 decimal 字符串传输

### 4.2 通用响应格式

```json
{
  "code": "OK",
  "message": "success",
  "request_id": "req_123",
  "data": {}
}
```

### 4.3 错误响应格式

```json
{
  "code": "TASK_APPROVAL_REQUIRED",
  "message": "high risk task requires approval",
  "request_id": "req_123",
  "details": {
    "task_id": "task_001"
  }
}
```

## 5. REST API 契约

### 5.1 登录

`POST /api/v1/auth/login`

请求：

```json
{
  "account": "user@example.com",
  "password": "******"
}
```

响应：

```json
{
  "code": "OK",
  "message": "success",
  "data": {
    "token": "jwt-token",
    "user": {
      "id": "u_001",
      "display_name": "Alice"
    }
  }
}
```

### 5.2 获取当前用户

`GET /api/v1/me`

响应字段：

1. user
2. roles
3. stores
4. feature_flags

### 5.3 获取广告目标

`GET /api/v1/agent-goals?agent_type=ad_agent&store_id=store_001`

### 5.4 更新广告目标

`PATCH /api/v1/agent-goals/{goal_id}`

请求：

```json
{
  "acos_target": "28.00",
  "daily_budget_cap": "800.00",
  "risk_profile": "balanced",
  "auto_approve_enabled": true,
  "auto_approve_budget_delta_pct": "10.00",
  "auto_approve_bid_delta_pct": "8.00"
}
```

### 5.5 获取建议列表

`GET /api/v1/agents/ad/suggestions`

查询参数：

|参数|说明|
|---|---|
|store_id|店铺 ID|
|site_code|站点|
|status|建议状态|
|risk_level|风险等级|
|page|页码|
|page_size|分页大小|

返回字段：

1. list
2. total
3. unread_high_risk_count

### 5.6 获取建议详情

`GET /api/v1/agents/ad/suggestions/{suggestion_id}`

返回字段：

1. suggestion
2. goal
3. latest_context_snapshot
4. approval_preview

### 5.7 批准建议

`POST /api/v1/agents/ad/suggestions/{suggestion_id}/approve`

请求：

```json
{
  "note": "approved by operator",
  "execute_immediately": true
}
```

响应：

```json
{
  "code": "OK",
  "message": "success",
  "data": {
    "approval_id": "ap_001",
    "task_id": "task_001",
    "task_status": "queued"
  }
}
```

### 5.8 拒绝建议

`POST /api/v1/agents/ad/suggestions/{suggestion_id}/reject`

请求：

```json
{
  "note": "campaign protected manually"
}
```

### 5.9 批量审批

`POST /api/v1/agents/ad/suggestions/batch-approve`

请求：

```json
{
  "suggestion_ids": ["sg_001", "sg_002"],
  "note": "batch approved",
  "execute_immediately": true
}
```

### 5.10 获取任务列表

`GET /api/v1/tasks?agent_type=ad_agent&store_id=store_001`

返回字段：

1. task 基础信息
2. 关联 suggestion 摘要
3. 关联 approval 摘要

### 5.11 获取任务详情

`GET /api/v1/tasks/{task_id}`

返回字段：

1. task
2. task_events
3. audit_logs
4. review_status

### 5.12 取消任务

`POST /api/v1/tasks/{task_id}/cancel`

约束：

1. 仅 `queued` 任务可取消
2. `running` 任务只允许标记中断请求，不保证立即停止

### 5.13 重试任务

`POST /api/v1/tasks/{task_id}/retry`

约束：

1. 仅 `failed` 状态允许
2. 高风险任务需要再次审批

### 5.13.1 手动触发任务 Worker（开发调试）

`POST /api/v1/tasks/run-once`

请求：

```json
{
  "limit": 20
}
```

返回字段：

1. picked
2. succeeded
3. failed
4. skipped
5. results（task_id / status / message）

### 5.14 获取复盘

`GET /api/v1/agents/ad/reviews/{task_id}`

返回字段：

1. before_metrics
2. after_metrics
3. summary
4. status

状态说明：

1. `pending`：任务未完成或复盘尚未生成
2. `partial`：任务失败，仅有执行前快照
3. `ready`：任务完成，执行前后快照齐全

### 5.14.1 获取复盘列表

`GET /api/v1/agents/ad/reviews?limit=200`

返回字段：

1. list
2. total
3. status_counts（按 `ready/partial/pending` 统计）

### 5.15 获取通知列表

`GET /api/v1/notifications`

查询参数：

1. `is_read`
2. `priority`
3. `agent_type`

### 5.16 标记消息已读

`POST /api/v1/notifications/{notification_id}/read`

### 5.17 获取审计日志

`GET /api/v1/audit-logs`

查询参数：

1. `actor_id`
2. `agent_type`
3. `target_type`
4. `target_id`
5. `page`
6. `page_size`

## 6. WebSocket 事件契约

### 6.1 通道

`/ws/v1/events`

### 6.2 事件格式

```json
{
  "event_type": "suggestion.created",
  "event_id": "evt_001",
  "occurred_at": "2026-03-10T10:00:00Z",
  "payload": {}
}
```

### 6.3 事件列表

|事件名|说明|
|---|---|
|suggestion.created|新建议生成|
|suggestion.updated|建议状态变化|
|approval.updated|审批结果变化|
|task.updated|任务状态变化|
|review.generated|复盘生成完成|
|notification.created|新通知生成|

## 7. 关键 JSON 示例

### 7.1 suggestion.action_payload_json

```json
{
  "task_type": "budget_increase",
  "target_type": "campaign",
  "target_id": "cmp_001",
  "before": {
    "budget": "50.00"
  },
  "after": {
    "budget": "60.00"
  },
  "idempotency_key": "ad_agent:campaign:cmp_001:budget:20260310"
}
```

### 7.2 review_snapshot.before_metrics_json

```json
{
  "spend": "120.50",
  "orders": 8,
  "sales": "430.00",
  "acos": "28.02",
  "roas": "3.57"
}
```

## 8. 数据库索引建议

1. `suggestion(store_id, status, risk_level, created_at desc)`
2. `approval(suggestion_id, status)`
3. `task(agent_type, store_id, status, created_at desc)`
4. `notification(user_id, is_read, created_at desc)`
5. `audit_log(agent_type, target_type, target_id, created_at desc)`

## 9. 向后兼容要求

1. 所有对象必须保留 `agent_type` 字段，保证 V2 接第二个 Agent 时无需重构主表。
2. 所有执行任务通过统一 `task` 表和 `task_event` 表追踪。
3. 插件和 Web 端使用同一套对象 schema，不允许各自定义兼容层。
