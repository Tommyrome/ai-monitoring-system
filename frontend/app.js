const API_BASE = "http://localhost:8080";
const WS_URL = "ws://localhost:8080/ws";
const CAMERA_CODE = "camera_01";

let authToken = null;
let ws = null;
let wsReconnectTimer = null;
let chart = null;

let stats = { total_events: 0, normal_events: 0, critical_events: 0 };
let knownPersons = [];

// AUTH
document.getElementById("login-btn").addEventListener("click", login);

async function login() {
  const username = document.getElementById("username").value;
  const password = document.getElementById("password").value;
  try {
    const res = await fetch(`${API_BASE}/api/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    if (!res.ok) throw new Error("credenziali non valide");
    const data = await res.json();
    authToken = data.token;
    document.getElementById("login-box").style.display = "none";
    initDashboard();
  } catch (e) {
    alert("Login fallito: " + e.message);
  }
}

// ---------------------------------------------------------------
// DASHBOARD INIT
// ---------------------------------------------------------------
async function initDashboard() {
  setupChart();
  await loadStats();
  await loadEvents();
  await loadKnownPersons();
  await loadInitialFrame();
  connectWebSocket();
}

// ---------------------------------------------------------------
// WEBSOCKET (aggiornamenti in tempo reale: eventi, statistiche, frame)
// ---------------------------------------------------------------
function connectWebSocket() {
  ws = new WebSocket(WS_URL);

  ws.onopen = () => {
    setConnLabel(true);
    if (wsReconnectTimer) {
      clearTimeout(wsReconnectTimer);
      wsReconnectTimer = null;
    }
  };

  ws.onmessage = (msg) => {
    let payload;
    try {
      payload = JSON.parse(msg.data);
    } catch (e) {
      return;
    }
    handleWsMessage(payload);
  };

  ws.onclose = () => {
    setConnLabel(false);
    // riconnessione automatica
    wsReconnectTimer = setTimeout(connectWebSocket, 2000);
  };

  ws.onerror = () => {
    ws.close();
  };
}

function setConnLabel(online) {
  const dot = document.getElementById("conn-dot");
  const label = document.getElementById("conn-label");
  dot.className = online ? "dot online" : "dot";
  label.textContent = online ? "LIVE" : "RICONNESSIONE...";
}

function handleWsMessage(payload) {
  switch (payload.type) {
    case "new_event":
      onNewEvent(payload.event);
      break;
    case "stats_update":
      stats = payload.stats;
      updateStatCards();
      updateChart();
      break;
    case "frame":
      if (payload.camera === CAMERA_CODE) renderFrame(payload.data);
      break;
    default:
      break;
  }
}

function onNewEvent(event) {
  appendLogEntry(event);
  setCameraOnline(true);
  loadKnownPersons(); // aggiorna last_seen / eventuali nuove persone
}

// ---------------------------------------------------------------
// FEED TELECAMERA
// ---------------------------------------------------------------
async function loadInitialFrame() {
  try {
    const res = await fetch(`${API_BASE}/api/frame?camera=${CAMERA_CODE}`, {
      headers: { Authorization: `Bearer ${authToken}` },
    });
    if (!res.ok) return; // nessun frame disponibile ancora, va bene
    const data = await res.json();
    renderFrame(data.data);
  } catch (e) {
    console.error("errore caricamento frame iniziale", e);
  }
}

function renderFrame(base64Jpeg) {
  const img = document.getElementById("camera-feed");
  const placeholder = document.getElementById("feed-placeholder");
  img.src = `data:image/jpeg;base64,${base64Jpeg}`;
  img.style.display = "block";
  placeholder.style.display = "none";
  setCameraOnline(true);
}

function setCameraOnline(online) {
  const dot = document.getElementById("camera-dot");
  const label = document.getElementById("camera-status-label");
  dot.className = online ? "dot online" : "dot";
  label.textContent = online ? "ONLINE" : "OFFLINE";
}

// ---------------------------------------------------------------
// PERSONE RILEVATE
// ---------------------------------------------------------------
async function loadKnownPersons() {
  try {
    const res = await fetch(`${API_BASE}/api/persons`, {
      headers: { Authorization: `Bearer ${authToken}` },
    });
    knownPersons = await res.json();
    renderKnownPersons();
  } catch (e) {
    console.error("Errore caricamento persons", e);
  }
}

function renderKnownPersons() {
  const container = document.getElementById("persons-list");
  document.getElementById("persons-count").textContent = knownPersons.length;

  if (knownPersons.length === 0) {
    container.innerHTML = `<div class="log-empty">Nessuna persona rilevata finora...</div>`;
    return;
  }

  container.innerHTML = "";
  knownPersons.forEach((person) => {
    const div = document.createElement("div");
    div.className = "person-item";
    div.innerHTML = `
      <div class="person-row">
        <input type="text" class="person-name-input" value="${escapeHtml(person.nome)}" data-id="${person.id}">
        <button class="rename-btn" data-id="${person.id}">Salva</button>
      </div>
      <label class="critical-toggle">
        <input type="checkbox" ${person.is_critical ? "checked" : ""}
               onchange="toggleCritical('${person.id}', this.checked)">
        Persona critica
      </label>
    `;
    container.appendChild(div);
  });

  container.querySelectorAll(".rename-btn").forEach((btn) => {
    btn.addEventListener("click", () => {
      const id = btn.dataset.id;
      const input = container.querySelector(`.person-name-input[data-id="${id}"]`);
      renamePerson(id, input.value);
    });
  });
}

function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str || "";
  return div.innerHTML;
}

async function renamePerson(personId, nome) {
  nome = (nome || "").trim();
  if (!nome) return;
  try {
    await fetch(`${API_BASE}/api/persons/${personId}`, {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}`,
      },
      body: JSON.stringify({ nome }),
    });
    loadKnownPersons();
  } catch (e) {
    console.error(e);
  }
}

