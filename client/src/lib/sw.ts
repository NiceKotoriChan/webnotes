// Service Worker 注册（生产模式启用；dev 下 Vite HMR 与 SW 冲突，跳过）
export function registerSW() {
  if (!import.meta.env.PROD) return;
  if (!('serviceWorker' in navigator)) return;
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch(() => {
      /* 注册失败不阻塞应用 */
    });
  });
}
