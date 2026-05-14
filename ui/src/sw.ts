/// <reference lib="webworker" />

export {};

const sw = self as unknown as ServiceWorkerGlobalScope;

interface PushPayload {
  title?: string;
  body?: string;
  tag?: string;
  url?: string;
}

sw.addEventListener("push", (event: PushEvent) => {
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

sw.addEventListener("notificationclick", (event: NotificationEvent) => {
  event.notification.close();
  const url: string = (event.notification.data as { url?: string })?.url || "/";

  event.waitUntil(
    sw.clients.matchAll({ type: "window", includeUncontrolled: true }).then((windowClients) => {
      for (const client of windowClients) {
        if ("focus" in client) {
          return client.focus();
        }
      }
      return sw.clients.openWindow(url);
    }),
  );
});

sw.addEventListener("install", (event: ExtendableEvent) => {
  event.waitUntil(sw.skipWaiting());
});

sw.addEventListener("activate", (event: ExtendableEvent) => {
  event.waitUntil(sw.clients.claim());
});
