import { DbConnection } from "./spacetimedb-bindings";

type AgarioConfig = {
  enabled?: boolean;
  host?: string;
  database?: string;
};

type PlayerSnapshot = {
  id: string;
  name: string;
  x: number;
  y: number;
  radius: number;
  mass: number;
  color: string;
  alive: boolean;
};

type FoodSnapshot = {
  id: number;
  x: number;
  y: number;
  mass: number;
};

type WindowWithAgario = Window &
  typeof globalThis & {
    SPACETIME_AGARIO_CONFIG?: AgarioConfig;
    SpacetimeAgario?: {
      status(): string;
      snapshot(): string;
      join(name: string): void;
      input(x: number, y: number): void;
      respawn(): void;
    };
  };

const win = window as WindowWithAgario;
const config = win.SPACETIME_AGARIO_CONFIG || {};

const state = {
  conn: undefined as DbConnection | undefined,
  connecting: undefined as Promise<DbConnection> | undefined,
  connected: false,
  status: config.enabled ? "connecting" : "offline fallback",
  identity: "",
  players: new Map<string, PlayerSnapshot>(),
  food: new Map<number, FoodSnapshot>(),
};

win.SpacetimeAgario = {
  status() {
    return state.status;
  },
  snapshot() {
    return JSON.stringify({
      connected: state.connected,
      playerId: state.identity,
      players: Array.from(state.players.values()),
      food: Array.from(state.food.values()),
    });
  },
  join(name: string) {
    if (!config.enabled) {
      state.status = "offline fallback";
      return;
    }
    connect()
      .then((conn) => conn.reducers.joinGame(name || "pilot"))
      .catch((error) => setError(error));
  },
  input(x: number, y: number) {
    if (!config.enabled || !state.connected || !state.conn) {
      return;
    }
    state.conn.reducers.setInput(x, y).catch((error: unknown) => setError(error));
  },
  respawn() {
    if (!config.enabled || !state.connected || !state.conn) {
      return;
    }
    state.conn.reducers.respawn().catch((error: unknown) => setError(error));
  },
};

if (config.enabled) {
  connect().catch((error) => setError(error));
}

async function connect(): Promise<DbConnection> {
  if (state.conn && state.connected) {
    return state.conn;
  }
  if (state.connecting) {
    return state.connecting;
  }
  state.connecting = new Promise<DbConnection>((resolve, reject) => {
    const host = normalizeHost(config.host || "wss://maincloud.spacetimedb.com");
    const database = config.database || "spacetime-agario";
    let settled = false;
    const timeout = window.setTimeout(() => {
      if (settled) {
        return;
      }
      settled = true;
      state.connecting = undefined;
      state.connected = false;
      reject(new Error(`connect timed out: ${host}/${database}`));
    }, 10000);
    try {
      DbConnection.builder()
        .withUri(host)
        .withDatabaseName(database)
        .withCompression("none")
        .onConnect((conn, identity) => {
          if (settled) {
            return;
          }
          settled = true;
          window.clearTimeout(timeout);
          state.conn = conn;
          state.connected = true;
          state.identity = identity.toHexString();
          state.status = "connected";
          subscribe(conn);
          resolve(conn);
        })
        .onConnectError((_ctx, error) => {
          if (settled) {
            return;
          }
          settled = true;
          window.clearTimeout(timeout);
          state.connected = false;
          state.connecting = undefined;
          reject(error);
        })
        .onDisconnect((_ctx, error) => {
          state.connected = false;
          state.connecting = undefined;
          state.status = error ? `disconnected: ${String(error)}` : "disconnected";
        })
        .build();
    } catch (error) {
      if (settled) {
        return;
      }
      settled = true;
      window.clearTimeout(timeout);
      state.connected = false;
      state.connecting = undefined;
      reject(error);
    }
  });
  return state.connecting;
}

function subscribe(conn: DbConnection) {
  conn.db.player.onInsert((_ctx, row) => state.players.set(identityString(row.identity), playerRow(row)));
  conn.db.player.onUpdate((_ctx, _oldRow, row) => state.players.set(identityString(row.identity), playerRow(row)));
  conn.db.player.onDelete((_ctx, row) => state.players.delete(identityString(row.identity)));
  conn.db.food.onInsert((_ctx, row) => state.food.set(Number(row.id), foodRow(row)));
  conn.db.food.onUpdate((_ctx, _oldRow, row) => state.food.set(Number(row.id), foodRow(row)));
  conn.db.food.onDelete((_ctx, row) => state.food.delete(Number(row.id)));
  conn
    .subscriptionBuilder()
    .onApplied(() => {
      state.status = "subscribed";
    })
    .subscribe(["SELECT * FROM player", "SELECT * FROM food"]);
}

function playerRow(row: any): PlayerSnapshot {
  return {
    id: identityString(row.identity),
    name: String(row.name || "pilot"),
    x: Number(row.x || 0),
    y: Number(row.y || 0),
    radius: Number(row.radius || 20),
    mass: Number(row.mass || 20),
    color: String(row.color || "#6dd8c7"),
    alive: Boolean(row.alive),
  };
}

function foodRow(row: any): FoodSnapshot {
  return {
    id: Number(row.id),
    x: Number(row.x || 0),
    y: Number(row.y || 0),
    mass: Number(row.mass || 2),
  };
}

function identityString(value: any): string {
  if (!value) {
    return "";
  }
  if (typeof value === "string") {
    return value;
  }
  if (typeof value.toHexString === "function") {
    return value.toHexString();
  }
  return String(value);
}

function normalizeHost(host: string): string {
  if (host.startsWith("http://")) {
    return "ws://" + host.slice("http://".length);
  }
  if (host.startsWith("https://")) {
    return "wss://" + host.slice("https://".length);
  }
  return host;
}

function setError(error: unknown) {
  state.status = error instanceof Error ? error.message : String(error);
  console.error("[spacetime-agario]", error);
}
