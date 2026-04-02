import { useEffect, useMemo, useRef, useState } from "preact/hooks";
import { fetchSensors, streamChat } from "./lib/api.js";
import { demoSensors } from "./lib/demo.js";

let messageSequence = 0;

const quickPrompts = [
  "今日の様子を教えて",
  "水やりは必要？",
  "光は足りていそう？"
];

const plantProfile = { name: "言葉", species: "パキラ", season: "春の気配" };

function createMessageId() {
  messageSequence += 1;
  return `message-${Date.now()}-${messageSequence}`;
}

const initialMessages = [
  {
    id: createMessageId(),
    role: "assistant",
    content:
      "こんにちは。Qwen とつながると、盆栽の様子を見ながら静かに返答します。まずは水やりや日照について聞いてみてください。"
  }
];

function formatPercent(value) {
  if (typeof value !== "number") return "--%";
  return `${Math.round(value)}%`;
}

function formatLux(value) {
  if (typeof value !== "number") return "-- lux";
  return `${Math.round(value).toLocaleString("ja-JP")} lux`;
}

function formatTemperature(value) {
  if (typeof value !== "number") return "--°C";
  return `${value.toFixed(1)}°C`;
}

function LeafMark() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path
        d="M12 19V11M12 11C9.3 11 7 8.9 7 6.2V5h1.8C10.6 5 12 6.4 12 8.2M12 11C14.7 11 17 8.9 17 6.2V5h-1.8C13.4 5 12 6.4 12 8.2"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function WifiMark() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path
        d="M5 10.5A11 11 0 0 1 19 10.5M8.5 14A6.5 6.5 0 0 1 15.5 14M12 18h.01"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function WaterMark() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path
        d="M12 3.8c2.9 3.7 4.8 6.1 4.8 8.6A4.8 4.8 0 1 1 7.2 12.4C7.2 9.9 9.1 7.5 12 3.8Z"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function SunMark() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="12" cy="12" r="3.4" fill="none" stroke="currentColor" strokeWidth="1.8" />
      <path
        d="M12 3.5V6M12 18V20.5M20.5 12H18M6 12H3.5M18 6L16.2 7.8M7.8 16.2L6 18M18 18l-1.8-1.8M7.8 7.8L6 6"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
      />
    </svg>
  );
}

function MessageBubble({ message, plantName }) {
  return (
    <article className={`message message-${message.role}`}>
      <div className="message-meta">
        <span>{message.role === "assistant" ? plantName : "You"}</span>
      </div>
      <p>{message.content || "..."}</p>
    </article>
  );
}

