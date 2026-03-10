# TradeMate 广告动作执行器接口定义

**文档版本**：V1.0  
**适用范围**：V1 广告 Agent  
**目的**：定义 7 类正式动作的统一执行器接口

## 1. 通用执行器接口

所有执行器实现统一接口：

```go
type ActionExecutor interface {
    Name() string
    Channel() string
    Validate(ctx Context, payload Payload) error
    Risk(ctx Context, payload Payload) RiskLevel
    Execute(ctx Context, payload Payload) (ExecutionResult, error)
    Verify(ctx Context, payload Payload, result ExecutionResult) error
}
```

## 2. 公共输入模型

```json
{
  "task_id": "task_001",
  "store_id": "store_001",
  "site_code": "US",
  "target_type": "campaign",
  "target_id": "cmp_001",
  "execution_channel": "api",
  "payload": {}
}
```

## 3. 公共返回模型

```json
{
  "execution_id": "exec_001",
  "channel": "api",
  "status": "success",
  "raw_result": {},
  "summary": "campaign budget updated from 50 to 60",
  "finished_at": "2026-03-10T10:00:00Z"
}
```

## 4. 动作定义

### 4.1 budget_increase

#### 默认通道

`api`

#### payload

```json
{
  "campaign_id": "cmp_001",
  "before_budget": "50.00",
  "after_budget": "60.00"
}
```

#### validate

1. `after_budget > before_budget`
2. 不超过 `daily_budget_cap`

### 4.2 budget_decrease

#### 默认通道

`api`

#### payload

```json
{
  "campaign_id": "cmp_001",
  "before_budget": "60.00",
  "after_budget": "50.00"
}
```

#### validate

1. `after_budget < before_budget`
2. 不低于系统最小预算

### 4.3 bid_increase

#### 默认通道

`api`

#### payload

```json
{
  "keyword_id": "kw_001",
  "before_bid": "0.95",
  "after_bid": "1.05"
}
```

### 4.4 bid_decrease

#### 默认通道

`api`

#### payload

```json
{
  "keyword_id": "kw_001",
  "before_bid": "1.05",
  "after_bid": "0.90"
}
```

### 4.5 campaign_pause

#### 默认通道

`api`

#### 备用通道

`browser_fallback`

#### payload

```json
{
  "campaign_id": "cmp_001",
  "reason": "high_spend_low_conversion"
}
```

### 4.6 campaign_resume

#### 默认通道

`api`

#### 备用通道

`browser_fallback`

#### payload

```json
{
  "campaign_id": "cmp_001",
  "reason": "recovery_conditions_met"
}
```

### 4.7 negative_keyword_add

#### 默认通道

`api`

#### 备用通道

`browser_fallback`

#### payload

```json
{
  "campaign_id": "cmp_001",
  "ad_group_id": "ag_001",
  "keyword_text": "bad keyword",
  "match_type": "negative_phrase"
}
```

## 5. 执行通道决策

### 5.1 API executor

适用：

1. budget_increase
2. budget_decrease
3. bid_increase
4. bid_decrease
5. campaign_pause
6. campaign_resume
7. negative_keyword_add

### 5.2 browser fallback executor

仅适用：

1. campaign_pause
2. campaign_resume
3. negative_keyword_add

条件：

1. API 通道不可用
2. 当前任务明确允许 fallback
3. 已完成人工审批

## 6. 验证规则

每个执行器执行后必须至少完成一项验证：

1. API 返回成功且读取到目标对象新状态
2. browser fallback 返回成功且 verify 通过

若验证失败：

1. 任务状态记为 `failed`
2. `failure_reason=VERIFY_FAILED`

## 7. 幂等规则

幂等键建议格式：

`{agent_type}:{task_type}:{target_id}:{normalized_after_value}:{yyyymmddhh}`

示例：

`ad_agent:budget_increase:cmp_001:60.00:2026031010`

## 8. 重试策略

|动作|自动重试|最大次数|
|---|---|---|
|budget_increase|是|2|
|budget_decrease|是|2|
|bid_increase|是|2|
|bid_decrease|是|2|
|campaign_pause|否|0|
|campaign_resume|否|0|
|negative_keyword_add|否|0|

## 9. 审计字段要求

每次执行必须记录：

1. task_id
2. executor_name
3. execution_channel
4. before_payload
5. after_payload
6. result_status
7. result_summary
