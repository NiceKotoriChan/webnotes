// 图标规则：标题 → mdi 图标名。
//
// 规则由后端托管（GET /icons/mdi_rules_default.json、/icons/mdi_rules_custom.json），
// 后端只发文件、不合并也不解析；合并与匹配在前端做。两份文件同构：
//
//   { "fallback": "file-document-outline", "rules": [ { "match": "\\.md$", "icon": "file-document-outline" } ] }
//
// match 是大小写不敏感的正则，匹配对象是笔记标题（含扩展名）；按数组顺序首条命中即停。
// 先跑 custom 再跑 default，所以自定义天然压过默认。
import { ref } from 'vue';

const DEFAULT_FILE = 'mdi_rules_default.json';
const CUSTOM_FILE = 'mdi_rules_custom.json';
const FALLBACK = 'file-document-outline';

interface RulesFile {
  fallback?: string;
  rules?: { match?: unknown; icon?: unknown }[];
}

const compiled = ref<{ re: RegExp; icon: string }[]>([]);
const fallbackIcon = ref(FALLBACK);

async function fetchRules(file: string): Promise<RulesFile | null> {
  try {
    // 后端对这两个文件每次现读盘并带 ETag，浏览器按 no-cache 语义重新验证即可
    const res = await fetch(`/icons/${file}`, { cache: 'no-cache' });
    if (!res.ok) return null; // 404 = 这个文件不在，按「没有这份规则」处理
    return (await res.json()) as RulesFile;
  } catch {
    return null;
  }
}

// 启动时拉一次。任何一份拿不到都只影响它自己：custom 缺 → 只有默认规则。
export async function loadIconRules(): Promise<void> {
  const [custom, def] = await Promise.all([fetchRules(CUSTOM_FILE), fetchRules(DEFAULT_FILE)]);
  const list: { re: RegExp; icon: string }[] = [];
  for (const file of [custom, def]) {
    for (const rule of file?.rules ?? []) {
      const { match, icon } = rule ?? {};
      if (typeof match !== 'string' || typeof icon !== 'string' || !match || !icon) continue;
      try {
        list.push({ re: new RegExp(match, 'i'), icon });
      } catch {
        // 规则最终由前端执行，后端不做正则语法校验；坏规则跳过，不影响其余
      }
    }
  }
  compiled.value = list;
  fallbackIcon.value = custom?.fallback || def?.fallback || FALLBACK;
}

export function matchIcon(title: string): string {
  const t = (title ?? '').trim();
  if (!t) return fallbackIcon.value;
  for (const { re, icon } of compiled.value) {
    if (re.test(t)) return icon;
  }
  return fallbackIcon.value;
}
