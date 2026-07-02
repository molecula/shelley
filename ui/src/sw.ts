/// <reference lib="webworker" />

export {};

// The ambient DOM lib (loaded for the rest of the app) types the global `self`
// as a Window, so we can't redeclare it. Cast once to the service-worker scope;
// using `sw.addEventListener` gives the callbacks their correct SW event types
// (PushEvent, NotificationEvent, ExtendableEvent).
const sw = self as unknown as ServiceWorkerGlobalScope;

interface PushPayload {
  title?: string;
  body?: string;
  tag?: string;
  url?: string;
}

sw.addEventListener("push", (event) => {
  let data: PushPayload = {};
  try {
    data = event.data?.json() ?? {};
  } catch {
    data = { body: event.data?.text() ?? "" };
  }

  const title = data.title || "Shelley";
  const options: NotificationOptions = {
    body: data.body || "",
    icon: "/icon-192.png",
    badge: "/icon-192.png",
    tag: data.tag || "shelley",
    data: { url: data.url || "/" },
  };

  event.waitUntil(sw.registration.showNotification(title, options));
});

sw.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const url: string = (event.notification.data as { url?: string })?.url || "/";

  event.waitUntil(
    sw.clients.matchAll({ type: "window", includeUncontrolled: true }).then((windowClients) => {
      for (const client of windowClients) {
        if ("focus" in client) {
          return (client as WindowClient).focus();
        }
      }
      return sw.clients.openWindow(url);
    }),
  );
});

sw.addEventListener("install", (event) => {
  event.waitUntil(sw.skipWaiting());
});

// Take control of all pages immediately on activation.
sw.addEventListener("activate", (event) => {
  event.waitUntil(sw.clients.claim());
});
