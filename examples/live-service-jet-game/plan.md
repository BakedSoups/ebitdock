# Multiplayer Space Snake Example Plan

## Purpose

This example should show why `ebitdock` is useful for Ebitengine games that are more than a standalone WASM file.

The game should be small enough to build, easy to understand visually, and still require multiple local services. A Snake.io-style spaceship game fits that well: simple 2D movement, clear multiplayer state, pickups, upgrades, persistence, and realtime networking.

## Working Concept

Working title: `orbit-snake`

Players control small spaceships in a shared 2D arena. Each ship leaves a glowing energy trail like a snake. Players collect space crystals to grow their trail, earn scrap, and buy upgrades. Hitting another trail destroys or damages the ship. Bigger ships are stronger but harder to maneuver.

Think:

- `snake.io` style movement and growth
- spaceship theme
- resource pickups
- simple upgrades
- one shared multiplayer arena
- persistent player progression

## Why This Is A Good Ebitdock Demo

This game is small, but it naturally needs a multi-port stack:

```text
browser client   Ebitengine WASM game
web service      serves static files and WASM
api service      player profile, upgrades, inventory
realtime service WebSocket positions, trails, pickups, collisions
database         persistent player state
admin service    local debug/reset tools
dashboard        ebitdock ports, logs, builds, service status
```

That makes `ebitdock dev` meaningful without making the game too large.

## Core Gameplay Loop

1. Player opens the browser client.
2. Client loads profile/upgrades from the API.
3. Client connects to realtime WebSocket.
4. Player pilots a ship around the arena.
5. Player collects crystals to grow and score points.
6. Trail length increases as crystals are collected.
7. Player avoids enemy trails and arena hazards.
8. Player banks scrap after a run.
9. Player buys upgrades.
10. Player re-enters with better stats.

## MVP Gameplay

- top-down 2D arena
- one ship per connected player
- smooth turning movement
- energy trail behind each ship
- crystal pickups
- trail collision
- death/respawn
- score counter
- one upgrade path
- API-backed player profile
- realtime WebSocket state sync

## Stretch Gameplay

- AI ships
- shield pickup
- boost pickup
- rare crystals
- team mode
- leaderboard
- seasonal/resettable arena
- planet obstacles
- wormhole teleports

## Controls

- `W` or up arrow: thrust
- `A/D` or left/right arrows: turn
- space: boost
- mouse/touch later for mobile steering

## Visual Style

Keep the visuals simple and readable:

- dark space background
- ships as triangles or small sprites
- trails as glowing colored segments
- crystals as bright polygons
- planets/asteroids as static obstacles
- compact HUD for score, length, scrap, upgrades

No heavy asset requirement for the first version.

## Why Multiple Ports Are Needed

### `8090` web

Serves:

- `index.html`
- `wasm_exec.js`
- `game.wasm`
- static assets

### `3001` api

Owns durable player state:

- profile
- total scrap
- unlocked upgrades
- selected ship color
- match history later

Example endpoints:

```text
GET  /health
GET  /players/{id}
POST /players/{id}/scrap
POST /players/{id}/upgrades
GET  /leaderboard
```

### `3002` realtime

Owns live arena state:

- connected players
- ship positions
- trail segments
- crystal spawns
- collisions
- deaths
- respawns

Example endpoints:

```text
GET /health
GET /ws
```

### `5432` database

Persists:

- players
- scrap
- upgrades
- high scores
- match summaries

### `9090` admin

Local-only debug service:

- reset arena
- spawn crystals
- kill test player
- inspect connected players
- inspect current pickups/trails

## Proposed Folder Shape

```text
examples/live-service-jet-game/
  plan.md
  ebitdock.yaml
  static/
    index.html
    style.css
    game.wasm
    wasm_exec.js
    assets/
  cmd/
    game/
      main.go
    api/
      main.go
    realtime/
      main.go
    admin/
      main.go
  internal/
    game/
      arena.go
      player.go
      render.go
      input.go
      net.go
    protocol/
      messages.go
    api/
      handlers.go
      store.go
    realtime/
      hub.go
      arena.go
      collision.go
    shared/
      types.go
  migrations/
    001_init.sql
  assets/
```

The folder name can be renamed later to `examples/orbit-snake/`.

## Target Ebitdock Config

This intentionally pushes the next `ebitdock` requirement: generic named services. Current code supports fixed `web` and `api`; this example should drive support for `realtime`, `database`, and `admin`.

