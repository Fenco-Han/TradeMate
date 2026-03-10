# TradeMate OpenClaw Extension

This directory hosts the initial OpenClaw extension skeleton for TradeMate.

## Current scope

1. Manifest and extension directory shape
2. Relay status RPC set
3. Browser fallback action contract
4. Fallback action stubs (pause/resume campaign, add negative keyword, pause keyword)

## Implemented RPC

1. `trademate.getActiveBrowserContext`
2. `trademate.attachBrowserRelay`
3. `trademate.getRelayStatus`
4. `trademate.runBrowserAction`

## Implemented tools

1. `ad_action_preview`
2. `browser_fallback_prepare`
3. `browser_fallback_execute`
4. `relay_health_check`

## Contract flow

Each browser fallback action follows:

1. `validateContext(ctx)`
2. `prepare(ctx, payload)`
3. `execute(ctx, preparedPayload)`
4. `verify(ctx, preparedPayload, executeResult)`

The contract implementation lives in `browser/action_contract.js`.
