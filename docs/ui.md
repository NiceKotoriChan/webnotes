# 视图

> 描述**当前代码**，真相源是 `client/src/`。⚠️ 后端按 [api.md](api.md) / [model.md](model.md) 落地时，本文一起改 —— 字段改名与路由收敛会让下面多处失效。

Vue 3 + TypeScript + Tailwind CSS v4（`@tailwindcss/vite` 插件），Vite 6，pnpm。

## 分层

| 目录 | 职责 |
|---|---|
| `client/src/api.ts` | 全部 HTTP 调用的薄封装：统一 `request()`、`ApiError`、领域类型，以及附件上传（sha256 秒传 + XHR 进度） |
| `client/src/composables/` | **模块级共享状态**：`useRepos` / `useNotes` / `useTags` / `useAssets` / `useCurrentRepo` / `useTheme`。`ref` 定义在模块作用域，全局只有一份 —— 是单例 store，不是"每次调用新建"的工厂 |
| `client/src/lib/` | 无状态纯函数：`tree`（扁平列表建树）、`md`（Markdown ↔ HTML）、`tiptap`（编辑器实例与命令）、`autoIcon`（标题→图标）、`drag`、`sw`、`icons` |
| `client/src/components/` | `Workspace.vue` 是主布局（三栏），其余按区域分目录：`ActivityBar/`、`SideBar/`、`Editor/` |

## 数据流

`Workspace.vue` 是编排中心：持有当前仓库、选中笔记、折叠状态、标签关联，把笔记操作（新建 / 改名 / 移动 / 删除 / 设图标）组装后向下传；叶子组件（`TreeList`、`TagsPanel`、`AssetPanel` …）通过回调或 emit 把动作交回上层，再由上层调 `client/src/api.ts` 并刷新对应 composable。

因此**状态的真实归属是 composable 模块，不是某个组件** —— 例如 `SearchPanel` 打开时调的 `loadAssets()` 与侧边栏 `AssetPanel` 是同一份状态，会互相刷新。

标签列表可以直接从已加载的笔记本地解析，不必每次变更都回头请求后端（见 [api.md](api.md)）。

## 关键机制

- **Markdown 是真相源**：打开笔记用 `marked` 渲染成 HTML 注入编辑器；编辑防抖 600ms 用 `turndown` 转回 Markdown 存库（`client/src/lib/md.ts`）。
- **所见即所得编辑器**：`@tiptap/core` + starter-kit + image / link / task-list，Vue 3 手动集成（`client/src/lib/tiptap.ts`、`client/src/components/Editor/NoteEditor.vue`）。往返对任务列表做了专门处理（Turndown 默认规则会丢 checkbox 勾选态）。
- **自动保存**：内容变化防抖 600ms + 5s 兜底 + `Ctrl/Cmd+S` 立即保存；状态栏显示 已保存 / 未保存 / 保存中。切换笔记靠父级 `:key` 重新挂载编辑器，不复用实例。
- **附件**：`crypto.subtle` 算 sha256 → 探测秒传 → `XMLHttpRequest` 上传并回报进度 → 按类型插入正文（图片 `<img>`，其他 `<a>`）。拖入与粘贴走同一条路径。（当前探测用 `HEAD`，目标设计改成 `GET .../assets/:sha/meta`。）
- **图标**：图标集由后端托管（`/icons/mdi.json`），前端启动即全量加载并注册到 `@iconify/vue/offline`，全程本地、可离线。没有自定义图标的笔记由 `client/src/lib/autoIcon.ts` 按标题关键词与扩展名匹配。
- **主题**：对齐 GitHub 配色。`client/src/app.css` 用 CSS 变量定义亮/暗两套原始值，经 `@theme inline` 注册成 Tailwind 的 `--color-*`；切换靠 `<html class="dark">`（`client/src/composables/useTheme.ts`），首次访问读 `prefers-color-scheme`。
- **PWA**：`client/public/manifest.webmanifest` + `client/public/sw.js`。静态资源缓存优先、`/icons/mdi.json` 网络优先（后端会自动更新它）、`/api/**` 永不缓存；仅生产模式注册（dev 下与 HMR 冲突）。

## 快捷键

| 按键 | 行为 |
|---|---|
| `Ctrl/Cmd+P` | 全局搜索面板。笔记走服务端 FTS，标签与附件在客户端匹配；`↑↓` 导航、`Enter` 选择、`Esc` 关闭 |
| `Ctrl/Cmd+S` | 立即保存当前笔记 |
