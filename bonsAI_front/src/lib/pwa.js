export function registerPwa() {
  if (!("serviceWorker" in navigator)) {
    return;
  }

  const serviceWorkerUrl = `${import.meta.env.BASE_URL}sw.js`;

  window.addEventListener("load", () => {
    navigator.serviceWorker.register(serviceWorkerUrl, {
      scope: import.meta.env.BASE_URL
    }).catch(() => {});
  });
}
