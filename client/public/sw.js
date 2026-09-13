// WebNotes Service Worker：离线可用（app shell 缓存）。
// 策略：静态资源（导航/JS/CSS/图标/manifest）缓存优先 + 网络回退；
// API 请求（/api/**）永不缓存，保证数据实时。
const CACHE = 'webnotes-v2';

// 预缓存 app shell：manifest 里的资源由 fetch 拦截填充
self.addEventListener('install', (e) => {
  self.skipWaiting();
  e.waitUntil(
    caches.open(CACHE).then((c) => c.addAll(['/', '/manifest.webmanifest', '/icons/icon.svg', '/icons/mdi.json']))
  );
});

self.addEventListener('activate', (e) => {
  e.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', (e) => {
  const url = new URL(e.request.url);

  // 仅处理同源 GET；API 与跨源直接放行（不缓存）
  if (e.request.method !== 'GET' || url.origin !== self.location.origin) return;
  if (url.pathname.startsWith('/api/')) return;

  // 静态资源：缓存优先，未命中回网络并回填缓存
  e.respondWith(
    caches.match(e.request).then(
      (cached) =>
        cached ||
        fetch(e.request).then((res) => {
          const copy = res.clone();
          caches.open(CACHE).then((c) => c.put(e.request, copy));
          return res;
        })
    )
  );
});
