// WebNotes Service Worker：离线可用（app shell 缓存）。
// 策略：静态资源（导航/JS/CSS/图标/manifest）缓存优先 + 网络回退；
// 图标集与图标规则都由后端托管且会自动更新，这些改网络优先（离线再回退缓存）；
// API 请求（/api/**）永不缓存，保证数据实时。
const CACHE = 'webnotes-v4';
const ICON_PREFIX = '/icons/';

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

  // 图标集 + 规则文件：网络优先（后端会更新它们），只有离线时才用缓存兜底
  if (url.pathname.startsWith(ICON_PREFIX)) {
    e.respondWith(
      (async () => {
        try {
          const res = await fetch(url.pathname); // 后端 ETag 命中即 304，浏览器内部消化
          // 404（比如自定义规则文件还没建）不落缓存，否则文件建好后前端永远拿不到
          if (res.status !== 404) {
            const cache = await caches.open(CACHE);
            await cache.put(url.pathname, res.clone());
          }
          return res;
        } catch {
          // 离线：落到缓存；缓存也没有才认输
          return (await caches.match(url.pathname)) || Response.error();
        }
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
