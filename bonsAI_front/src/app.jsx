import { useEffect, useMemo, useRef, useState } from "preact/hooks";
import { fetchSensors, fetchSystemStatus, streamChat, translateChat } from "./lib/api.js";
import { createDemoSensorsSnapshot, createVeryDryDemoSensorsSnapshot } from "./lib/demo.js";

let messageSequence = 0;

const copyByLanguage = {
  ja: {
    quickPrompts: ["今日の様子を教えて", "水やりは必要？", "光は足りていそう？"],
    plantProfile: {
      name: "言の葉",
      species: "パキラ",
      season: "春の気配"
    },
    initialMessage:
      "こんにちは。ローカル LLM とつながると、盆栽の様子を見ながら静かに返答します。まずは水やりや日照について聞いてみてください。",
    connectionError:
      "LLM に接続できませんでした。llama.cpp サーバーと Go バックエンドが起動しているか確認してください。",
    userLabel: "あなた",
    emptyMessage: "...",
    statusLive: "LLM 接続中",
    statusOffline: "LLM オフライン",
    chatTitle: (plantName) => `${plantName}と話す`,
    sensorSoil: "土壌",
    sensorLight: "照度",
    sensorTemperature: "温度",
    sensorHumidity: "湿度",
    messageLabel: "メッセージ",
    messagePlaceholder: "植物にそっと話しかけるように入力してください",
    streamingHint: (plantName) => `${plantName} が言葉を選んでいます...`,
    idleHint: "LLM 接続時は /api/chat/stream からストリーミング表示します。",
    submitIdle: "送信",
    submitStreaming: "Streaming...",
    languageToggle: "表示言語を切り替える",
    languageJapanese: "日本語",
    languageEnglish: "English",
    lastUpdatedFallback: "たった今",
    demoModeLabel: "デモ表示",
    demoModeLive: "通常",
    demoModeVeryDry: "とても乾いた状態",
    demoAlertToggle: "通知デモ",
    demoAlertToggleOn: "通知デモ ON",
    demoAlertToggleOff: "通知デモ OFF",
    networkOnline: "ネット接続あり",
    networkOffline: "ネット接続なし",
    installApp: "アプリを追加",
    installedApp: "アプリ表示中"
  },
  en: {
    quickPrompts: [
      "How is it doing today?",
      "Does it need water?",
      "Is it getting enough light?"
    ],
    plantProfile: {
      name: "Kotonoha",
      species: "Pachira",
      season: "Hints of spring"
    },
    initialMessage:
      "Hello. When the local LLM is connected, I can quietly respond while keeping an eye on your bonsai. Start by asking about watering or light.",
    connectionError:
      "The LLM could not be reached. Please make sure the llama.cpp server and Go backend are running.",
    userLabel: "You",
    emptyMessage: "...",
    statusLive: "LLM Live",
    statusOffline: "LLM Offline",
    chatTitle: (plantName) => `Talk with ${plantName}`,
    sensorSoil: "Soil",
    sensorLight: "Light",
    sensorTemperature: "Temp",
    sensorHumidity: "Humidity",
    messageLabel: "Message",
    messagePlaceholder: "Write as if you are gently speaking to your plant",
    streamingHint: (plantName) => `${plantName} is choosing its words...`,
    idleHint: "When connected, streaming responses appear from /api/chat/stream.",
    submitIdle: "Send",
    submitStreaming: "Streaming...",
    languageToggle: "Toggle display language",
    languageJapanese: "Japanese",
    languageEnglish: "English",
    lastUpdatedFallback: "just now",
    demoModeLabel: "Demo View",
    demoModeLive: "Live",
    demoModeVeryDry: "Very Dry",
    demoAlertToggle: "Alert Demo",
    demoAlertToggleOn: "Alert Demo On",
    demoAlertToggleOff: "Alert Demo Off",
    networkOnline: "Online",
    networkOffline: "Offline",
    installApp: "Install App",
    installedApp: "App Mode"
  }
};

