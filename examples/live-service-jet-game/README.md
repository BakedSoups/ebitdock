# Live Service Jet Game

This example is a compact showcase for `ebitdock`: an Ebitengine WASM game prototype that needs more than a static web folder.

It demonstrates a browser game with a full local service stack:

- Ebitengine game compiled to WASM
- static web container serving the browser shell
- realtime WebSocket service for arena-state plumbing
- API service for player data
- Postgres database container
- admin/debug service
- ebitdock dashboard for ports, logs, build state, and service health

## Why This Is A Good Ebitdock Demo

Most small WASM games can be opened from a static page. Live-service games need more: ports, containers, logs, rebuilds, health checks, backend services, realtime sockets, and persistence.

This project keeps the game code Go-native while letting `ebitdock` orchestrate the development environment around it. Running one command starts the same kind of stack a browser game naturally grows into when it adds APIs, persistence, and realtime services.

## Run It

From this directory:

```sh
ebitdock doctor
ebitdock dev
```

Then open:

```text
Game:      http://localhost:8090
Dashboard: http://localhost:8091
```

Stop the stack with:

```sh
ebitdock down
```

## Services

`ebitdock.yaml` defines the full local stack:

```text
web       :8090  static browser client
api       :3001  player/profile service
realtime  :3002  WebSocket arena-state prototype
admin     :9090  local debug/admin service
database  :5432  Postgres
dashboard :8091  ebitdock dashboard
```

## Gameplay

Controls:

```text
W        thrust forward
A / D    rotate
Space    shoot
1-4      upgrade speed, turn, damage, fire rate
R/Enter  respawn after death
```

The current game loop includes login, pickups, visible shooting, XP, level points, upgrades, death, and respawn. The realtime service is present as development plumbing and is still a prototype, so treat this as a live-service stack demo rather than finished multiplayer gameplay.

## What To Watch In Ebitdock

The dashboard is the main point of the example:

- Ports show every service in one place.
- Logs can be viewed globally or by service.
- WASM rebuild status shows the trigger file and build output.
- Docker Compose services are started from the generated `.ebitdock/compose.yaml`.
- Source changes rebuild the game WASM.

This makes the example useful for testing ebitdock itself and for showing how Ebitengine projects can grow into multi-service browser games without adopting a Node framework.
