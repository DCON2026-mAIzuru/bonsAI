const CACHE_NAME = "bonsai-shell-v1";
const STATIC_DESTINATIONS = new Set(["document", "script", "style", "image", "font", "manifest"]);
const API_PATH_PREFIXES = ["/api", "/healthz"];

function getScopeUrl() {
  return new URL(self.registration.scope);
}

function getIndexUrl() {
  return new URL("index.html", getScopeUrl()).toString();
}

function getAppShellUrls() {
  const scopeUrl = getScopeUrl();

  return [
    new URL("./", scopeUrl).toString(),
    getIndexUrl(),
    new URL("manifest.webmanifest", scopeUrl).toString(),
    new URL("runtime-config.js", scopeUrl).toString(),
    new URL("icons/icon.svg", scopeUrl).toString(),
    new URL("icons/pwa-192.png", scopeUrl).toString(),
    new URL("icons/pwa-512.png", scopeUrl).toString(),
    new URL("icons/maskable-512.png", scopeUrl).toString(),
    new URL("icons/apple-touch-icon.png", scopeUrl).toString()
  ];
}

function shouldBypass(request) {
  if (request.method !== "GET") {
    return true;
  }

  const url = new URL(request.url);
  if (url.origin !== self.location.origin) {
    return true;
  }

  const acceptHeader = request.headers.get("accept") || "";
  if (acceptHeader.includes("text/event-stream")) {
    return true;
  }

  return API_PATH_PREFIXES.some(
    (prefix) => url.pathname === prefix || url.pathname.startsWith(`${prefix}/`)
  );
}

async function cacheAppShell() {
  const cache = await caches.open(CACHE_NAME);
  await cache.addAll(getAppShellUrls());
}

async function cleanupOldCaches() {
  const cacheNames = await caches.keys();
  await Promise.all(
    cacheNames
      .filter((cacheName) => cacheName !== CACHE_NAME)
      .map((cacheName) => caches.delete(cacheName))
  );
}

async function cacheResponse(cache, request, response) {
  if (!response || !response.ok) {
    return response;
  }

  await cache.put(request, response.clone());
  return response;
}

async function handleNavigation(request) {
  const cache = await caches.open(CACHE_NAME);

  try {
    const response = await fetch(request);
    await cacheResponse(cache, request, response.clone());
    return response;
  } catch (_error) {
    return (
      (await cache.match(request)) ||
      (await cache.match(getIndexUrl())) ||
      new Response("Offline", {
        status: 503,
        statusText: "Offline"
      })
    );
  }
}

async function handleStaticAsset(request) {
  const cache = await caches.open(CACHE_NAME);
  const cachedResponse = await cache.match(request, {
    ignoreSearch: request.destination === "document"
  });

  if (cachedResponse) {
    void fetch(request)
      .then((response) => cacheResponse(cache, request, response))
      .catch(() => {});
    return cachedResponse;
  }

  try {
    const response = await fetch(request);
    return await cacheResponse(cache, request, response);
  } catch (_error) {
    return (
      cachedResponse ||
      new Response("Offline", {
        status: 503,
        statusText: "Offline"
      })
    );
  }
}

self.addEventListener("install", (event) => {
  event.waitUntil(
    cacheAppShell().then(() => self.skipWaiting())
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    cleanupOldCaches().then(() => self.clients.claim())
  );
});

self.addEventListener("fetch", (event) => {
  const { request } = event;

  if (shouldBypass(request)) {
    return;
  }

  if (request.mode === "navigate") {
    event.respondWith(handleNavigation(request));
    return;
  }

  if (!STATIC_DESTINATIONS.has(request.destination)) {
    return;
  }

  event.respondWith(handleStaticAsset(request));
});
