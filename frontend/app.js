const API_BASE = "http://localhost:8080";
const WS_URL = "ws://localhost:8080/ws";

let authToken = null;
let ws = null;
let chart = null;

const dailyStats = { info: 0, warning: 0, critical: 0 };
let knownPersons = [];
const cameraCounts = {};

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

// DASHBOARD
async function initDashboard() {
  await loadCameras();
  await loadEvents();
  await loadKnownPersons();
  connectWebSocket();
  setupChart();
}

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
  container.innerHTML = "";
  knownPersons.forEach(person => {
    const div = document.createElement("div");
    div.className = "person-item";
    div.innerHTML = `
      <span>${person.nome || 'Person_' + person.track_id}</span>
      <label>
        <input type="checkbox" ${person.is_critical ? 'checked' : ''} 
               onchange="toggleCritical('${person.id}', this.checked)">
        Critica
      </label>
    `;
    container.appendChild(div);
  });
}

async function toggleCritical(personId, isCritical) {
  try {
    await fetch(`${API_BASE}/api/persons/${personId}`, {
      method: "PATCH",
      headers: { 
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}` 
      },
      body: JSON.stringify({ is_critical: isCritical })
    });
    loadKnownPersons(); // refresh
  } catch (e) {
    console.error(e);
  }
}

// ---------------------------------------------------------------
// RENDERING
// ---------------------------------------------------------------
function renderCameras(cameras) {
  const grid = document.getElementById("camera-grid");
  grid.innerHTML = "";
  document.getElementById("camera-count").textContent = cameras.length;

  cameras.forEach((cam) => {
    cameraCounts[cam.id] = cameraCounts[cam.id] || 0;
    const tile = document.createElement("div");
    tile.className = "camera-tile";
    tile.id = `camera-${cam.id}`;
    tile.innerHTML = `
      <div class="camera-status-row">
        <span class="dot ${cam.stato === "online" ? "online" : ""}"></span>
        <span>${cam.stato.toUpperCase()}</span>
      </div>
      <div class="camera-name">${cam.nome}</div>
      <div class="camera-people">${cameraCounts[cam.id]}</div>
      <div class="camera-meta">
        <span>${cam.posizione || "-"}</span>
        <span>persone</span>
      </div>
    `;
    grid.appendChild(tile);
  });
}

function bumpCameraCount(event) {
  const tile = document.getElementById(`camera-${event.camera_id}`);
  if (!tile) return; // telecamera non ancora nella lista locale, verra' aggiornata al prossimo refresh
  const el = tile.querySelector(".camera-people");
  const current = parseInt(el.textContent, 10) || 0;
  el.textContent = current + 1;
}

function appendLogEntry(event) {
  const log = document.getElementById("event-log");
  const empty = log.querySelector(".log-empty");
  if (empty) empty.remove();

  const entry = document.createElement("div");
  entry.className = "log-entry";
  const time = new Date(event.timestamp).toLocaleTimeString("it-IT");
  const confidencePct = Math.round((event.confidence || 0) * 100);

  entry.innerHTML = `
    <span class="time">${time}</span>
    <span class="severity ${event.severity}">${event.severity.toUpperCase()}</span>
    <span>${describeEvent(event)}</span>
    <span class="conf">${confidencePct}%</span>
  `;
  log.prepend(entry);

  // limita il DOM a 100 righe per evitare rallentamenti su sessioni lunghe
  while (log.children.length > 100) log.removeChild(log.lastChild);
}

function describeEvent(event) {
  switch (event.tipo_evento) {
    case "person_detected":
      return `Persona rilevata${event.camera_nome ? " — " + event.camera_nome : ""}`;
    case "zone_alert":
      return `Accesso non autorizzato${event.zone ? " (" + event.zone + ")" : ""}`;
    case "anomaly":
      return `Anomalia rilevata${event.zone ? " (" + event.zone + ")" : ""}`;
    default:
      return event.tipo_evento;
  }
}

function tallyStat(event) {
  if (dailyStats[event.severity] !== undefined) dailyStats[event.severity]++;
  if (chart) updateChart();
}

function updateStatCards() {
  const total = dailyStats.info + dailyStats.warning + dailyStats.critical;
  document.getElementById("stat-total").textContent = total;
  document.getElementById("stat-warning").textContent = dailyStats.warning;
  document.getElementById("stat-critical").textContent = dailyStats.critical;
}

// ---------------------------------------------------------------
// CHART (distribuzione eventi per severity)
// ---------------------------------------------------------------
function setupChart() {
  const ctx = document.getElementById("events-chart").getContext("2d");
  chart = new Chart(ctx, {
    type: "bar",
    data: {
      labels: ["INFO", "WARNING", "CRITICAL"],
      datasets: [{
        data: [dailyStats.info, dailyStats.warning, dailyStats.critical],
        backgroundColor: ["#39ff88", "#ffb020", "#ff4d4d"],
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
  chart.data.datasets[0].data = [dailyStats.info, dailyStats.warning, dailyStats.critical];
  chart.update();
}

async function loadEvents() {
  try {
    const res = await fetch(`${API_BASE}/api/events?limit=50`, {
      headers: { Authorization: `Bearer ${authToken}` },
    });
    const events = await res.json();
    events.reverse().forEach(appendLogEntry);
    events.forEach(tallyStat);
    updateStatCards();
  } catch (e) {
    console.error("errore caricamento eventi", e);
  }
  loadKnownPersons(); // refresh dopo eventi
}