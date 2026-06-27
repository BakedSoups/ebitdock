# ebitdock

<p align="center">
  <img src="https://github.com/user-attachments/assets/ae9adc1b-b700-4d4b-a323-1cd6fb137cda" width="120" valign="middle">
  <img src="https://github.com/user-attachments/assets/232c0ddf-930e-45f1-be24-e2bb97bc28f2" width="120" valign="middle">
</p>

Ebitengine is a lightweight way to build games in Go, and its WebAssembly target makes browser games feel natural without pulling in a JavaScript framework.

I built `ebitdock` while experimenting with live-service browser games in Ebitengine. The game loop was the easy part; the surrounding stack was the part that kept growing: WASM builds, static web serving, realtime services, APIs, databases, ports, logs, rebuilds, and deployment shape.

After using `wasmserve` and Docker Compose together, this turned into a small orchestrator for the pieces around an Ebitengine WASM game. I open sourced it because it may help other people building browser games that need more than a single static HTML file.

`ebitdock` is a Go-native dev orchestrator for Ebitengine WASM games. It containerizes the local development stack around your game so the same shape can move toward CI/CD, deployment, backend services, realtime multiplayer, databases, dashboards, and repeatable builds.

It does not generate your web app, hide Ebitengine, require Node.js, or become a game framework. Your project owns the game code, HTML shell, JS bridge, assets, APIs, and databases. `ebitdock` owns the orchestration around them.

https://github.com/user-attachments/assets/0d759e8a-a444-49ab-b845-e4a6491ce40e

## Demo

`ebitdock dev` builds the Ebitengine WASM game, starts the local Docker Compose stack, serves the browser client, and exposes a dashboard for ports, logs, build status, and service health.

The example live-service game runs web, API, realtime, admin, and Postgres services together.

## Dashboard

Keep track of ports, service status, WASM build state, checks, watched files, and logs from one local dashboard.

https://github.com/user-attachments/assets/dc2699cc-467a-4abf-a762-ec1deef21e3f

## Containerized WASM Stack

Build the Ebitengine WASM output in a Go container and run the surrounding services through Docker Compose, which makes the development stack easier to reproduce in CI/CD pipelines.

https://github.com/user-attachments/assets/e9ad2063-6cc1-43a4-ac03-2d865bf3c105



## Requirements

- Go
- Docker with the Compose plugin
- wasmserve
- Linux/macOS first

## Install

From this repo:

```sh
go install ./cmd/ebitdock
ebitdock install tools
```

Make sure Go's bin directory is on your PATH:

```sh
export PATH="$HOME/go/bin:$PATH"
```

`ebitdock install tools` installs Go-based helper tools such as `wasmserve`. Docker is installed through your OS or Docker Desktop; `ebitdock doctor` will tell you if Docker or the Compose plugin is missing.

## Commands

```sh
ebitdock init [name|.]
ebitdock install tools
ebitdock dev
ebitdock down
ebitdock wasm
ebitdock build wasm
ebitdock logs
ebitdock doctor
```

## Existing Project

From your Ebitengine repo:

```sh
ebitdock init
```

This writes only `ebitdock.yaml`. It does not overwrite your game, static files, assets, or backend.

Edit the generated config so `game.package`, `game.output`, `services.web.root`, and any API ports match your project.

Then run:

```sh
ebitdock doctor
ebitdock dev
```

`ebitdock doctor` is the toolchain check. It reports missing Docker, missing `wasmserve`, bad ports, missing game packages, and static root issues before you start a dev session.

To compile only the browser build:

```sh
ebitdock wasm
```

`ebitdock build wasm` is the longer equivalent.

## What Dev Does

`ebitdock dev`:

- verifies `wasmserve` is installed for Ebitengine browser dev diagnostics
- builds your Ebitengine game to WASM in a Go Docker container
- copies the matching `wasm_exec.js` from that same Go image
- writes `.ebitdock/compose.yaml`
- starts Docker Compose for the web/API services
- starts the local dashboard
- watches configured files and rebuilds WASM on source changes
- writes logs to `.ebitdock/ebitdock.log`

The dashboard shows ports, build/check status, watched paths, errors, and recent logs.

To stop the containers and release the ports opened by the ebitdock Compose stack:

```sh
ebitdock down
```

## Example Config

```yaml
project: my-game

game:
  package: ./cmd/game
  output: ./static/game.wasm

wasm:
  exec: ./static/wasm_exec.js

docker:
  compose_file: ./.ebitdock/compose.yaml
  go_image: golang:1.24

services:
  web:
    root: ./static
    port: 8080
    image: nginx:1.27-alpine
    workdir: /usr/share/nginx/html
    volumes:
      - ./static:/usr/share/nginx/html:ro

  api:
    enabled: false
    command: go run ./server
    port: 3001
    image: golang:1.24
    workdir: /app
    volumes:
      - .:/app

dashboard:
  port: 8081

watch:
  rebuild:
    - ./cmd/**/*.go
    - ./internal/**/*.go
    - ./assets/**
  static:
    - ./static/**
```

## Why This Helps

Ebitengine keeps the game loop Go-native and lightweight. `ebitdock` handles the surrounding dev orchestration: containerized WASM builds, wasmserve-aware browser diagnostics, static web serving, service ports, logs, health, databases, realtime backends, and dashboard visibility.

That makes it useful for simple browser builds and especially for live-service games that need more than one process.

## GitHub Checks

The included GitHub Actions workflow runs formatting, vet, tests, CLI build, and an init smoke test on pull requests and pushes.
