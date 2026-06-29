# Spacetime Agario Plan

## Purpose

Show `ebitdock` as a dev orchestrator for online Ebitengine browser games with an authoritative realtime data layer.

## V1

- Browser-playable Ebitengine blob game.
- Local fallback simulation so the example boots without SpacetimeDB.
- SpacetimeDB Rust module with public `player` and `food` tables.
- TypeScript bridge generated from SpacetimeDB bindings.
- Go WASM client reads snapshots through `window.SpacetimeAgario`.
- wasmserve is the browser-facing dev server.

## Next

- Add local SpacetimeDB server service once the desired Docker image/CLI flow is stable.
- Add viewport subscriptions instead of subscribing to the full world.
- Move player collision fully server-side on a scheduled tick.
- Add spectator leaderboard and match reset controls.
- Add a dashboard card for SpacetimeDB publish/bindings status.
