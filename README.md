# ebitdock

`ebitdock` is a lightweight Go-native CLI for Ebitengine WebAssembly development. It builds your game to WASM, serves your existing static web root with the right headers, starts optional local services, watches files, and exposes a compact dashboard for ports, logs, and build status.

It does not require Node.js, Docker, or a generated browser framework. Your project owns its HTML, JS, audio, assets, and browser bridge code.

## Install

```sh
go install ./cmd/ebitdock
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
ebitdock export web
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

`ebitdock dev` starts the configured static server, dashboard server, optional API command, watcher, and initial WASM build.

Startup output is an aligned table:

```text
SERVICE    STATUS    URL/DETAILS
web        running   http://localhost:8080
dashboard  running   http://localhost:8081
backend    disabled  -
wasm       ok        514ms
watch      active    6 patterns
```

## Build

```sh
ebitdock build wasm
```

Runs roughly:

```sh
GOOS=js GOARCH=wasm go build -mod=mod -o ./static/game.wasm ./cmd/game
```

It also copies the matching `wasm_exec.js` from the installed Go toolchain to the configured `wasm.exec` path.

## Export

```sh
ebitdock export web
```

Builds WASM and copies the configured static root into:

```text
dist/
```

## GitHub Checks

The included GitHub Actions workflow runs formatting, vet, tests, CLI build, and an init smoke test on pull requests and pushes.
