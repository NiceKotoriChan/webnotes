// Markdown 渲染与源码编辑辅助。
// 正文的真相源就是 Markdown 本身：编辑器是源码框，右侧用 markdown-it 实时渲染预览。
import MarkdownIt from 'markdown-it';

const md = new MarkdownIt({
  html: false, // 不解析正文里的裸 HTML：笔记内容一律走 Markdown 语法
  linkify: true, // 裸 URL 自动成链接
  breaks: true, // 单换行即 <br>，与笔记的书写直觉一致
});

// 外链一律新窗口打开
const renderToken =
  md.renderer.rules.link_open ??
  ((tokens, idx, opts, _env, self) => self.renderToken(tokens, idx, opts));
md.renderer.rules.link_open = (tokens, idx, opts, env, self) => {
  tokens[idx].attrSet('target', '_blank');
  tokens[idx].attrSet('rel', 'noopener noreferrer');
  return renderToken(tokens, idx, opts, env, self);
};

export function renderMarkdown(src: string): string {
  return md.render(src ?? '');
}

// —— 以下三个是给工具栏用的纯文本变换：进出都是「文本 + 选区」，不碰 DOM ——

export interface TextSel {
  text: string;
  start: number;
  end: number;
}

// 用 before/after 包住选区；没有选区就插入 before+after 并把光标放中间
export function wrapSelection(s: TextSel, before: string, after = before): TextSel {
  const picked = s.text.slice(s.start, s.end);
  const inner = picked || '';
  const text = s.text.slice(0, s.start) + before + inner + after + s.text.slice(s.end);
  const start = s.start + before.length;
  return { text, start, end: start + inner.length };
}

// 给选区覆盖的整行加前缀（列表、引用、标题）
export function prefixLines(s: TextSel, prefix: string): TextSel {
  const lineStart = s.text.lastIndexOf('\n', s.start - 1) + 1;
  let lineEnd = s.text.indexOf('\n', s.end);
  if (lineEnd === -1) lineEnd = s.text.length;

  const block = s.text.slice(lineStart, lineEnd);
  const next = block
    .split('\n')
    .map((l) => (l.startsWith(prefix) ? l.slice(prefix.length) : prefix + l))
    .join('\n');
  const text = s.text.slice(0, lineStart) + next + s.text.slice(lineEnd);
  return { text, start: lineStart, end: lineStart + next.length };
}

// 在光标处插入片段，并把光标移到片段末尾（或指定偏移）
export function insertAt(s: TextSel, snippet: string, cursorOffset = snippet.length): TextSel {
  const text = s.text.slice(0, s.start) + snippet + s.text.slice(s.end);
  const pos = s.start + cursorOffset;
  return { text, start: pos, end: pos };
}
