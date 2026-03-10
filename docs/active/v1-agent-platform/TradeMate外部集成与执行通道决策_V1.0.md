# TradeMate 外部集成与执行通道决策

**文档版本**：V1.0  
**适用范围**：TradeMate V1 平台底座 + 广告 Agent  
**目的**：明确广告数据接入方式、OpenClaw 集成策略、各类动作的最终执行通道

## 1. 决策结论

### 1.1 广告数据接入方式

V1 明确采用：

1. `Amazon Ads API` 作为广告账户、Campaign、Ad Group、Keyword、Negative Keyword、Search Term 报表的权威数据源。
2. `Amazon Marketing Stream` 作为小时级近实时指标和 Campaign 变更消息源。
3. `Amazon Ads 报表拉取` 作为日级对账和历史回填机制。

不采用：

1. 浏览器抓取广告数据作为正式数据源。
2. OpenClaw 浏览器会话作为广告指标的权威来源。

### 1.2 OpenClaw 集成策略

V1 明确采用：

1. OpenClaw 作为 `runtime / browser capability / fallback automation layer`。
2. TradeMate 核心业务系统独立于 OpenClaw，包括：
   - 账号与权限
   - 目标与约束
   - 建议模型
   - 审批流
   - 任务中心
   - 审计与复盘
3. 集成方式优先采用 `workspace extensions / plugins`，不 fork OpenClaw 主干。

不采用：

1. 直接把全部业务系统写成 OpenClaw 内部插件集合。
2. 让 OpenClaw 直接承载平台核心业务数据库和状态机。

### 1.3 动作执行通道

V1 明确采用：

1. `API-first`
2. `Browser-fallback`
3. `No browser-first for supported ad actions`

也就是说：

1. 能通过 Amazon Ads API 完成的动作，全部走 API。
2. 浏览器执行仅作为正式例外通道，不作为默认路径。

## 2. 官方依据与推导

### 2.1 Amazon Ads API

Amazon 官方说明 Ads API 可以：

1. 以编程方式管理 Campaign。
2. 做关键词、出价、预算的自动优化。
3. 访问 Sponsored Ads、AMC、Amazon Marketing Stream。

来源：

1. Amazon Ads API 官方介绍  
   https://advertising.amazon.com/es-es/about-api

关键内容：

1. 官方写明可以“以编程方式管理 Campaign”。
2. 官方写明 API 可用于“自动优化 bids、keywords、budgets”。
3. 官方写明 API 接入覆盖 Sponsored Ads 和 Amazon Marketing Stream。

### 2.2 Amazon Marketing Stream

Amazon 官方说明 Marketing Stream：

1. 通过 Amazon Ads API 提供近实时的小时级指标和 Campaign 变更消息。
2. 适合做 intraday optimization。
3. 需要 AWS 作为数据接收端，使用相同 access token 即可订阅。

来源：

1. Amazon Marketing Stream 官方介绍  
   https://advertising.amazon.com/solutions/products/amazon-marketing-stream

### 2.3 OpenClaw

OpenClaw 官方文档说明：

1. 插件可以注册 Gateway RPC、HTTP routes、agent tools、background services。
2. 插件可通过 workspace extensions 加载，不必修改 OpenClaw 核心代码。
3. 浏览器能力支持 Chrome extension relay。
4. relay 需要用户手动 attach 到 tab，不会自动接管。

来源：

1. OpenClaw Plugins  
   https://docs.openclaw.ai/tools/plugin
2. OpenClaw Browser  
   https://docs.openclaw.ai/tools/browser

## 3. 广告数据接入决策

### 3.1 认证与授权

V1 采用：

1. Amazon Ads 官方第三方应用授权模式。
2. 每个广告账户必须由店铺主账号或管理员完成授权。
3. TradeMate 只保存 OAuth token 与账户映射，不保存广告平台账号密码。

理由：

1. Amazon 官方提供第三方应用授权与撤销机制。
2. 账户访问范围与授权用户权限一致，适合做权限映射。

### 3.2 数据分层

#### A. 控制面数据

对象：

1. ad_account
2. campaign
3. ad_group
4. keyword
5. negative_keyword

接入方式：

1. Amazon Ads API 拉取
2. 写操作后立即做定向回读

同步策略：

1. 首次接入做全量同步
2. 之后每 15 分钟增量同步一次
3. 每次成功写操作后 1 分钟内做定向刷新

#### B. 指标流数据

对象：

1. 小时级花费
2. 点击量
3. 销售额
4. ACOS / ROAS 相关基础指标
5. Campaign 变更消息

接入方式：

1. Amazon Marketing Stream
2. AWS SQS 作为默认接收端

同步策略：

1. 实时消费 stream 消息
2. 入库为小时级 metrics snapshot
3. 供建议引擎和实时提醒使用

