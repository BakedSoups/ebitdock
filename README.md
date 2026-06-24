# ebitdock

I really like Ebitengine. It is a great Go game engine, and its browser target makes it possible to compile Go games into lightweight WebAssembly experiences that run directly in the browser.

The part I found awkward was everything around the game: port management, local service orchestration, and the setup needed for browser games that talk to backends, databases, or multiple APIs. That gets especially annoying for layered `.io`-style games or web games that need more than one local process.

`ebitdock` exists for that layer. It is a lightweight Go-native CLI that builds your Ebitengine game to WASM, serves your existing static web root with the right headers, starts optional local services, watches files, and exposes a compact dashboard for ports, logs, and build status.

It does not require Node.js, Docker, or a generated browser framework. Your project owns its HTML, JS, audio, assets, and browser bridge code.

## Install

```sh
go install ./cmd/ebitdock
```

`ebitdock dev` uses `wasmserve` as the browser game runner:

```sh
go install github.com/hajimehoshi/wasmserve@latest
```

During local development:

```sh
go run ./cmd/ebitdock --help
```

## Quick Start

For an existing Ebitengine project:

```sh
cd /path/to/your-game
ebitdock init
ebitdock dev
```

`init` writes `ebitdock.yaml` if it does not already exist. It does not overwrite or generate your web app.

For a basic project folder:

```sh
ebitdock init my-game
```

This creates:

```text
my-game/
  ebitdock.yaml
```

Add your Go game package and static web root, then edit the YAML paths to match.

Open the URLs printed by `dev`, usually:

```text
web        http://localhost:8080
dashboard  http://localhost:8081
```

## Commands

```sh
ebitdock init [name|.]
ebitdock dev
ebitdock build wasm
ebitdock logs
ebitdock doctor
```

## Project Model

Your app owns the static web root:

```text
static/
  index.html
  wasm_exec.js      # written by ebitdock build wasm
  game.wasm         # written by ebitdock build wasm
  audio/
  assets/
```

Your HTML is responsible for loading Go WASM:

```html
<script src="./wasm_exec.js"></script>
<script>
  const go = new Go();
  WebAssembly.instantiateStreaming(fetch("./game.wasm"), go.importObject)
    .then((result) => go.run(result.instance));
</script>
```

Browser-specific behavior such as audio unlock, Howler setup, local storage, or JS bridge functions belongs in your project, not in `ebitdock`.

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
    - ./levels/**

  static:
    - ./static/**
```

`watch.rebuild` triggers a WASM rebuild during `dev`.

`watch.static` is logged as a static file change. `ebitdock` does not inject browser reload code; refresh or use your own web tooling if needed.

## Dev Mode

`ebitdock dev` starts `wasmserve`, the dashboard server, optional API command, watcher, and project-local logging.

For this config:

```yaml
game:
  package: ./cmd/game

services:
  web:
    port: 8080
```

dev mode runs:

```sh
wasmserve -http :8080 ./cmd/game
```

If `wasmserve` is missing, install it with:

```sh
go install github.com/hajimehoshi/wasmserve@latest
```

Startup output is an aligned table:

```text
SERVICE    STATUS    URL/DETAILS
wasmserve  running   http://localhost:8080
dashboard  running   http://localhost:8081
backend    disabled  -
watch      active    6 patterns
```

Source and static file changes are logged during dev. `wasmserve` owns browser-target rebuilds.

## Build

```sh
ebitdock build wasm
```

Runs roughly:

```sh
GOOS=js GOARCH=wasm go build -mod=mod -o ./static/game.wasm ./cmd/game
```

It also copies the matching `wasm_exec.js` from the installed Go toolchain to the configured `wasm.exec` path.

## Doctor

```sh
ebitdock doctor
```

Checks the local config and toolchain:

```text
CHECK       STATUS    DETAILS
config      ok        ebitdock.yaml
go          ok        go1.24.4 linux/amd64
wasmserve   ok        /home/alex/go/bin/wasmserve
game        ok        ./cmd/game
web         ok        ./static
dashboard   ok        :8081
api         disabled  -
```

## GitHub Checks

The included GitHub Actions workflow runs formatting, vet, tests, CLI build, and an init smoke test on pull requests and pushes.
