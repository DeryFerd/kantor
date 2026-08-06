// Cross-browser: Firefox/Safari expose `browser` (promise-based); Chrome exposes
// `chrome` (promise-based on MV3 for the APIs used here). One namespace for all.
const browserApi = globalThis.browser ?? globalThis.chrome;

const DEFAULT_STATE = {
  apiBaseUrl: "",
  dashboardUrl: "",
  token: "",
  sessionId: "",
  consented: false,
  paused: false,
  idleTimeoutSeconds: 7200,
  excludedDomains: [],
  queuedEntries: [],
  currentTab: null,
  trackerState: "stopped",
  lastSummary: null,
  lastHeartbeatAt: null,
  lastError: "",
  updateAvailable: false,
  latestVersion: "",
};

const HEARTBEAT_ALARM = "kantor-heartbeat";
const HEARTBEAT_INTERVAL_MINUTES = 0.5;
// Offline queue cap. Sized to hold a full workday of 30s heartbeats (~2880/day)
// with headroom, so a long offline stretch does not silently drop early activity.
const MAX_QUEUED_ENTRIES = 5000;

browserApi.runtime.onInstalled.addListener((details) => {
  void initializeState();
  // On update, the KANTOR dashboard tab still runs the OLD content script (dead
  // runtime). Reload those tabs so a fresh content script loads and the user
  // does not have to refresh manually or hit "context invalidated".
  if (details?.reason === "update") {
    void reloadDashboardTabs();
  }
});

browserApi.runtime.onStartup.addListener(() => {
  void initializeState();
});

browserApi.runtime.onSuspend?.addListener(() => {
  void bestEffortEndSession();
});

browserApi.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === HEARTBEAT_ALARM) {
    void handleHeartbeatTick();
  }
});

browserApi.tabs.onActivated.addListener(() => {
  void updateCurrentTabSnapshot();
});

browserApi.tabs.onUpdated.addListener((_tabId, changeInfo, tab) => {
  if (changeInfo.status === "complete" && tab.active) {
    void updateCurrentTabSnapshot();
  }
});

browserApi.windows.onFocusChanged.addListener(() => {
  void updateCurrentTabSnapshot();
});

browserApi.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  void (async () => {
    try {
      switch (message?.type) {
        case "tracker:get-state":
          sendResponse(await getExtensionState());
          break;
        case "tracker:save-config":
          await updateState({
            apiBaseUrl: sanitizeApiBaseUrl(message.payload.apiBaseUrl),
            dashboardUrl: sanitizeDashboardUrl(message.payload.dashboardUrl),
            token: String(message.payload.token || "").trim(),
          });
          await refreshConsent();
          await fetchTodaySummary();
          await checkForUpdate();
          sendResponse({ ok: true });
          break;
        case "tracker:set-options":
          await updateState({
            idleTimeoutSeconds: normalizeIdleTimeout(message.payload.idleTimeoutSeconds),
            excludedDomains: normalizeExcludedDomains(message.payload.excludedDomains),
          });
          sendResponse({ ok: true });
          break;
        case "tracker:grant-consent":
          sendResponse(await grantConsent());
          break;
        case "tracker:revoke-consent":
          sendResponse(await revokeConsent());
          break;
        case "tracker:pause":
          await updateState({ paused: true, trackerState: "paused" });
          sendResponse({ ok: true });
          break;
        case "tracker:resume":
          await updateState({ paused: false });
          await ensureActiveSession();
          sendResponse({ ok: true });
          break;
        case "tracker:stop":
          await stopTracking({ revokeConsent: true });
          sendResponse({ ok: true });
          break;
        case "tracker:refresh":
          await refreshConsent();
          await fetchTodaySummary();
          await checkForUpdate();
          sendResponse(await getExtensionState());
          break;
        default:
          sendResponse({ ok: false, error: "Unsupported action" });
      }
    } catch (error) {
      const messageText = error instanceof Error ? error.message : "Unexpected extension error";
      await updateState({ lastError: messageText });
      sendResponse({ ok: false, error: messageText });
    }
  })();

  return true;
});

