# TradeMate OpenClaw 扩展设计

**文档版本**：V1.0  
**适用范围**：TradeMate V1  
**目的**：明确 OpenClaw 在本项目中的扩展目录、接口边界和运行模式

## 1. 设计结论

TradeMate V1 不修改 OpenClaw 核心主干，采用工作区扩展方式集成。

建议目录：

```text
.openclaw/
└─ extensions/
   └─ trademate/
      ├─ manifest.json
      ├─ rpc/
      ├─ tools/
      ├─ browser/
      └─ background/
```

## 2. 扩展职责

### 2.1 RPC

用于桥接 TradeMate 后端与 OpenClaw runtime：

1. `trademate.getActiveBrowserContext`
2. `trademate.attachBrowserRelay`
3. `trademate.runBrowserAction`
4. `trademate.getRelayStatus`

### 2.2 Tools

用于 Agent runtime 调用受控能力：

1. `ad_action_preview`
2. `browser_fallback_prepare`
3. `browser_fallback_execute`
4. `relay_health_check`

### 2.3 Browser helpers

用于统一浏览器 fallback 逻辑：

1. attach 当前 tab
2. 定位广告控制台页面
3. 校验当前页面是否匹配期望状态
4. 执行最小动作并返回结果

### 2.4 Background services

用于维持轻量状态同步：

1. relay 状态轮询
2. 浏览器动作超时监控
3. fallback 审计补充上报

## 3. 扩展接口草案

### 3.1 RPC: `trademate.getRelayStatus`

请求：

```json
{
  "tab_id": 123
}
```

响应：

```json
{
  "attached": true,
  "browser": "chrome",
  "tab_id": 123,
  "url": "https://advertising.amazon.com/..."
}
```

### 3.2 RPC: `trademate.runBrowserAction`

请求：

```json
{
  "action_name": "pause_campaign",
  "task_id": "task_001",
  "target_id": "cmp_001",
  "payload": {
    "campaign_id": "cmp_001"
  }
}
```

响应：

```json
{
  "accepted": true,
  "execution_id": "bx_001"
}
```

## 4. Browser action contract

每个浏览器动作必须实现统一 contract：

1. `validateContext(ctx)`
2. `prepare(ctx, payload)`
3. `execute(ctx, payload)`
4. `verify(ctx, payload)`
5. `summarize(result)`

## 5. V1 支持的 browser fallback actions

1. `pause_campaign`
2. `resume_campaign`
3. `add_negative_keyword`
4. `pause_keyword`

说明：

1. 这些动作不是默认路径。
2. 仅在 API 不可用且满足 fallback 条件时启用。

## 6. 与后端边界

TradeMate 后端负责：

1. 决定是否允许 fallback
2. 下发明确 action_name 和 payload
3. 写 task / audit / notification 主记录

OpenClaw 扩展负责：

1. 执行浏览器动作
2. 返回执行结果
3. 返回必要的上下文证据

## 7. 错误码约定

|错误码|说明|
|---|---|
|RELAY_NOT_ATTACHED|未 attach 当前 tab|
|UNSUPPORTED_PAGE|当前页面不支持动作|
|ACTION_TIMEOUT|动作执行超时|
|VERIFY_FAILED|执行后校验失败|
|HOST_CONTROL_DENIED|未允许 host browser control|

## 8. 开发优先级

1. 先实现 relay 状态检查
2. 再实现浏览器动作通用 contract
3. 最后补单个 fallback action
