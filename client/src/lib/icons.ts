// 图标数据：初始化即加载全量 mdi 集合。
// 图标集由后端托管（GET /icons/mdi.json，后端会自动更新它），全程本地、SW 预缓存、离线可用，不发外网请求。
import { ref } from 'vue';
import { addCollection, type IconifyJSON } from '@iconify/vue/offline';

// 图标选择器的默认网格。**与标题自动匹配无关** —— 自动匹配走后端下发的规则文件，见 lib/rules.ts。
export const PRESET_ICONS = [
  'file-document-outline', 'file-outline', 'file-code-outline', 'folder', 'folder-open', 'notebook',
  'book', 'book-open-variant', 'calendar', 'calendar-range', 'clock-outline', 'checkbox-marked',
  'format-list-bulleted', 'format-list-checks', 'code-tags', 'console', 'code-braces', 'bug',
  'database', 'server', 'cloud-outline', 'lock-outline', 'key', 'account-outline', 'account-group',
  'heart', 'star', 'flag', 'target', 'lightbulb-on-outline', 'rocket-launch', 'wallet', 'cart',
  'email-outline', 'message-text-outline', 'image', 'video', 'music', 'map-outline', 'web', 'link',
  'paperclip', 'tag', 'cog', 'home', 'bell-outline', 'archive',
];

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