export default function App() {
  const [messages, setMessages] = useState(initialMessages);
  const [input, setInput] = useState("");
  const [sensors, setSensors] = useState(demoSensors);
  const [chatSource, setChatSource] = useState("offline");
  const [isStreaming, setIsStreaming] = useState(false);
  const listRef = useRef(null);

  useEffect(() => {
    let active = true;
    const controller = new AbortController();

    async function loadSensors() {
      const nextSensors = await fetchSensors(controller.signal);
      if (!active) return;
      setSensors(nextSensors);
    }

    loadSensors();
    const intervalId = window.setInterval(loadSensors, 30000);

    return () => {
      active = false;
      controller.abort();
      window.clearInterval(intervalId);
    };
  }, []);

  useEffect(() => {
    const node = listRef.current;
    if (!node) return;
    node.scrollTop = node.scrollHeight;
  }, [messages, isStreaming]);

  const selectedPlant = plantProfile;
  const isConnected = chatSource === "live";

  const promptHistory = useMemo(
    () =>
      messages.slice(-8).map((message) => ({
        role: message.role,
        content: message.content
      })),
    [messages]
  );

  async function handleSend(nextMessage) {
    const trimmed = nextMessage.trim();
    if (!trimmed || isStreaming) return;

    const userMessage = {
      id: createMessageId(),
      role: "user",
      content: trimmed
    };
    const assistantMessage = {
      id: createMessageId(),
      role: "assistant",
      content: ""
    };

    setInput("");
    setIsStreaming(true);
    setMessages((current) => [...current, userMessage, assistantMessage]);

    let source = "offline";

    try {
      const result = await streamChat({
        message: trimmed,
        history: promptHistory,
        sensors,
        onDelta: (delta) => {
          setMessages((current) =>
            current.map((message) =>
              message.id === assistantMessage.id
                ? { ...message, content: `${message.content}${delta}` }
                : message
            )
          );
        }
      });
      source = result.source;
    } catch (_error) {
      setMessages((current) =>
        current.map((message) =>
          message.id === assistantMessage.id
            ? {
                ...message,
                content:
                  "Qwen に接続できませんでした。llama.cpp サーバーと Go バックエンドが起動しているか確認してください。"
              }
            : message
        )
      );
    } finally {
      setChatSource(source);
      setIsStreaming(false);
    }
  }

  return (
    <div className="shell">
      <div className="shell-pattern" />

      <header className="hero">
        <div className="hero-copy">
          <h1>言の葉</h1>
          <p className="hero-lead">{selectedPlant.species}</p>
        </div>
        <div className="hero-status">
          <span className={`status-pill${isConnected ? " status-pill-online" : ""}`}>
            <span className="status-dot" />
            {isConnected ? "Qwen Live" : "Qwen Offline"}
          </span>
          <span className="status-time">{sensors.lastUpdated}</span>
        </div>
      </header>

      <main className="chat-layout">
        <section className="chat panel">
          <header className="chat-header">
            <div className="chat-title-block">
              <div className="selected-avatar chat-avatar" aria-hidden="true">
                <LeafMark />
              </div>
              <div className="chat-heading-copy">
                <h2>{selectedPlant.name} と話す</h2>
                <p className="chat-subline">
                  {selectedPlant.species} ・ {sensors.lastUpdated}
                </p>
              </div>
              <span className="season-chip">{selectedPlant.season}</span>
            </div>
            <div className="chat-header-side" aria-hidden="true">
              <WifiMark />
            </div>
          </header>

          <div className="sensor-strip">
            <span className="mini-badge">
              <WaterMark />
              土壌 {formatPercent(sensors.soilMoisture)}
            </span>
            <span className="mini-badge">
              <SunMark />
              {formatLux(sensors.illuminance)}
            </span>
            <span className="mini-badge">
              温度 {formatTemperature(sensors.temperature)}
            </span>
            <span className="mini-badge">
              湿度 {formatPercent(sensors.humidity)}
            </span>
          </div>

          <div ref={listRef} className="message-list">
            {messages.map((message) => (
              <MessageBubble key={message.id} message={message} plantName={selectedPlant.name} />
            ))}
          </div>

          <form
            className="composer"
            onSubmit={(event) => {
              event.preventDefault();
              handleSend(input);
            }}
          >
            <div className="prompt-row">
              {quickPrompts.map((prompt) => (
                <button
                  key={prompt}
                  className="prompt-chip"
                  type="button"
                  onClick={() => handleSend(prompt)}
                  disabled={isStreaming}
                >
                  {prompt}
                </button>
              ))}
            </div>

            <label className="sr-only" htmlFor="message">
              メッセージ
            </label>
            <textarea
              id="message"
              value={input}
              onInput={(event) => setInput(event.currentTarget.value)}
              placeholder="植物にそっと話しかけるように入力してください"
              rows={4}
              disabled={isStreaming}
            />
            <div className="composer-footer">
              <p>
                {isStreaming
                  ? `${selectedPlant.name} が言葉を選んでいます...`
                  : "Qwen 接続時は /api/chat/stream からストリーミング表示します。"}
              </p>
              <button type="submit" disabled={isStreaming || !input.trim()}>
                {isStreaming ? "Streaming..." : "Send"}
              </button>
            </div>
          </form>
        </section>
      </main>
    </div>
  );
}
