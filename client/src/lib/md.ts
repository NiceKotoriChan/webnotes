// Markdown ↔ TipTap(HTML) 转换工具。
// 设计：Markdown 是真相源（与后端一致）。打开笔记时用 marked 渲染 Markdown→HTML 喂给
// TipTap；编辑器输出用 Turndown 把 HTML 转回 Markdown 存库，保证旧笔记无缝兼容、
// 搜索(FTS)与 API 语义不变。
import { marked } from 'marked';
import TurndownService from 'turndown';

// marked 全程启用 GFM + breaks（单个换行即 <br>，与编辑器的所见即所得行为对齐）
marked.setOptions({ gfm: true, breaks: true });

const turndown = new TurndownService({
  headingStyle: 'atx',
  codeBlockStyle: 'fenced',
  bulletListMarker: '-',
  emDelimiter: '*',
});

// 覆盖默认 listItem：支持 TipTap 任务列表的勾选态（默认规则会把 checkbox 丢弃）。
// 其余行为与默认一致（有序/无序前缀、前后换行）。
turndown.remove(['li']);
turndown.addRule('listItem', {
  filter: 'li',
  replacement: (content, node) => {
    const el = node as Element;
    content = content.replace(/^\n+/, '').replace(/\n+$/, '\n');
    const cb = el.querySelector('input[type="checkbox"]');
    if (cb) {
      const checked = cb.hasAttribute('checked');
      return `- [${checked ? 'x' : ' '}] ${content}`;
    }
    const parent = el.parentNode as Element | null;
    let prefix = '- ';
    if (parent?.nodeName === 'OL') {
      const start = Number(parent.getAttribute('start') ?? 1);
      const index = Array.prototype.indexOf.call(parent.children, el);
      prefix = `${start + index}. `;
    }
    return prefix + content + (el.nextSibling ? '\n' : '');
  },
});

// Markdown → HTML（打开笔记喂 TipTap）
export function mdToHtml(md: string): string {
  return marked.parse(md ?? '', { async: false }) as string;
}

// HTML（TipTap 输出） → Markdown（存库）
export function htmlToMd(html: string): string {
  return turndown.turndown(html ?? '');
}
