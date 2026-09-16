// WebNotes Service Worker：离线可用（app shell 缓存）。
// 策略：静态资源（导航/JS/CSS/图标/manifest）缓存优先 + 网络回退；
// 图标集 /icons/mdi.json 由后端托管且会自动更新，改网络优先（离线再回退缓存）；
// API 请求（/api/**）永不缓存，保证数据实时。
const CACHE = 'webnotes-v3';
const ICON_SET = '/icons/mdi.json';

// 预缓存 app shell：manifest 里的资源由 fetch 拦截填充
self.addEventListener('install', (e) => {
  self.skipWaiting();
  e.waitUntil(
    caches.open(CACHE).then((c) => c.addAll(['/', '/manifest.webmanifest', '/icons/icon.svg']))
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

  // 图标集：网络优先（后端会自动更新它），只有离线时才用缓存兜底
  if (url.pathname === ICON_SET) {
    e.respondWith(
      (async () => {
        try {
          const res = await fetch(url.pathname); // 后端 ETag 命中即 304，浏览器内部消化
          if (res.ok) {
            const cache = await caches.open(CACHE);
            await cache.put(ICON_SET, res.clone());
            return res;
          }
        } catch {
          /* 离线：落到缓存 */
        }
        return (await caches.match(ICON_SET)) || Response.error();
      })()
    );
    return;
  }

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
