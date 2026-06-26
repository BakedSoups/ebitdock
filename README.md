# ebitdock

`ebitdock` is a Docker-first local dev CLI for Ebitengine browser games.

I like Ebitengine: it is a great Go game engine that compiles Go code into WebAssembly and stays incredibly lightweight. The hard part is not the game engine. The hard part is port management and containerization when building layered IO games, browser games with backend databases, or games that need communication between multiple APIs. `ebitdock` is the small orchestration layer for that.

It does not generate your web app, hide Ebitengine, require Node.js, or become a game framework. Your project owns the game code, HTML shell, JS bridge, assets, APIs, and databases.

## Requirements

- Go
- Docker with the Compose plugin
- Linux/macOS first

## Install

From this repo:

```sh
go install ./cmd/ebitdock
```

Make sure Go's bin directory is on your PATH:

```sh
export PATH="$HOME/go/bin:$PATH"
```

## Commands

```sh
ebitdock init [name|.]
ebitdock dev
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

To compile only the browser build:

```sh
ebitdock wasm
```

`ebitdock build wasm` is the longer equivalent.

## What Dev Does

`ebitdock dev`:

- builds your Ebitengine game to WASM in a Go Docker container
- copies the matching `wasm_exec.js` from that same Go image
- writes `.ebitdock/compose.yaml`
- starts Docker Compose for the web/API services
- starts the local dashboard
- watches configured files and rebuilds WASM on source changes
- writes logs to `.ebitdock/ebitdock.log`

The dashboard shows ports, build/check status, watched paths, errors, and recent logs.

## Example Config

```yaml
project: my-game

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

For a basic single-player WASM game, Docker may be more than you need. For a live-service game, it gives you one repeatable local stack: web client, game WASM build, API server, realtime server, database/cache, ports, logs, and CI-compatible commands.

## GitHub Checks

The included GitHub Actions workflow runs formatting, vet, tests, CLI build, and an init smoke test on pull requests and pushes.