const SENSOR_ALERT_COOLDOWN_MS = 30 * 60 * 1000;

function createMessageId() {
  messageSequence += 1;
  return `message-${Date.now()}-${messageSequence}`;
}

function detectMessageLanguage(message) {
  if (/[ぁ-んァ-ン一-龯]/u.test(message)) return "ja";
  if (/[A-Za-z]/.test(message)) return "en";
  return "ja";
}

function buildLocalizedTranslations(copyKey) {
  return {
    ja: copyByLanguage.ja[copyKey],
    en: copyByLanguage.en[copyKey]
  };
}

function createChatMessage({
  role,
  content,
  sourceLanguage = detectMessageLanguage(content),
  translations = {},
  isStreaming = false,
  useTranslationsInHistory = false
}) {
  const normalizedContent = typeof content === "string" ? content : "";
  const nextTranslations = { ...translations };

  if (normalizedContent) {
    nextTranslations[sourceLanguage] = normalizedContent;
  }

  return {
    id: createMessageId(),
    role,
    content: normalizedContent,
    sourceLanguage,
    translations: nextTranslations,
    pendingTranslationLanguage: null,
    failedTranslationLanguage: null,
    isStreaming,
    useTranslationsInHistory
  };
}

function createLocalizedAssistantMessage(copyKey) {
  return createChatMessage({
    role: "assistant",
    content: copyByLanguage.ja[copyKey],
    sourceLanguage: "ja",
    translations: buildLocalizedTranslations(copyKey),
    useTranslationsInHistory: true
  });
}

function createLocalizedAssistantTextMessage(translations) {
  return createChatMessage({
    role: "assistant",
    content: translations.ja,
    sourceLanguage: "ja",
    translations,
    useTranslationsInHistory: true
  });
}

function formatPercent(value) {
  if (typeof value !== "number") return "--%";
  return `${Math.round(value)}%`;
}

function formatLux(value, language) {
  if (typeof value !== "number") return "-- lux";
  return `${Math.round(value).toLocaleString(language === "ja" ? "ja-JP" : "en-US")} lux`;
}

function formatTemperature(value) {
  if (typeof value !== "number") return "--°C";
  return `${value.toFixed(1)}°C`;
}

function formatLastUpdated(value, language) {
  if (!value || value === "just now" || value === "たった今") {
    return copyByLanguage[language].lastUpdatedFallback;
  }
  const parsed = Date.parse(value);
  if (!Number.isNaN(parsed)) {
    return new Intl.DateTimeFormat(language === "ja" ? "ja-JP" : "en-US", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit"
    }).format(new Date(parsed));
  }
  return value;
}

function isFiniteNumber(value) {
  return typeof value === "number" && Number.isFinite(value);
}

function sensorDelta(previousValue, nextValue) {
  if (!isFiniteNumber(previousValue) || !isFiniteNumber(nextValue)) {
    return null;
  }
  return nextValue - previousValue;
}

