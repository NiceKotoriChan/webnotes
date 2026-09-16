// 图标数据：初始化即加载全量 mdi 集合。
// 图标集由后端托管（GET /icons/mdi.json，后端会自动更新它），全程本地、SW 预缓存、离线可用，不发外网请求。
import { ref } from 'vue';
import { addCollection, type IconifyJSON } from '@iconify/vue/offline';

const fullNames = ref<string[]>([]);

// 初始化即加载全量
fetch('/icons/mdi.json')
  .then((r) => r.json())
  .then((data) => {
    addCollection(data as IconifyJSON);
    fullNames.value = Object.keys(data.icons || {});
  })
  .catch(() => {});

export function allMdiNames(): string[] {
  return fullNames.value;
}
