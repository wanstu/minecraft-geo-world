(() => {
  const file = (location.pathname.split("/").pop() || "generator.html").toLowerCase();
  if (file !== "generator.html") return;

  const chips = Array.from(document.querySelectorAll(".toolchip[data-layer]"));
  const frame = document.querySelector(".map-frame");
  const legend = document.querySelector(".canvas-wrap > .legend");
  const original = frame?.innerHTML || "";

  const layers = {
    overview: {label:"世界概览", src:null},
    plate: {label:"板块分区", src:"generated/plates.png"},
    elevation: {label:"宏观海拔", src:"generated/elevation.png"},
    current: {label:"表层洋流", src:"generated/climate/ocean-current.png"},
    rain: {label:"年降水", src:"generated/climate/precipitation.png"},
  };

  chips.forEach((chip) => {
    chip.onclick = () => {
      chips.forEach((x) => x.classList.remove("active"));
      chip.classList.add("active");

      const layer = layers[chip.dataset.layer];
      if (!frame || !layer) return;

      if (!layer.src) {
        frame.innerHTML = original;
        if (legend) legend.hidden = false;
      } else {
        frame.innerHTML =
          '<img src="' + layer.src + '" alt="' + layer.label +
          '" style="width:100%;height:100%;object-fit:contain;display:block;background:#091016">';
        if (legend) legend.hidden = true;
      }

      const current = document.querySelector("[data-cursor-layer]");
      if (current) current.textContent = layer.label;
    };
  });

  const mapWrap = document.querySelector(".map-frame");
  if (mapWrap) {
    mapWrap.addEventListener("mousemove", (event) => {
      const rect = mapWrap.getBoundingClientRect();
      if (!rect.width || !rect.height) return;

      const rx = Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width));
      const rz = Math.max(0, Math.min(1, (event.clientY - rect.top) / rect.height));

      const worldSize = 131072;
      const x = Math.round((rx - 0.5) * worldSize);
      const z = Math.round((rz - 0.5) * worldSize);
      const latitude = 82 - rz * 164;

      const coord = document.querySelector("[data-cursor-coord]");
      const lat = document.querySelector("[data-cursor-lat]");

      if (coord) {
        coord.textContent =
          (x >= 0 ? "+" : "") + x.toLocaleString() + ", " +
          (z >= 0 ? "+" : "") + z.toLocaleString();
      }
      if (lat) {
        lat.textContent =
          Math.abs(latitude).toFixed(1) + "° " + (latitude >= 0 ? "N" : "S");
      }
    });
  }
})();
