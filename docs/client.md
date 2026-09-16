vue3 + tailwindCSS（Tailwind v4，`@tailwindcss/vite`）。

## 技术要点

- **Tailwind v4**：配色对齐 GitHub（`@theme` 自定义色板 `--color-*`），明暗主题靠
  `<html class="dark">` 切换，见 `src/app.css`。
- **所见即所得编辑器**：`@tiptap/core` + starter-kit + image/link/task-list 扩展，
  Svelte 5 手动集成（`src/lib/tiptap.ts`、`src/components/WysiwygEditor.svelte`）。
- **Markdown 是真相源**：打开笔记用 `marked` 渲染 Markdown→HTML 注入编辑器；编辑防抖
  600ms 用 `turndown` 把 HTML 转回 Markdown 存库（`src/lib/md.ts`）。旧笔记无缝兼容，
  后端 FTS 搜索语义不变。
- **自动保存**：内容变化防抖 600ms 保存 + 5s 兜底 + Ctrl/Cmd+S 立即保存；状态栏显示
  已保存/未保存/保存中。
- **附件**：客户端算 sha256 秒传；上传后插入编辑器光标处。
- **PWA**：`public/manifest.webmanifest` + `public/sw.js`（app shell 缓存，API 不缓存），
  生产模式注册，可离线打开、可安装。

## 功能

ctrl + p 搜索，独立组件，和 VSC 一样
