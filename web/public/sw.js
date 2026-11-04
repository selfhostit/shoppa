// Service worker: keeps the app shell cached so the app opens instantly, even
// offline, and keeps product photos so a list looks the same in a shop with no
// signal. Other API calls and the admin UI always go to the network.
const CACHE = 'shell-v3';
const PHOTOS = 'photos-v1';
const photo = /^\/api\/app\/products\/[a-z0-9]+\/photo$/;
const SHELL = ['/', '/manifest.webmanifest', '/icons/icon-192.png'];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches
      .open(CACHE)
      .then((c) => c.addAll(SHELL))
      .then(() => self.skipWaiting()),
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(keys.filter((k) => k !== CACHE && k !== PHOTOS).map((k) => caches.delete(k))),
      )
      .then(() => self.clients.claim()),
  );
});

self.addEventListener('fetch', (event) => {
  const req = event.request;
  const url = new URL(req.url);
  if (req.method !== 'GET' || url.origin !== self.location.origin) return;

  // a product's photo doesn't change: the copy kept here, else the server's
  if (photo.test(url.pathname)) {
    event.respondWith(
      caches.open(PHOTOS).then((c) =>
        c.match(req).then(
          (hit) =>
            hit ||
            fetch(req).then((res) => {
              if (res.ok) c.put(req, res.clone());
              return res;
            }),
        ),
      ),
    );
    return;
  }
  if (url.pathname.startsWith('/api/') || url.pathname.startsWith('/_/')) return;

  // pages: network first so updates show up, the cached shell when offline
  if (req.mode === 'navigate') {
    event.respondWith(fetch(req).catch(() => caches.match('/')));
    return;
  }

  // assets have hashed names, so a cached copy is always current
  event.respondWith(
    caches.match(req).then(
      (hit) =>
        hit ||
        fetch(req).then((res) => {
          if (res.ok) {
            const copy = res.clone();
            caches.open(CACHE).then((c) => c.put(req, copy));
          }
          return res;
        }),
    ),
  );
});

// Push notifications sent by the server (backend/push).
self.addEventListener('push', (event) => {
  const msg = event.data ? event.data.json() : { title: 'New notification' };
  event.waitUntil(
    self.registration.showNotification(msg.title, {
      body: msg.body,
      tag: msg.tag,
      icon: '/icons/icon-192.png',
      badge: '/icons/icon-192.png',
      data: { url: msg.url || '/' },
    }),
  );
});

// Tapping one opens its page, reusing a window that's already open.
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const url = new URL(event.notification.data?.url || '/', self.location.origin).href;
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((windows) => {
      const open = windows.find((w) => new URL(w.url).origin === self.location.origin);
      if (open) return open.navigate(url).then((w) => (w || open).focus());
      return self.clients.openWindow(url);
    }),
  );
});
