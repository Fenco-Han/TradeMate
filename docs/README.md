# Docs Guide

本目录已按“当前有效文档 / 历史文档 / 调研 / 参考素材”分层整理，供人类和其他 agent 快速识别。

## 目录结构

```text
docs/
├─ active/
│  └─ v1-agent-platform/   # 当前有效基线，优先阅读
├─ archive/
│  └─ v4/                  # 历史 PRD / 技术方案，仅供回溯
├─ research/               # 调研材料
├─ reference-designs/      # 参考设计图
└─ README.md               # 本索引
```

## 阅读顺序

如果你是 agent，默认只读 `docs/active/v1-agent-platform/`，除非需要回溯历史决策。

推荐阅读顺序：

1. `docs/active/v1-agent-platform/TradeMate Agent平台产品需求文档_PRD_V1.0.md`
2. `docs/active/v1-agent-platform/TradeMate Agent平台技术方案_V1.0.md`
3. `docs/active/v1-agent-platform/TradeMate外部集成与执行通道决策_V1.0.md`
4. `docs/active/v1-agent-platform/TradeMate广告Agent功能规格说明_V1.0.md`
5. `docs/active/v1-agent-platform/TradeMate核心数据模型与API契约_V1.0.md`
6. `docs/active/v1-agent-platform/TradeMate数据库SQL草案_V1.0.sql`
7. `docs/active/v1-agent-platform/TradeMate OpenClaw扩展设计_V1.0.md`
8. `docs/active/v1-agent-platform/TradeMate广告动作执行器接口定义_V1.0.md`
9. `docs/active/v1-agent-platform/TradeMate插件与Web页面原型交互说明_V1.0.md`
10. `docs/active/v1-agent-platform/TradeMate开发计划_V1.0.md`
11. `docs/active/v1-agent-platform/TradeMate Codex任务清单与执行计划_V1.0.md`

## 目录说明

### `active/v1-agent-platform`

当前生效文档，代表 TradeMate 最新产品与技术基线：

1. 平台级 PRD
2. 平台级技术方案
3. 外部集成与执行通道决策
4. 广告 Agent 功能规格
5. 核心数据模型与 API 契约
6. 数据库 SQL 草案
7. OpenClaw 扩展设计
8. 广告动作执行器接口定义
9. 插件与 Web 页面原型交互说明
10. 开发计划
11. Codex 任务清单与执行计划

### `archive/v4`

历史版本文档，仅用于参考旧思路和需求来源，不作为当前开发基线：

1. 旧版插件 PRD
2. 旧版技术方案
3. 旧版 V4 技术方案

### `research`

调研材料。当前主要为：

1. 亚马逊跨境电商插件产品调研报告

### `reference-designs`

竞品和参考界面截图，仅用于交互和视觉参考，不代表当前信息架构或功能范围。

## 当前有效基线

当前所有开发、评审、拆解任务默认以 `docs/active/v1-agent-platform/` 下文档为准。

如与 `archive/` 中内容冲突，以 `active/` 为准。