async function toggleCritical(personId, isCritical) {
  try {
    await fetch(`${API_BASE}/api/persons/${personId}`, {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}`,
      },
      body: JSON.stringify({ is_critical: isCritical }),
    });
    loadKnownPersons();
  } catch (e) {
    console.error(e);
  }
}

// ---------------------------------------------------------------
// EVENT LOG
// ---------------------------------------------------------------
function appendLogEntry(event) {
  const log = document.getElementById("event-log");
  const empty = log.querySelector(".log-empty");
  if (empty) empty.remove();

  const entry = document.createElement("div");
  entry.className = "log-entry";
  const time = new Date(event.timestamp).toLocaleTimeString("it-IT");
  const confidencePct = Math.round((event.confidence || 0) * 100);
  const tipo = (event.tipo_evento || "NORMAL").toUpperCase();

  entry.innerHTML = `
    <span class="time">${time}</span>
    <span class="tipo ${tipo.toLowerCase()}">${tipo}</span>
    <span>${describeEvent(event)}</span>
    <span class="conf">${confidencePct}%</span>
  `;
  log.prepend(entry);

  // limita il DOM a 100 righe per evitare rallentamenti su sessioni lunghe
  while (log.children.length > 100) log.removeChild(log.lastChild);
}

function describeEvent(event) {
  const nome = event.person_nome || (event.track_id ? `Persona ${event.track_id}` : "Persona");
  if (event.tipo_evento === "CRITICAL") {
    return `⚠ Rilevata persona critica: ${nome}`;
  }
  return `Persona rilevata: ${nome}`;
}

// ---------------------------------------------------------------
// STATISTICHE
// ---------------------------------------------------------------
async function loadStats() {
  try {
    const res = await fetch(`${API_BASE}/api/stats`, {
      headers: { Authorization: `Bearer ${authToken}` },
    });
    stats = await res.json();
    updateStatCards();
    updateChart();
  } catch (e) {
    console.error("errore caricamento statistiche", e);
  }
}

function updateStatCards() {
  document.getElementById("stat-total").textContent = stats.total_events;
  document.getElementById("stat-normal").textContent = stats.normal_events;
  document.getElementById("stat-critical").textContent = stats.critical_events;
}

// ---------------------------------------------------------------
// CHART (distribuzione eventi normali / critici)
// ---------------------------------------------------------------
function setupChart() {
  const ctx = document.getElementById("events-chart").getContext("2d");
  chart = new Chart(ctx, {
    type: "bar",
    data: {
      labels: ["NORMALI", "CRITICI"],
      datasets: [{
        data: [stats.normal_events, stats.critical_events],
        backgroundColor: ["#39ff88", "#ff4d4d"],
        borderRadius: 2,
      }],
    },
    options: {
      responsive: true,
      plugins: { legend: { display: false } },
      scales: {
        x: { ticks: { color: "#6f8a70", font: { family: "IBM Plex Mono", size: 10 } }, grid: { color: "#1e2a1e" } },
        y: { ticks: { color: "#6f8a70", font: { family: "IBM Plex Mono", size: 10 } }, grid: { color: "#1e2a1e" }, beginAtZero: true },
      },
    },
  });
}

function updateChart() {
  if (!chart) return;
  chart.data.datasets[0].data = [stats.normal_events, stats.critical_events];
  chart.update();
}

async function loadEvents() {
  try {
    const res = await fetch(`${API_BASE}/api/events?limit=50`, {
      headers: { Authorization: `Bearer ${authToken}` },
    });
    const events = await res.json();
    events.reverse().forEach(appendLogEntry);
    if (events.length > 0) setCameraOnline(true);
  } catch (e) {
    console.error("errore caricamento eventi", e);
  }
}