async function initializeState() {
  const state = await loadState();
  await saveState({ ...DEFAULT_STATE, ...state });
  await browserApi.alarms.clear(HEARTBEAT_ALARM);
  await browserApi.alarms.create(HEARTBEAT_ALARM, {
    delayInMinutes: HEARTBEAT_INTERVAL_MINUTES,
    periodInMinutes: HEARTBEAT_INTERVAL_MINUTES,
  });
  await updateCurrentTabSnapshot();
  await refreshConsent();
  await ensureActiveSession();
  await checkForUpdate();
}

async function getExtensionState() {
  const state = await loadState();
  return {
    ...state,
    dashboardUrl: resolveDashboardUrl(state),
    installedVersion: getExtensionVersion(),
  };
}

async function handleHeartbeatTick() {
  const state = await loadState();
  if (!state.apiBaseUrl || !state.token || state.paused) {
    await updateState({ trackerState: state.paused ? "paused" : "stopped" });
    return;
  }

  const tabInfo = await getCurrentTabInfo(state.excludedDomains);
  if (!tabInfo) {
    await updateState({ currentTab: null, trackerState: "stopped" });
    return;
  }

  const idleState = await queryIdleState(state.idleTimeoutSeconds);
  const payload = {
    session_id: state.sessionId,
    url: tabInfo.url,
    domain: tabInfo.domain,
    page_title: tabInfo.title || tabInfo.domain,
    is_idle: idleState !== "active",
    timestamp: new Date().toISOString(),
    timezone_offset_minutes: getTimezoneOffsetMinutes(),
    timezone_name: getTimezoneName(),
    extension_version: getExtensionVersion(),
  };

  await updateState({
    currentTab: {
      ...tabInfo,
      idleState,
      category: state.lastSummary?.top_domains?.find((item) => item.domain === tabInfo.domain)?.category || "uncategorized",
    },
    trackerState: idleState === "active" ? "active" : "idle",
  });

  await flushQueue();
  await sendHeartbeat(payload);
  await fetchTodaySummary();
}