#### C. 报表与回填数据

对象：

1. Search Term report
2. Placement / 其他日级报表
3. 对账与回填数据

接入方式：

1. Amazon Ads API reports

同步策略：

1. 每日店铺时区 `03:00` 启动 T+1 报表回填
2. 每日店铺时区 `05:00` 完成最近 14 天滚动修正
3. 对 Search Term 数据提供每日一次增量更新

### 3.3 V1 明确不做

1. 不通过浏览器页面解析广告指标作为正式口径。
2. 不用插件本地缓存替代后端权威广告数据。
3. 不用 OpenClaw 会话抓取历史报表来替代 Ads API。

## 4. OpenClaw 集成决策

### 4.1 集成位置

OpenClaw 在 TradeMate V1 中处于：

1. 浏览器上下文层
2. Agent runtime 层
3. 浏览器 fallback automation 层

不处于：

1. 核心业务状态存储层
2. 核心任务编排真源层
3. 核心审批和审计真源层

### 4.2 推荐集成方式

V1 采用：

1. 在工作区内通过 `.openclaw/extensions/` 或 plugin load paths 扩展。
2. 为 TradeMate 注册专用：
   - Gateway RPC
   - agent tools
   - background services
   - browser helper

不采用：

1. 修改 OpenClaw 核心仓库源码作为首版主路径。

### 4.3 浏览器集成方式

V1 采用：

1. Chrome extension relay
2. 本地或同机 Gateway
3. 用户手动 attach 到当前 tab

适用场景：

1. 插件现场上下文识别
2. 浏览器内工作现场辅助
3. API 不支持时的 fallback 执行

约束：

1. relay 不自动接管 tab
2. host browser control 需要允许 host control
3. 远程 Gateway 场景需要 node host 在浏览器所在机器上

### 4.4 TradeMate 与 OpenClaw 的边界

TradeMate 后端负责：

1. 数据接入
2. 规则引擎
3. suggestion / approval / task / review 状态机
4. 用户与权限
5. 审计与通知

OpenClaw 负责：

1. Browser relay
2. 浏览器工具调用
3. 未来非 API 覆盖动作的 fallback automation
4. 辅助型 agent tool runtime

## 5. 动作执行通道决策

### 5.1 总原则

执行通道优先级固定为：

1. 官方 API
2. 浏览器 fallback
3. 人工手动处理

### 5.2 V1 动作与通道矩阵

|动作|默认通道|备用通道|最终决策|
|---|---|---|---|
|Campaign 预算调整|Amazon Ads API|无|API|
|Keyword 竞价调整|Amazon Ads API|无|API|
|Campaign 暂停|Amazon Ads API|浏览器 fallback|API 优先|
|Campaign 恢复|Amazon Ads API|浏览器 fallback|API 优先|
|否词添加|Amazon Ads API|浏览器 fallback|API 优先|
|低效词降价|Amazon Ads API|无|API|
|低效词暂停|Amazon Ads API|浏览器 fallback|API 优先|
|高潜词加价|Amazon Ads API|无|API|

### 5.3 浏览器 fallback 触发条件

仅当以下条件之一满足时，允许从 API 切到浏览器执行：

1. 当前站点或账户未开放对应 API 能力。
2. 当前功能只在广告控制台可用，官方 API 暂未覆盖。
3. API 出现持续性异常，且该动作被标记为允许 fallback。
4. 用户显式授权本次使用浏览器执行。

### 5.4 浏览器 fallback 限制

1. fallback 动作必须单独记录 `execution_channel=browser_fallback`。
2. fallback 默认不参与自动放行。
3. fallback 一律要求人工审批。
4. fallback 不用于批量高频执行。

## 6. V1 最终工程实现

### 6.1 推荐实现架构

1. TradeMate 后端服务
   - Ads API 接入
   - Marketing Stream 消费
   - 建议引擎
   - 审批 / 任务 / 审计 / 复盘
2. Chrome 插件
   - 主入口
   - 现场提醒
   - 单条审批与查看
3. Web 管理后台
   - 配置
   - 批量审批
   - 任务中心
   - 复盘分析
4. OpenClaw 扩展层
   - browser relay
   - fallback browser actions
   - TradeMate agent tools

### 6.2 技术落地顺序

1. 先完成 Ads API + Marketing Stream 接入
2. 再完成 suggestion / approval / task 状态机
3. 再完成 API-first 的 7 类动作
4. 最后补 OpenClaw browser fallback

## 7. 结论摘要

### V1 最终拍板

1. 数据读取：`Amazon Ads API + Amazon Marketing Stream`
2. 数据权威源：`Amazon Ads API / Stream`
3. OpenClaw：`runtime + browser + fallback`
4. 动作执行：`API-first`
5. 浏览器执行：`仅做 fallback，不做主路径`
