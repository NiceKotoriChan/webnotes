# 客户端架构

> **与代码同步** —— 真相源是 `client/src/`。分层或数据流变化时同步本文档。

Vue 3 + TypeScript + Tailwind CSS v4（走 `@tailwindcss/vite` 插件）。构建工具 Vite 6，包管理 pnpm。

## 分层

| 目录 | 职责 |
|---|---|
| `client/src/api.ts` | 全部 HTTP 调用的薄封装：统一 `request()`、`ApiError`、领域类型，以及附件上传（sha256 秒传 + XHR 进度回报） |
| `client/src/composables/` | **模块级共享状态**：`useRepos` / `useNotes` / `useTags` / `useAssets` / `useCurrentRepo` / `useTheme`。状态 `ref` 定义在模块作用域，所以全局只有一份 —— 这不是"每次调用新建一份"的工厂，而是单例式的 store |
| `client/src/lib/` | 无状态的纯函数与适配层：`tree`（扁平列表建树）、`md`（Markdown ↔ HTML）、`tiptap`（编辑器实例与命令封装）、`autoIcon`（标题→图标匹配）、`drag`（跨递归实例的拖拽状态）、`sw`（Service Worker 注册）、`icons`（图标集加载） |
| `client/src/components/` | 界面组件。`Workspace.vue` 是主布局（活动栏 / 侧边栏 / 编辑器三栏），其余按界面区域分目录：`ActivityBar/`、`SideBar/`、`Editor/` |

## 数据流

`Workspace.vue` 是编排中心：它持有当前仓库、选中笔记、折叠状态、标签关联，并把笔记操作（新建 / 改名 / 移动 / 删除 / 设置图标）与标签操作组装后向下传。叶子组件（`TreeList`、`TagsPanel`、`AssetPanel` …）通过这些回调或 emit 把动作交回上层，再由上层调用 `client/src/api.ts` 并刷新对应的 composable。

因此**状态的真实归属是 composable 模块，而不是某个组件**：例如 `SearchPanel` 打开时调用的 `loadAssets()` 与侧边栏 `AssetPanel` 用的是同一份状态，会互相刷新。

## 关键机制

- **Markdown 是真相源**：打开笔记用 `marked` 把 Markdown 渲染成 HTML 注入编辑器；编辑防抖 600ms 用 `turndown` 把 HTML 转回 Markdown 存库（`client/src/lib/md.ts`）。旧笔记无缝兼容，后端 FTS 搜索语义不变。
- **所见即所得编辑器**：`@tiptap/core` + starter-kit + image / link / task-list 扩展，Vue 3 手动集成（`client/src/lib/tiptap.ts`、`client/src/components/Editor/NoteEditor.vue`）。Markdown↔HTML 的往返对任务列表做了专门处理（Turndown 默认规则会丢掉 checkbox 勾选态，被覆盖的 `listItem` 规则补上了）。
- **自动保存**：内容变化防抖 600ms + 5s 兜底定时 + `Ctrl/Cmd+S` 立即保存；状态栏显示 已保存 / 未保存 / 保存中。切换笔记靠父级的 `:key` 重新挂载编辑器，而不是复用实例。
- **附件**：客户端 `crypto.subtle` 算 sha256 → `HEAD` 判断秒传 → `XMLHttpRequest` 上传并回报进度 → 上传完成后按类型插入正文（图片插 `<img>`，其他插 `<a>`）。拖入与粘贴的文件走同一条路径。
- **图标**：图标集由后端托管（`GET /icons/mdi.json`），前端启动即全量加载并注册到 `@iconify/vue/offline`，全程本地、不发外网请求、可离线。没有自定义图标的笔记由 `client/src/lib/autoIcon.ts` 按标题关键词与扩展名自动匹配。
- **主题**：对齐 GitHub 配色。`client/src/app.css` 里用 CSS 变量定义亮/暗两套原始值，再经 `@theme inline` 注册为 Tailwind 的 `--color-*` 语义色；明暗切换靠 `<html class="dark">`（`client/src/composables/useTheme.ts`），首次访问读 `prefers-color-scheme`。
- **PWA**：`client/public/manifest.webmanifest` + `client/public/sw.js`。策略为静态资源缓存优先、`/icons/mdi.json` 网络优先（因为后端会自动更新它）、`/api/**` 永不缓存；仅生产模式注册（dev 下与 Vite HMR 冲突）。

## 快捷键

| 按键 | 行为 |
|---|---|
| `Ctrl/Cmd+P` | 打开全局搜索面板。笔记走服务端 FTS，标签与附件在客户端匹配；支持 `↑↓` 导航、`Enter` 选择、`Esc` 关闭 |
| `Ctrl/Cmd+S` | 立即保存当前笔记 |

## 相关

活动栏与各面板的**后续规划**（日历、概览、设置等尚未实现的项）记录在 [plan-ui.md](plan-ui.md)。