async function sendHeartbeat(payload) {
  const state = await loadState();

  let sessionId = state.sessionId;
  if (!sessionId) {
    sessionId = await ensureActiveSession();
    payload.session_id = sessionId;
  }
  // Without a session we cannot attribute the beat; queue it for a later flush
  // (which re-stamps it with a valid session) instead of sending an empty id.
  if (!sessionId) {
    await queueEntry(payload);
    return;
  }

  try {
    const response = await authorizedRequest("/tracker/heartbeat", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    if (response?.data?.session?.id) {
      await updateState({
        sessionId: response.data.session.id,
        trackerState: payload.is_idle ? "idle" : "active",
        lastHeartbeatAt: new Date().toISOString(),
        lastError: "",
      });
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : "Failed to send heartbeat";
    if (message.includes("TRACKER_SESSION_NOT_FOUND")) {
      const nextSessionId = await ensureActiveSession(true);
      if (nextSessionId) {
        await sendHeartbeat({ ...payload, session_id: nextSessionId });
        return;
      }
    }
    if (message.includes("CONSENT_REQUIRED")) {
      await updateState({ consented: false, trackerState: "stopped", lastError: "Consent required" });
      return;
    }
    // Network error / offline / transient failure: keep the beat for later.
    await queueEntry(payload);
    await updateState({ lastError: message });
  }
}

async function flushQueue() {
  const state = await loadState();
  const pending = state.queuedEntries;
  if (!pending.length) {
    return;
  }

  // Re-attach the current session so entries queued under an old/empty session
  // (e.g. one closed while offline) still land instead of being skipped server-side.
  const sessionId = await ensureActiveSession();
  if (!sessionId) {
    return;
  }
  const entries = pending.map((entry) => ({ ...entry, session_id: sessionId }));

  try {
    await authorizedRequest("/tracker/entries/batch", {
      method: "POST",
      body: JSON.stringify({ entries }),
    });
    // Drop only the entries we actually sent (the oldest `sentCount`) inside the
    // lock, so beats queued while this batch was in flight are not lost.
    const sentCount = entries.length;
    await withStateLock(async () => {
      const current = await loadState();
      await saveState({ ...current, queuedEntries: current.queuedEntries.slice(sentCount) });
    });
  } catch (error) {
    await updateState({
      lastError: error instanceof Error ? error.message : "Failed to sync queued entries",
    });
  }
}

async function ensureActiveSession(forceRestart = false) {
  const state = await loadState();
  if (!state.consented || state.paused || !state.token) {
    return "";
  }
  if (state.sessionId && !forceRestart) {
    return state.sessionId;
  }

  try {
    const response = await authorizedRequest("/tracker/sessions/start", {
      method: "POST",
      body: JSON.stringify({
        timestamp: new Date().toISOString(),
        timezone_offset_minutes: getTimezoneOffsetMinutes(),
        timezone_name: getTimezoneName(),
        extension_version: getExtensionVersion(),
      }),
    });
    const sessionId = response?.data?.session_id || "";
    await updateState({
      sessionId,
      trackerState: "active",
      lastError: "",
    });
    return sessionId;
  } catch (error) {
    await updateState({
      trackerState: "stopped",
      lastError: error instanceof Error ? error.message : "Failed to start tracker session",
    });
    return "";
  }
}

async function stopTracking(options = {}) {
  const revokeConsentOnStop = Boolean(options?.revokeConsent);
  const state = await loadState();
  if (state.sessionId) {
    try {
      await authorizedRequest(`/tracker/sessions/${state.sessionId}/end`, {
        method: "PATCH",
        body: JSON.stringify({ timestamp: new Date().toISOString() }),
      });
    } catch {
      // Best-effort shutdown.
    }
  }

  if (revokeConsentOnStop && state.consented) {
    try {
      await authorizedRequest("/tracker/consent", {
        method: "DELETE",
        body: JSON.stringify({}),
      });
    } catch (error) {
      await updateState({
        lastError: error instanceof Error ? error.message : "Failed to revoke tracker consent",
      });
      throw error;
    }
  }

  await updateState({
    consented: revokeConsentOnStop ? false : state.consented,
    sessionId: "",
    paused: true,
    trackerState: "stopped",
  });
}

async function bestEffortEndSession() {
  const state = await loadState();
  if (!state.sessionId || !state.token) {
    return;
  }
  try {
    await authorizedRequest(`/tracker/sessions/${state.sessionId}/end`, {
      method: "PATCH",
      body: JSON.stringify({ timestamp: new Date().toISOString() }),
    });
  } catch {
    // Ignore suspend race conditions.
  }
}

async function grantConsent() {
  const response = await authorizedRequest("/tracker/consent", {
    method: "POST",
    body: JSON.stringify({}),
  });
  await updateState({ consented: true, paused: false, lastError: "" });
  await ensureActiveSession(true);
  return response;
}

async function revokeConsent() {
  await authorizedRequest("/tracker/consent", {
    method: "DELETE",
    body: JSON.stringify({}),
  });
  await updateState({
    consented: false,
    paused: true,
    sessionId: "",
    trackerState: "stopped",
  });
  return { ok: true };
}

async function refreshConsent() {
  const state = await loadState();
  if (!state.apiBaseUrl || !state.token) {
    return;
  }
  try {
    const response = await authorizedRequest("/tracker/consent", { method: "GET" });
    await updateState({
      consented: Boolean(response?.data?.consented),
      lastError: "",
    });
  } catch (error) {
    await updateState({
      consented: false,
      lastError: error instanceof Error ? error.message : "Failed to read consent",
    });
  }
}

async function fetchTodaySummary() {
  const state = await loadState();
  if (!state.apiBaseUrl || !state.token || !state.consented) {
    return;
  }
  const date = formatLocalDate();
  try {
    const summary = await authorizedRequest(`/tracker/my-activity?date_from=${date}&date_to=${date}`, {
      method: "GET",
    });
    await updateState({
      lastSummary: summary.data,
      lastError: "",
    });
  } catch (error) {
    await updateState({
      lastError: error instanceof Error ? error.message : "Failed to fetch activity summary",
    });
  }
}

// checkForUpdate asks the backend for the latest published extension version and
// flags the UI when the installed build is behind, so the user knows to
// re-download/reinstall.
async function checkForUpdate() {
  const state = await loadState();
  if (!state.apiBaseUrl || !state.token) {
    return;
  }
  try {
    const response = await authorizedRequest("/tracker/extension/status", { method: "GET" });
    const latest = String(response?.data?.latest_version || "").trim();
    if (!latest) {
      return;
    }
    await updateState({
      latestVersion: latest,
      updateAvailable: isVersionOlder(getExtensionVersion(), latest),
    });
  } catch {
    // Non-fatal: version check never blocks tracking.
  }
}

async function getCurrentTabInfo(excludedDomains) {
  const [tab] = await browserApi.tabs.query({ active: true, lastFocusedWindow: true });
  if (!tab?.url || !tab.active) {
    return null;
  }

  if (!isTrackableUrl(tab.url)) {
    return null;
  }

  const url = new URL(tab.url);
  const domain = url.hostname.toLowerCase();
  if (excludedDomains.includes(domain)) {
    return null;
  }

  return {
    url: tab.url,
    domain,
    title: tab.title || domain,
  };
}

async function updateCurrentTabSnapshot() {
  const state = await loadState();
  const tab = await getCurrentTabInfo(state.excludedDomains);
  await updateState({ currentTab: tab });
}

async function queueEntry(entry) {
  await withStateLock(async () => {
    const state = await loadState();
    const nextQueue = [...state.queuedEntries, entry];
    if (nextQueue.length > MAX_QUEUED_ENTRIES) {
      console.warn("KANTOR tracker: offline queue full, dropping oldest entries");
      nextQueue.splice(0, nextQueue.length - MAX_QUEUED_ENTRIES);
    }
    await saveState({ ...state, queuedEntries: nextQueue });
  });
}

async function queryIdleState(idleTimeoutSeconds) {
  return browserApi.idle.queryState(normalizeIdleTimeout(idleTimeoutSeconds));
}

function isTrackableUrl(rawUrl) {
  return !/^chrome:|^chrome-extension:|^moz-extension:|^about:|^edge:|^file:/i.test(rawUrl);
}

function sanitizeApiBaseUrl(value) {
  return String(value || "").trim().replace(/\/+$/, "");
}

function sanitizeDashboardUrl(value) {
  return String(value || "").trim();
}

function normalizeExcludedDomains(items) {
  return Array.from(
    new Set(
      (Array.isArray(items) ? items : [])
        .map((item) => String(item || "").trim().toLowerCase())
        .filter(Boolean),
    ),
  );
}

// Default idle threshold is 2 hours: a user is only counted idle after 2h with
// no keyboard/mouse input. Configurable per user in the options page (min 60s).
function normalizeIdleTimeout(value) {
  const parsed = Number(value || 7200);
  if (!Number.isFinite(parsed) || parsed < 60) {
    return 7200;
  }
  return Math.round(parsed);
}

function getTimezoneOffsetMinutes() {
  return new Date().getTimezoneOffset();
}

function getTimezoneName() {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "";
  } catch {
    return "";
  }
}

function getExtensionVersion() {
  try {
    return browserApi.runtime.getManifest()?.version || "";
  } catch {
    return "";
  }
}

// isVersionOlder compares dotted numeric versions (e.g. "1.9" < "1.10").
function isVersionOlder(current, latest) {
  const toParts = (value) => String(value || "").split(".").map((part) => Number.parseInt(part, 10) || 0);
  const a = toParts(current);
  const b = toParts(latest);
  const length = Math.max(a.length, b.length);
  for (let i = 0; i < length; i += 1) {
    const left = a[i] || 0;
    const right = b[i] || 0;
    if (left < right) {
      return true;
    }
    if (left > right) {
      return false;
    }
  }
  return false;
}

function formatLocalDate(value = new Date()) {
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, "0");
  const day = String(value.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function toDashboardUrl(apiBaseUrl) {
  const value = sanitizeApiBaseUrl(apiBaseUrl);
  if (!value) {
    return "";
  }
  if (value.endsWith("/api/v1")) {
    return `${value.slice(0, -7)}/operational/tracker`;
  }
  return value.replace(/\/$/, "");
}

function resolveDashboardUrl(state) {
  const explicit = sanitizeDashboardUrl(state.dashboardUrl);
  if (explicit) {
    return explicit;
  }
  return toDashboardUrl(state.apiBaseUrl);
}

// reloadDashboardTabs refreshes the tabs on the configured dashboard origin so
// that, after an extension update, the orphaned content script is replaced by a
// fresh one instead of throwing "Extension context invalidated".
async function reloadDashboardTabs() {
  const state = await loadState();
  const origin = dashboardOrigin(resolveDashboardUrl(state)) || dashboardOrigin(state.apiBaseUrl);
  if (!origin) {
    return;
  }
  try {
    const tabs = await browserApi.tabs.query({ url: `${origin}/*` });
    for (const tab of tabs) {
      if (tab.id != null) {
        await browserApi.tabs.reload(tab.id);
      }
    }
  } catch {
    // Best-effort: missing permission or a closed tab must never break startup.
  }
}

function dashboardOrigin(value) {
  try {
    return new URL(String(value || "").trim()).origin;
  } catch {
    return "";
  }
}

async function authorizedRequest(path, init) {
  const state = await loadState();
  if (!state.apiBaseUrl || !state.token) {
    throw new Error("Extension belum terhubung. Hubungkan dari dashboard KANTOR atau gunakan setup manual.");
  }

  const response = await fetch(`${sanitizeApiBaseUrl(state.apiBaseUrl)}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${state.token}`,
      ...(init?.headers || {}),
    },
  });

  const payload = await response.json().catch(() => null);

  if (response.status === 401) {
    // The extension authenticates with a long-lived Personal Access Token; a 401
    // means it was revoked or is invalid. There is nothing to refresh — prompt the
    // user to reconnect from the dashboard.
    await updateState({
      lastError: "Sesi tracker berakhir. Hubungkan ulang dari dashboard KANTOR.",
    });
    throw new Error("UNAUTHORIZED: reconnect required");
  }

  if (!response.ok || !payload?.success) {
    const code = payload?.error?.code || `HTTP_${response.status}`;
    const message = payload?.error?.message || "Request failed";
    throw new Error(`${code}: ${message}`);
  }

  return payload;
}

async function loadState() {
  const state = await browserApi.storage.local.get(DEFAULT_STATE);
  return { ...DEFAULT_STATE, ...state };
}

async function saveState(nextState) {
  await browserApi.storage.local.set(nextState);
}

// All state mutations run through a single promise chain so concurrent
// read-modify-write calls (heartbeat tick, message handlers) cannot clobber each
// other (e.g. lose queued entries or the pause flag).
let stateLock = Promise.resolve();

function withStateLock(fn) {
  const run = stateLock.then(fn, fn);
  stateLock = run.then(
    () => undefined,
    () => undefined,
  );
  return run;
}

async function updateState(partial) {
  await withStateLock(async () => {
    const current = await loadState();
    await saveState({ ...current, ...partial });
  });
}