```yaml
project: orbit-snake

game:
  package: ./cmd/game
  output: ./static/game.wasm

wasm:
  exec: ./static/wasm_exec.js

docker:
  enabled: true
  compose_file: ./.ebitdock/compose.yaml
  go_image: golang:1.24

services:
  web:
    kind: static
    root: ./static
    port: 8090
    image: nginx:1.27-alpine
    volumes:
      - ./static:/usr/share/nginx/html:ro

  api:
    kind: go
    enabled: true
    command: go run ./cmd/api
    port: 3001
    env:
      PORT: "3001"
      DATABASE_URL: postgres://game:game@database:5432/game?sslmode=disable
    depends_on:
      - database

  realtime:
    kind: go
    enabled: true
    command: go run ./cmd/realtime
    port: 3002
    env:
      PORT: "3002"
      API_URL: http://api:3001
    depends_on:
      - api

  admin:
    kind: go
    enabled: true
    command: go run ./cmd/admin
    port: 9090
    env:
      PORT: "9090"
      API_URL: http://api:3001
      REALTIME_URL: http://realtime:3002

  database:
    kind: postgres
    enabled: true
    image: postgres:16-alpine
    port: 5432
    env:
      POSTGRES_USER: game
      POSTGRES_PASSWORD: game
      POSTGRES_DB: game
    volumes:
      - orbit-snake-db:/var/lib/postgresql/data

dashboard:
  port: 8091

watch:
  rebuild:
    - ./cmd/game/**/*.go
    - ./internal/game/**/*.go
    - ./internal/protocol/**/*.go
    - ./assets/**
  static:
    - ./static/**
  restart:
    api:
      - ./cmd/api/**/*.go
      - ./internal/api/**/*.go
      - ./internal/shared/**/*.go
    realtime:
      - ./cmd/realtime/**/*.go
      - ./internal/realtime/**/*.go
      - ./internal/protocol/**/*.go
    admin:
      - ./cmd/admin/**/*.go
      - ./internal/**/*.go
```

## Data Model Sketch

### Player

- id
- name
- color
- total_scrap
- high_score
- created_at

### Upgrade

- player_id
- speed_level
- turn_level
- boost_level
- shield_level

### MatchSummary

- id
- player_id
- score
- crystals_collected
- ships_destroyed
- duration_seconds
- ended_at

## Realtime State Sketch

### Ship

- player_id
- x
- y
- angle
- speed
- alive
- score
- trail_segments

### Crystal

- id
- x
- y
- value
- rarity

### Arena

- width
- height
- ships
- crystals
- obstacles

## Client/Server Message Sketch

Use plain JSON first.

Client to realtime:

```json
{
  "type": "input",
  "player_id": "p1",
  "turn": -1,
  "thrust": true,
  "boost": false
}
```

Realtime to client:

```json
{
  "type": "state",
  "ships": [],
  "crystals": [],
  "events": []
}
```

Useful message types:

- `join`
- `input`
- `state`
- `crystal.collected`
- `ship.hit`
- `ship.dead`
- `ship.respawn`
- `score.updated`

## What Ebitdock Should Show

Dashboard should show:

- web: `http://localhost:8090`
- dashboard: `http://localhost:8091`
- api: `http://localhost:3001`
- realtime: `ws://localhost:3002/ws`
- admin: `http://localhost:9090`
- database: `localhost:5432`
- WASM build status
- last build duration
- service health
- grouped logs per service
- watched paths
- current errors

Terminal output should feel like Docker Compose plus Go tooling:

```text
SERVICE     STATUS    PORTS
web         running   8090->8090
api         running   3001->3001
realtime    running   3002->3002
admin       running   9090->9090
database    running   5432->5432
dashboard   running   8091
wasm        ok        431ms
watch       active    9 patterns
```

## Implementation Phases

### Phase 1: Planning Example

- keep this `plan.md`
- add a target `ebitdock.yaml`
- document missing generic-service support

### Phase 2: Generic Service Model In Ebitdock

Update `ebitdock` itself:

- replace fixed `services.web`/`services.api` model with named services
- keep `web` as a conventional service
- support service kind: `static`, `go`, `postgres`, `custom`
- support `depends_on`
- support ports, env, volumes, command, image, dockerfile
- dashboard should list all services

### Phase 3: Compose Generation

Generate compose for all enabled services:

- static web container
- Go command containers
- Postgres container
- named volumes
- dependency order
- stable service names

### Phase 4: Minimal Local Client

Build the Ebitengine client with local-only simulation first:

- ship movement
- trail rendering
- crystal pickups
- score
- death on trail collision

### Phase 5: API And Realtime

Add Go services:

- API with in-memory player profile
- realtime WebSocket hub
- shared protocol package
- local health endpoints

### Phase 6: Persistence

Add Postgres:

- migrations
- player profile persistence
- upgrades
- high scores

### Phase 7: Dashboard Improvements

Use the example to drive:

- grouped service logs
- health checks
- port table
- compose status
- restart buttons later

## Success Criteria

The example is successful when:

- `ebitdock dev` starts web, api, realtime, admin, database, and dashboard
- browser loads the Ebitengine WASM client
- player can steer a ship and collect crystals
- ship trail grows after collecting crystals
- realtime service receives input messages
- API stores scrap/upgrades
- database persists profile state across restarts
- dashboard clearly shows every service and port
- editing client Go code rebuilds WASM
- editing API/realtime code restarts the right service

## Design Constraints

- no Node.js requirement
- Go services by default
- Docker Compose as the local runtime
- generated files should be readable
- config should be explicit
- Ebitengine remains visible and normal
- the example should be understandable without cloud infrastructure

## Open Questions

- Should collisions be client-predicted or fully server authoritative?
- Should the realtime service tick at 10, 20, or 30 Hz for the MVP?
- Should the first version allow shooting, or only trail collision?
- Should `ebitdock logs` stream compose logs or read its own grouped log files?
- Should ebitdock restart services from file changes or rely on Docker Compose watch later?
