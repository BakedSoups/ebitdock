(function () {
  const config = window.SPACETIME_AGARIO_CONFIG || {};
  const state = {
    enabled: Boolean(config.enabled),
    status: config.enabled ? "bridge ready" : "offline fallback",
    connected: false,
    playerId: "local",
    players: [],
    food: [],
  };

  window.SpacetimeAgario = {
    status() {
      return state.status;
    },
    snapshot() {
      return JSON.stringify({
        connected: state.connected,
        playerId: state.playerId,
        players: state.players,
        food: state.food,
      });
    },
    join(name) {
      state.status = state.enabled ? "bindings not built" : "offline fallback";
      console.log("[spacetime-agario] join", name, state.status);
    },
    input(x, y) {
      state.targetX = x;
      state.targetY = y;
    },
    respawn() {
      console.log("[spacetime-agario] respawn");
    },
  };
})();