function buildSensorAlert(previousSensors, nextSensors) {
  const candidates = [];

  const soilDelta = sensorDelta(previousSensors.soilMoisture, nextSensors.soilMoisture);
  if (soilDelta !== null && Math.abs(soilDelta) >= 8) {
    const delta = Math.round(Math.abs(soilDelta));
    const current = Math.round(nextSensors.soilMoisture);

    if (soilDelta < 0) {
      candidates.push({
        key: "soil-down",
        score: Math.abs(soilDelta) / 8,
        translations: {
          ja: `土のうるおいがさっきより${delta}%ほど下がって、少しのどが渇いてきました。今は土壌水分${current}%なので、表土の様子をのぞいてもらえるとうれしいです。`,
          en: `My soil moisture just dropped by about ${delta}%, so I am starting to feel a little thirsty. It is now at ${current}%, and I would love a quick check on the surface soil.`
        }
      });
    } else {
      candidates.push({
        key: "soil-up",
        score: Math.abs(soilDelta) / 8,
        translations: {
          ja: `土がさっきより${delta}%ほどしっとりして、ほっとしています。今は土壌水分${current}%で、ひとまず落ち着いています。`,
          en: `My soil just became about ${delta}% more moist, and I am feeling relieved. I am now at ${current}% soil moisture and fairly comfortable for the moment.`
        }
      });
    }
  }

  const temperatureDelta = sensorDelta(previousSensors.temperature, nextSensors.temperature);
  if (temperatureDelta !== null && Math.abs(temperatureDelta) >= 2.5) {
    const delta = Math.abs(temperatureDelta).toFixed(1);
    const current = nextSensors.temperature.toFixed(1);

    if (temperatureDelta > 0) {
      candidates.push({
        key: "temperature-up",
        score: Math.abs(temperatureDelta) / 2.5,
        translations: {
          ja: `気温がさっきより${delta}℃上がって、少しぽかぽかしてきました。今は${current}℃なので、暑くなりすぎていないか気にしてもらえるとうれしいです。`,
          en: `The temperature jumped by ${delta}C, and things are starting to feel a bit toasty. I am now at ${current}C, so I would appreciate a quick check that it is not getting too warm.`
        }
      });
    } else {
      candidates.push({
        key: "temperature-down",
        score: Math.abs(temperatureDelta) / 2.5,
        translations: {
          ja: `気温がさっきより${delta}℃下がって、少し身をすくめています。今は${current}℃なので、冷えすぎていないか見てもらえると安心です。`,
          en: `The temperature dropped by ${delta}C, and I am curling up a little. I am now at ${current}C, so a quick check that it is not getting too chilly would help.`
        }
      });
    }
  }

  const humidityDelta = sensorDelta(previousSensors.humidity, nextSensors.humidity);
  if (humidityDelta !== null && Math.abs(humidityDelta) >= 12) {
    const delta = Math.round(Math.abs(humidityDelta));
    const current = Math.round(nextSensors.humidity);

    if (humidityDelta > 0) {
      candidates.push({
        key: "humidity-up",
        score: Math.abs(humidityDelta) / 12,
        translations: {
          ja: `湿度がさっきより${delta}%ほど上がって、空気がしっとりしてきました。今は${current}%なので、むれすぎないかだけ少し見ていてください。`,
          en: `Humidity rose by about ${delta}%, and the air feels noticeably softer now. It is at ${current}% at the moment, so just a small check that things are not getting too muggy would be lovely.`
        }
      });
    } else {
      candidates.push({
        key: "humidity-down",
        score: Math.abs(humidityDelta) / 12,
        translations: {
          ja: `湿度がさっきより${delta}%ほど下がって、空気が少しからっとしました。今は${current}%なので、乾きすぎていないか気にかけてもらえるとうれしいです。`,
          en: `Humidity fell by about ${delta}%, and the air feels a little drier now. It is currently ${current}%, so I would appreciate a quick glance to make sure things are not drying out too much.`
        }
      });
    }
  }

  const illuminanceDelta = sensorDelta(previousSensors.illuminance, nextSensors.illuminance);
  if (illuminanceDelta !== null && Math.abs(illuminanceDelta) >= 5000) {
    const delta = Math.round(Math.abs(illuminanceDelta)).toLocaleString("en-US");
    const current = Math.round(nextSensors.illuminance).toLocaleString("en-US");

    if (illuminanceDelta > 0) {
      candidates.push({
        key: "light-up",
        score: Math.abs(illuminanceDelta) / 5000,
        translations: {
          ja: `光がいっきに${delta}lxほど明るくなって、葉っぱが少し目を細めています。今は${current}lxなので、強すぎない光かだけ見てもらえるとうれしいです。`,
          en: `The light suddenly jumped by about ${delta} lux, and my leaves are squinting a little. I am at ${current} lux now, so I would love a quick check that the light is not too strong.`
        }
      });
    } else {
      candidates.push({
        key: "light-down",
        score: Math.abs(illuminanceDelta) / 5000,
        translations: {
          ja: `光がいっきに${delta}lxほど弱くなって、少しだけ眠たくなってきました。今は${current}lxなので、置き場所の明るさを見てもらえると助かります。`,
          en: `The light suddenly dropped by about ${delta} lux, and I am starting to feel a little sleepy. I am at ${current} lux now, so a quick look at my spot for brightness would help.`
        }
      });
    }
  }

  if (candidates.length === 0) {
    return null;
  }

  candidates.sort((left, right) => right.score - left.score);
  return candidates[0];
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

function MessageBubble({ message, plantName, userLabel, emptyMessage, language }) {
  const displayContent = message.translations[language] ?? message.content;

  return (
    <article className={`message message-${message.role}`}>
      <div className="message-meta">
        <span>{message.role === "assistant" ? plantName : userLabel}</span>
      </div>
      <p>{displayContent || emptyMessage}</p>
    </article>
  );
}

export default function App() {
  const [language, setLanguage] = useState("ja");
  const [messages, setMessages] = useState(() => [createLocalizedAssistantMessage("initialMessage")]);
  const [input, setInput] = useState("");
  const [sensors, setSensors] = useState(() => createDemoSensorsSnapshot());
  const [demoSensorMode, setDemoSensorMode] = useState("live");
  const [demoAlertMode, setDemoAlertMode] = useState(false);
  const [chatSource, setChatSource] = useState("offline");
  const [isStreaming, setIsStreaming] = useState(false);
  const [isOnline, setIsOnline] = useState(() => window.navigator.onLine);
  const [installPrompt, setInstallPrompt] = useState(null);
  const [isStandalone, setIsStandalone] = useState(
    () =>
      window.navigator.standalone === true ||
      window.matchMedia?.("(display-mode: standalone)").matches === true
  );
  const listRef = useRef(null);
  const previousLiveSensorsRef = useRef(null);
  const lastAlertAtRef = useRef(new Map());
  const pendingAlertRef = useRef(null);
  const copy = copyByLanguage[language];
  const activeSensors = useMemo(
    () => (demoSensorMode === "veryDry" ? createVeryDryDemoSensorsSnapshot() : sensors),
    [demoSensorMode, sensors]
  );

  useEffect(() => {
    let active = true;
    const controller = new AbortController();

    async function loadSystemStatus() {
      try {
        const status = await fetchSystemStatus(controller.signal);
        if (!active) return;
        setChatSource(status.llmConnected ? "live" : "offline");
      } catch (_error) {
        if (!active) return;
        setChatSource("offline");
      }
    }

    async function loadSensors() {
      const nextSensors = await fetchSensors(controller.signal);
      if (!active) return;
      setSensors(nextSensors);
    }

    loadSystemStatus();
    loadSensors();
    const intervalId = window.setInterval(loadSensors, 10000);

    return () => {
      active = false;
      controller.abort();
      window.clearInterval(intervalId);
    };
  }, []);

  useEffect(() => {
    document.documentElement.lang = language;
  }, [language]);

  useEffect(() => {
    const displayModeQuery = window.matchMedia?.("(display-mode: standalone)");

    function syncOnlineState() {
      setIsOnline(window.navigator.onLine);
    }

    function syncStandaloneState() {
      setIsStandalone(
        window.navigator.standalone === true || displayModeQuery?.matches === true
      );
    }

    function handleBeforeInstallPrompt(event) {
      event.preventDefault();
      setInstallPrompt(event);
    }

    function handleAppInstalled() {
      setInstallPrompt(null);
      syncStandaloneState();
    }

    syncOnlineState();
    syncStandaloneState();

    window.addEventListener("online", syncOnlineState);
    window.addEventListener("offline", syncOnlineState);
    window.addEventListener("beforeinstallprompt", handleBeforeInstallPrompt);
    window.addEventListener("appinstalled", handleAppInstalled);
    displayModeQuery?.addEventListener?.("change", syncStandaloneState);

    return () => {
      window.removeEventListener("online", syncOnlineState);
      window.removeEventListener("offline", syncOnlineState);
      window.removeEventListener("beforeinstallprompt", handleBeforeInstallPrompt);
      window.removeEventListener("appinstalled", handleAppInstalled);
      displayModeQuery?.removeEventListener?.("change", syncStandaloneState);
    };
  }, []);

  useEffect(() => {
    const shouldMonitorAlerts = activeSensors.source === "live" || demoAlertMode;
    if (!shouldMonitorAlerts) {
      previousLiveSensorsRef.current = null;
      pendingAlertRef.current = null;
      return;
    }

    const previousSensors = previousLiveSensorsRef.current;
    previousLiveSensorsRef.current = activeSensors;
    if (!previousSensors) return;

    const alert = buildSensorAlert(previousSensors, activeSensors);
    if (!alert) return;

    const now = Date.now();
    const lastAlertAt = lastAlertAtRef.current.get(alert.key) ?? 0;
    if (now-lastAlertAt < SENSOR_ALERT_COOLDOWN_MS) {
      return;
    }

    lastAlertAtRef.current.set(alert.key, now);

    if (isStreaming) {
      pendingAlertRef.current = alert;
      return;
    }

    setMessages((current) => [...current, createLocalizedAssistantTextMessage(alert.translations)]);
  }, [activeSensors, demoAlertMode, isStreaming]);

  useEffect(() => {
    if (isStreaming || !pendingAlertRef.current) return;

    const alert = pendingAlertRef.current;
    pendingAlertRef.current = null;
    setMessages((current) => [...current, createLocalizedAssistantTextMessage(alert.translations)]);
  }, [isStreaming]);

  useEffect(() => {
    const pendingMessages = messages
      .filter((message) => {
        if (!message.content.trim()) return false;
        if (message.isStreaming) return false;
        if (message.translations[language]) return false;
        if (message.pendingTranslationLanguage === language) return false;
        if (message.failedTranslationLanguage === language) return false;
        return true;
      })
      .map((message) => ({
        id: message.id,
        role: message.role,
        content: message.content
      }));

    if (pendingMessages.length === 0) return;

    const pendingIds = new Set(pendingMessages.map((message) => message.id));
    let cancelled = false;

    setMessages((current) =>
      current.map((message) =>
        pendingIds.has(message.id)
          ? { ...message, pendingTranslationLanguage: language }
          : message
      )
    );

    translateChat({ messages: pendingMessages, targetLanguage: language })
      .then((translations) => {
        if (cancelled) return;

        const translationsById = new Map(
          translations.map((translation) => [translation.id, translation.content])
        );

        setMessages((current) =>
          current.map((message) => {
            if (!pendingIds.has(message.id)) return message;

            const translatedContent = translationsById.get(message.id);
            return {
              ...message,
              translations: translatedContent
                ? { ...message.translations, [language]: translatedContent }
                : message.translations,
              pendingTranslationLanguage:
                message.pendingTranslationLanguage === language
                  ? null
                  : message.pendingTranslationLanguage,
              failedTranslationLanguage:
                translatedContent ? null : language
            };
          })
        );
      })
      .catch(() => {
        if (cancelled) return;

        setMessages((current) =>
          current.map((message) =>
            pendingIds.has(message.id)
              ? {
                  ...message,
                  pendingTranslationLanguage:
                    message.pendingTranslationLanguage === language
                      ? null
                      : message.pendingTranslationLanguage,
                  failedTranslationLanguage: language
                }
              : message
          )
        );
      });

    return () => {
      cancelled = true;
    };
  }, [language, messages]);

  useEffect(() => {
    const node = listRef.current;
    if (!node) return;
    node.scrollTop = node.scrollHeight;
  }, [messages, isStreaming]);

  const selectedPlant = copy.plantProfile;
  const quickPrompts = copy.quickPrompts;
  const isConnected = chatSource === "live";
  const canInstall = Boolean(installPrompt) && !isStandalone;

  const promptHistory = useMemo(
    () =>
      messages.slice(-8).map((message) => ({
        role: message.role,
        content: message.useTranslationsInHistory
          ? (message.translations[language] ?? message.content)
          : message.content
      })),
    [language, messages]
  );

  function handleLanguageToggle() {
    const nextLanguage = language === "ja" ? "en" : "ja";

    setMessages((current) =>
      current.map((message) =>
        message.failedTranslationLanguage === nextLanguage
          ? { ...message, failedTranslationLanguage: null }
          : message
      )
    );
    setLanguage(nextLanguage);
  }

  async function handleInstall() {
    if (!installPrompt) return;

    installPrompt.prompt();

    try {
      await installPrompt.userChoice;
    } catch (_error) {
      // Ignore prompt dismissal; the browser may emit the event again later.
    } finally {
      setInstallPrompt(null);
    }
  }

  async function handleSend(nextMessage) {
    const trimmed = nextMessage.trim();
    if (!trimmed || isStreaming) return;

    const userMessage = createChatMessage({
      role: "user",
      content: trimmed
    });
    const assistantMessage = createChatMessage({
      role: "assistant",
      content: "",
      sourceLanguage: detectMessageLanguage(trimmed),
      isStreaming: true
    });

    setInput("");
    setIsStreaming(true);
    setMessages((current) => [...current, userMessage, assistantMessage]);

    let source = "offline";

    try {
      const result = await streamChat({
        message: trimmed,
        history: promptHistory,
        sensors: activeSensors,
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
                content: copyByLanguage.ja.connectionError,
                sourceLanguage: "ja",
                translations: buildLocalizedTranslations("connectionError"),
                pendingTranslationLanguage: null,
                failedTranslationLanguage: null,
                isStreaming: false,
                useTranslationsInHistory: true
              }
            : message
        )
      );
    } finally {
      setMessages((current) =>
        current.map((message) => {
          if (message.id !== assistantMessage.id || !message.isStreaming) {
            return message;
          }

          const sourceLanguage = detectMessageLanguage(message.content);
          return {
            ...message,
            sourceLanguage,
            translations: message.content
              ? { ...message.translations, [sourceLanguage]: message.content }
              : message.translations,
            isStreaming: false
          };
        })
      );
      setChatSource(source);
      setIsStreaming(false);
    }
  }

  return (
    <div className="shell" lang={language}>
      <div className="shell-pattern" />

      <header className="hero">
        <div className="hero-copy">
          <h1>{selectedPlant.name}</h1>
          <p className="hero-lead">{selectedPlant.species}</p>
        </div>
        <div className="hero-status">
          <span className={`status-pill${isConnected ? " status-pill-online" : ""}`}>
            <span className="status-dot" />
            {isConnected ? copy.statusLive : copy.statusOffline}
          </span>
          <span className="status-time">{formatLastUpdated(activeSensors.lastUpdated, language)}</span>
          <div className="status-meta">
            <span className={`status-chip${isOnline ? " status-chip-online" : ""}`}>
              {isOnline ? copy.networkOnline : copy.networkOffline}
            </span>
            {isStandalone ? <span className="status-chip">{copy.installedApp}</span> : null}
            {canInstall ? (
              <button
                type="button"
                className="status-chip status-chip-button"
                onClick={handleInstall}
              >
                {copy.installApp}
              </button>
            ) : null}
          </div>
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
                <h2>{copy.chatTitle(selectedPlant.name)}</h2>
                <p className="chat-subline">
                  {selectedPlant.species} ・ {formatLastUpdated(activeSensors.lastUpdated, language)}
                </p>
              </div>
              <span className="season-chip">{selectedPlant.season}</span>
            </div>
            <div className="chat-header-side">
              <div className="language-switcher">
                <span
                  className={`language-option${language === "ja" ? " language-option-active" : ""}`}
                >
                  {copy.languageJapanese}
                </span>
                <button
                  type="button"
                  className={`language-switch${language === "en" ? " language-switch-on" : ""}`}
                  role="switch"
                  aria-checked={language === "en"}
                  aria-label={copy.languageToggle}
                  onClick={handleLanguageToggle}
                >
                  <span className="language-switch-thumb" />
                </button>
                <span
                  className={`language-option${language === "en" ? " language-option-active" : ""}`}
                >
                  {copy.languageEnglish}
                </span>
              </div>
              <div className="demo-switcher" role="group" aria-label={copy.demoModeLabel}>
                <span className="demo-switcher-label">{copy.demoModeLabel}</span>
                <button
                  type="button"
                  className={`demo-chip${demoSensorMode === "live" ? " demo-chip-active" : ""}`}
                  onClick={() => setDemoSensorMode("live")}
                >
                  {copy.demoModeLive}
                </button>
                <button
                  type="button"
                  className={`demo-chip${demoSensorMode === "veryDry" ? " demo-chip-active" : ""}`}
                  onClick={() => setDemoSensorMode("veryDry")}
                >
                  {copy.demoModeVeryDry}
                </button>
                <button
                  type="button"
                  className={`demo-chip${demoAlertMode ? " demo-chip-active" : ""}`}
                  aria-pressed={demoAlertMode}
                  onClick={() => setDemoAlertMode((current) => !current)}
                  title={demoAlertMode ? copy.demoAlertToggleOn : copy.demoAlertToggleOff}
                >
                  {copy.demoAlertToggle}
                </button>
              </div>
              {canInstall ? (
                <button
                  type="button"
                  className="mobile-utility-button"
                  onClick={handleInstall}
                >
                  {copy.installApp}
                </button>
              ) : null}
              <div className="chat-header-signal" aria-hidden="true">
                <WifiMark />
              </div>
            </div>
          </header>

          <div className="sensor-strip">
            <span className="mini-badge">
              <WaterMark />
              {copy.sensorSoil} {formatPercent(activeSensors.soilMoisture)}
            </span>
            <span className="mini-badge">
              <SunMark />
              {copy.sensorLight} {formatLux(activeSensors.illuminance, language)}
            </span>
            <span className="mini-badge">
              {copy.sensorTemperature} {formatTemperature(activeSensors.temperature)}
            </span>
            <span className="mini-badge">
              {copy.sensorHumidity} {formatPercent(activeSensors.humidity)}
            </span>
          </div>

          <section className="conversation-pane">
            <div ref={listRef} className="message-list">
              {messages.map((message) => (
                <MessageBubble
                  key={message.id}
                  message={message}
                  plantName={selectedPlant.name}
                  userLabel={copy.userLabel}
                  emptyMessage={copy.emptyMessage}
                  language={language}
                />
              ))}
            </div>
          </section>

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
              {copy.messageLabel}
            </label>
            <textarea
              id="message"
              value={input}
              onInput={(event) => setInput(event.currentTarget.value)}
              placeholder={copy.messagePlaceholder}
              rows={4}
              disabled={isStreaming}
            />
            <div className="composer-footer">
              <p>
                {isStreaming
                  ? copy.streamingHint(selectedPlant.name)
                  : copy.idleHint}
              </p>
              <button type="submit" disabled={isStreaming || !input.trim()}>
                {isStreaming ? copy.submitStreaming : copy.submitIdle}
              </button>
            </div>
          </form>
        </section>
      </main>
    </div>
  );
}
