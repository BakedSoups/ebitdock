# ebitdock

`ebitdock` is local orchestration for Ebitengine browser games.

For a small single-player WASM game, it is convenience: build WASM, serve static files, watch changes, and show errors. For multiplayer or live-service browser games, it becomes the useful abstraction around the game: ports, APIs, logs, dashboards, and local services in one place.

It does not generate your web app, hide Ebitengine, require Node.js, or require Docker. Your project owns its HTML, JS bridge, audio setup, assets, and backend code.

## Why

Ebitengine makes the game part pleasant: write Go, compile to WebAssembly, run in the browser.

The awkward part is everything around the game once it grows beyond a single WASM file:

```text
game client      Ebitengine WASM
web shell        static files, audio, JS bridge
api server       scores, auth, inventory, content
realtime server  rooms, matchmaking, WebSockets
database/cache   local state for development
dashboard        ports, logs, build status
```

`ebitdock dev` is intended to become the one command that starts and tracks that local stack.

## Install

```sh
go install ./cmd/ebitdock
```

During local development:

```sh
go run ./cmd/ebitdock --help
```

## Commands

```sh
ebitdock init [name|.]
ebitdock dev
ebitdock build wasm
ebitdock logs
```

## Quick Start

For an existing Ebitengine project:

```sh
cd /path/to/your-game
ebitdock init
ebitdock dev
```

`init` writes only `ebitdock.yaml`. It does not overwrite or generate your game, web shell, assets, or backend.

For a basic project folder:

```sh
ebitdock init my-game
```

This creates:

```text
my-game/
  ebitdock.yaml
```

Edit the YAML paths to match your Go game package and static web root.

## Configuration

```yaml
project: my-game

game:
  package: ./cmd/game
  output: ./static/game.wasm

wasm:
  exec: ./static/wasm_exec.js

services:
  web:
    root: ./static
    port: 8080

  api:
    enabled: false
    command: go run ./server
    port: 3001

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

For a minimal game, disable or omit API services. For a live-service game, add the local API/realtime/database processes your project needs.

## Project Model

Your project owns the browser app:

```text
static/
  index.html
  wasm_exec.js      # written by ebitdock build wasm
  game.wasm         # written by ebitdock build wasm
  audio/
  assets/
```

Your HTML loads Go WASM:

```html
<script src="./wasm_exec.js"></script>
<script>
  const go = new Go();
  WebAssembly.instantiateStreaming(fetch("./game.wasm"), go.importObject)
    .then((result) => go.run(result.instance));
</script>
```

Browser-specific behavior such as audio unlock, Howler setup, local storage, or JS bridge functions belongs in your project.

## Dev Mode

```sh
ebitdock dev
```

Starts the configured web server, dashboard, optional API command, watcher, and initial WASM build.

```text
SERVICE    STATUS    URL/DETAILS
web        running   http://localhost:8080
dashboard  running   http://localhost:8081
backend    disabled  -
wasm       ok        514ms
watch      active    6 patterns
```

`watch.rebuild` triggers WASM rebuilds. `watch.static` logs static file changes.

## Build

```sh
ebitdock build wasm
```

Runs roughly:

```sh
GOOS=js GOARCH=wasm go build -mod=mod -o ./static/game.wasm ./cmd/game
```

It also copies the matching `wasm_exec.js` from the installed Go toolchain to the configured `wasm.exec` path.

## Next

The next planned dev runner is external `wasmserve`:

```sh
go install github.com/hajimehoshi/wasmserve@latest
```

`ebitdock` will use it as the Ebitengine WASM dev server while keeping responsibility for orchestration, ports, logs, optional services, and dashboard state.

## GitHub Checks

The included GitHub Actions workflow runs formatting, vet, tests, CLI build, and an init smoke test on pull requests and pushes.
