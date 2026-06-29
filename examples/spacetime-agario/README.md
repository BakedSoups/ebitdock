# Spacetime Agario

This example is a small Agar.io-style Ebitengine WASM game wired for SpacetimeDB.

Development on this example is paused for now. The checked-in code is a scaffold that shows the intended stack shape, not a finished multiplayer demo.

## Tech Stack

- `Ebitengine`: renders the Agar.io-style browser game from Go.
- `Go WASM`: compiles `./cmd/game` into the browser client.
- `wasmserve`: serves the WASM dev build and browser shell during `ebitdock dev`.
- `SpacetimeDB`: intended authoritative multiplayer state for players, food, movement, growth, and respawn.
- `TypeScript bridge`: connects the browser to generated SpacetimeDB bindings and exposes a small `window.SpacetimeAgario` API to Go.
- `Docker Compose`: available through ebitdock for surrounding services, but this example does not currently start a local SpacetimeDB container.
- `ebitdock dashboard`: tracks dev status, ports, rebuilds, watched files, and logs.

## Ports

| Service | Port | Used For |
| --- | ---: | --- |
| `wasmserve` / web | `8080` | Browser game during dev |
| `dashboard` | `8081` | ebitdock dashboard |
| `SpacetimeDB maincloud` | external | Cloud-hosted multiplayer database when enabled |
| `SpacetimeDB local` | usually `3000` | Optional local SpacetimeDB server if you run `spacetime start` yourself |

## Current State

- The Ebitengine client boots with an offline fallback.
- The Rust SpacetimeDB module is present under `spacetimedb/agario`.
- The TypeScript bridge source is present under `web/src`.
- `static/js/agario-spacetime.js` is a fallback bridge so the game can load before generated bindings exist.
- The real SpacetimeDB path still needs bindings generated, bridge bundled, and a database published.

## Files

- `cmd/game`: starts the Ebitengine WASM game.
- `internal/game`: rendering, local fallback simulation, input, and JS bridge calls.
- `spacetimedb/agario`: Rust SpacetimeDB module.
- `web/src/agario-spacetime.ts`: browser bridge built from generated SpacetimeDB bindings.
- `static`: user-owned browser shell and fallback JS bridge.
- `ebitdock.yaml`: wasmserve dev server, dashboard, watch paths, and Docker settings.

## Later Setup

When we resume this example, the intended setup is:

1. Install SpacetimeDB.
2. Publish the module:

   ```sh
   cd examples/spacetime-agario/spacetimedb
   spacetime publish spacetime-agario --server maincloud --yes
   ```

3. Generate bindings and bundle the bridge:

   ```sh
   cd examples/spacetime-agario
   npm install
   npm run build:spacetime
   ```

4. Enable SpacetimeDB in `static/config.js`:

   ```js
   window.SPACETIME_AGARIO_CONFIG = {
     enabled: true,
     host: "wss://maincloud.spacetimedb.com",
     database: "spacetime-agario",
   };
   ```

5. Run:

   ```sh
   ebitdock doctor
   ebitdock dev
   ```

Open `http://localhost:8080` and use two browser tabs.
