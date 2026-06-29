# Live Service Jet Game

This example is a multiplayer space-snake / jet arena prototype. It is a compact showcase for `ebitdock`: an Ebitengine WASM game that needs more than a static web folder.


https://github.com/user-attachments/assets/b536490b-677a-4d7f-af8d-a6488d756cda

<p align="center">
  <img src="https://github.com/user-attachments/assets/232c0ddf-930e-45f1-be24-e2bb97bc28f2" width="130" valign="middle">
  <img src="https://github.com/user-attachments/assets/1cf0453b-89e7-4a93-a755-8f4add089698" width="350" valign="middle">
</p>


## Tech Stack

- `Ebitengine`: renders the browser game from Go.
- `Go WASM`: compiles `./cmd/game` into `./static/game.wasm`.
- `Docker WASM build`: `ebitdock` builds the game inside the configured Go image.
- `nginx`: serves the static browser shell from `./static`.
- `Go API service`: handles player/profile-style backend data.
- `Go realtime service`: WebSocket arena-state prototype for multiplayer movement, bullets, crystals, and respawn.
- `Postgres`: persistence layer for service data.
- `Go admin service`: local debug/admin surface for the stack.
- `Docker Compose`: runs the app services as containers from `.ebitdock/compose.yaml`.
- `ebitdock dashboard`: shows ports, build state, service logs, watched files, and errors.
--- 

This project keeps the game code Go-native while letting `ebitdock` orchestrate the development environment around it. Running one command starts the same kind of stack a browser game naturally grows into when it adds APIs, persistence, and realtime services.

## How Ebitdock Manages It

`ebitdock dev` reads `ebitdock.yaml`, builds the WASM game, writes a Docker Compose file, starts the configured containers, watches source files, rebuilds on game changes, and opens the dashboard.

In this example, `wasm.dev_server: docker` is used because the browser shell loads `game.wasm` from the static web container. That means the full browser app is served by Docker instead of wasmserve.

The generated Compose stack includes:

- `web`: static nginx container for the browser client.
- `api`: Go backend container.
- `realtime`: Go WebSocket container.
- `admin`: Go debug/admin container.
- `database`: Postgres container with a named volume.

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

## Ports

`ebitdock.yaml` defines the full local stack:

| Service | Port | Runtime | Used For |
| --- | ---: | --- | --- |
| `web` | `8090` | `nginx:1.27-alpine` | Static browser shell, `game.wasm`, `wasm_exec.js`, CSS |
| `api` | `3001` | `golang:1.24` | Player/profile API service |
| `realtime` | `3002` | `golang:1.24` | WebSocket arena-state prototype |
| `database` | `5432` | `postgres:16-alpine` | Persistent game/service data |
| `admin` | `9090` | `golang:1.24` | Local debug/admin page |
| `dashboard` | `8091` | host ebitdock process | Ports, logs, build state, watch state, errors |

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
- `ebitdock down` tears down the generated Compose stack and releases the ports.

This makes the example useful for testing ebitdock itself and for showing how Ebitengine projects can grow into multi-service browser games without adopting a Node framework.
