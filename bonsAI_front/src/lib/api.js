import { createDemoSensorsSnapshot } from "./demo.js";

const runtimeConfig = window.__BONSAI_CONFIG__ || {};
const apiBase = (import.meta.env.VITE_API_BASE || runtimeConfig.apiBase || "").replace(
  /\/$/,
  ""
);

function toNumber(value, fallback) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function normalizeSensors(payload = {}) {
  const fallbackSensors = createDemoSensorsSnapshot();

  return {
    temperature: toNumber(
      payload.temperature ?? payload.temp_c ?? payload.temp,
      fallbackSensors.temperature
    ),
    humidity: toNumber(payload.humidity ?? payload.humidity_percent, fallbackSensors.humidity),
    soilMoisture: toNumber(
      payload.soilMoisture ?? payload.soil_moisture ?? payload.moisture,
      fallbackSensors.soilMoisture
    ),
    illuminance: toNumber(
      payload.illuminance ?? payload.light_lux ?? payload.lux,
      fallbackSensors.illuminance
    ),
    lastUpdated: payload.lastUpdated ?? payload.timestamp ?? fallbackSensors.lastUpdated,
    source: payload.source ?? "live"
  };
}

export async function fetchSensors(signal) {
  try {
    const response = await fetch(`${apiBase}/api/sensors`, {
      headers: { Accept: "application/json" },
      signal
    });

    if (!response.ok) {
      throw new Error(`Sensor API returned ${response.status}`);
    }

    const payload = await response.json();
    return normalizeSensors(payload);
  } catch (_error) {
    return createDemoSensorsSnapshot();
  }
}

export async function fetchSystemStatus(signal) {
  const response = await fetch(`${apiBase}/api/system/status`, {
    headers: { Accept: "application/json" },
    signal
  });

  if (!response.ok) {
    throw new Error(`System API returned ${response.status}`);
  }

  const payload = await response.json();
  return {
    llmConnected: payload.llmConnected === true
  };
}

function extractDelta(payload) {
  if (payload == null) return "";
  if (typeof payload === "string") return payload;
  return (
    payload.delta ??
    payload.token ??
    payload.content ??
    payload.message ??
    payload.text ??
    ""
  );
}

async function consumeTextStream(stream, onDelta) {
  const reader = stream.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    onDelta(decoder.decode(value, { stream: true }));
  }
}

function handleSseEvent(rawEvent, onDelta) {
  let eventType = "message";
  const dataLines = [];

  for (const line of rawEvent.split(/\r?\n/)) {
    if (line.startsWith("event:")) {
      eventType = line.slice(6).trim();
    } else if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).trimStart());
    }
  }

  const data = dataLines.join("\n").trim();
  if (!data) return false;
  if (data === "[DONE]" || eventType === "done") return true;

  try {
    const parsed = JSON.parse(data);
    const delta = extractDelta(parsed);
    if (delta) onDelta(delta);
    return parsed.done === true;
  } catch (_error) {
    onDelta(data);
    return false;
  }
}

async function consumeSseStream(stream, onDelta) {
  const reader = stream.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value || new Uint8Array(), { stream: !done });

    let boundary = buffer.indexOf("\n\n");
    while (boundary !== -1) {
      const eventBlock = buffer.slice(0, boundary);
      buffer = buffer.slice(boundary + 2);

      if (handleSseEvent(eventBlock, onDelta)) {
        return;
      }

      boundary = buffer.indexOf("\n\n");
    }

    if (done) {
      if (buffer.trim()) {
        handleSseEvent(buffer, onDelta);
      }
      break;
    }
  }
}

export async function streamChat({ message, history, sensors, onDelta }) {
  const response = await fetch(`${apiBase}/api/chat/stream`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: "text/event-stream, text/plain, application/json"
    },
    body: JSON.stringify({
      message,
      history,
      sensors
    })
  });

  if (!response.ok || !response.body) {
    throw new Error(`Chat API returned ${response.status}`);
  }

  const contentType = response.headers.get("content-type") || "";

  if (contentType.includes("text/event-stream")) {
    await consumeSseStream(response.body, onDelta);
  } else {
    await consumeTextStream(response.body, onDelta);
  }

  return { source: "live" };
}

export async function translateChat({ messages, targetLanguage }) {
  const response = await fetch(`${apiBase}/api/chat/translate`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json"
    },
    body: JSON.stringify({
      messages,
      targetLanguage
    })
  });

  if (!response.ok) {
    throw new Error(`Translate API returned ${response.status}`);
  }

  const payload = await response.json();
  if (!Array.isArray(payload.translations)) {
    return [];
  }

  return payload.translations.map((translation) => ({
    id: translation.id,
    content: translation.content ?? ""
  }));
}
