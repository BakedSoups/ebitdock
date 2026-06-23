(function () {
  let audioContext;

  async function unlockAudio() {
    if (!audioContext) {
      audioContext = new (window.AudioContext || window.webkitAudioContext)();
    }
    if (audioContext.state === "suspended") {
      await audioContext.resume();
    }
  }

  window.EbitDock = {
    ready() {},
    log(message) {
      console.log("[game]", message);
    },
    save(key, value) {
      localStorage.setItem("ebitdock:" + key, value);
    },
    load(key) {
      return localStorage.getItem("ebitdock:" + key) || "";
    },
    playSound(path, volume) {
      unlockAudio().then(() => {
        const audio = new Audio(path);
        audio.volume = Math.max(0, Math.min(1, Number(volume) || 1));
        audio.play();
      });
    },
    setCanvasSize(width, height) {
      document.documentElement.style.setProperty("--canvas-width", width + "px");
      document.documentElement.style.setProperty("--canvas-height", height + "px");
    },
    submitScore(score) {
      return fetch("/api/score", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ score })
      }).catch(() => {});
    }
  };

  window.addEventListener("pointerdown", unlockAudio, { once: true });
  window.addEventListener("keydown", unlockAudio, { once: true });
})();
