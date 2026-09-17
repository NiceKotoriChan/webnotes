# WebNotes

本地优先的个人笔记系统。数据全部保存在自己的磁盘上，用浏览器访问，可离线使用。

- 笔记按**仓库**（repo）组织，每个仓库是一棵可拖拽的树
- 正文是 **Markdown**，编辑时所见即所得（TipTap），存库前转回 Markdown
- 标签、全文搜索（SQLite FTS5）、附件（内容寻址 + 秒传）、回收站
- 前端为 PWA，可安装、可离线

## 技术栈

| 层 | 选型 |
|---|---|
| 客户端 | Vue 3 + TypeScript + Tailwind CSS v4 + TipTap（`client/`） |
| 服务端 | Go + gin + modernc sqlite（纯 Go 驱动，无需 cgo）（`server/`） |
| 存储 | 每个仓库一个自包含目录 `<data>/<uuid>/`，内含 `data.db` + `assets/` |

## 快速开始

需要 Go 1.26+、Node 20+、pnpm。

```bash
# 终端 1：后端，监听 :8080（必须在 server/ 里运行）
cd server && go run .

# 终端 2：前端，监听 :5173
cd client && pnpm install && pnpm dev
```

打开 http://localhost:5173 。细节（工作目录校验、代理、构建、验证方法）见 [docs/dev.md](docs/dev.md)。

## 配置

**没有配置文件、环境变量或命令行参数** —— 全部硬编码在 `server/config.go`（端口、数据目录、图标集来源与检查间隔等）。改配置 = 改常量 + 重启。完整清单见 [docs/api.md](docs/api.md#配置)。

## 目录结构

```
webnotes/
├── server/              Go 后端
│   ├── main.go          装配与启动
│   ├── config.go        全部配置常量
│   ├── api/             HTTP 路由与 handler
│   ├── store/           仓库索引与建库 schema
│   ├── asset/           附件文件与状态机
│   └── icons/           图标集本地副本与自动更新
├── client/              Vue 前端
│   ├── src/
│   │   ├── api.ts           HTTP 客户端
│   │   ├── components/      界面组件
│   │   ├── composables/     模块级共享状态
│   │   └── lib/             纯函数工具
│   └── public/          manifest 与 Service Worker
├── data/                运行时数据（gitignore）：仓库目录、repos.json、icons/
└── docs/                文档（见下）
```

## 文档

入口是 [docs/README.md](docs/README.md)。文档平铺在 `docs/` 下，权威性看每篇文首：

| 文档 | 定位 |
|---|---|
| [api.md](docs/api.md) | **权威** · HTTP 接口契约 |
| [model.md](docs/model.md) | **权威** · 数据模型与行为语义 |
| [schema.md](docs/schema.md) | **权威** · 数据库 schema 与磁盘布局 |
| [client.md](docs/client.md) | **权威** · 前端架构 |
| [dev.md](docs/dev.md) | 操作指南 · 本地开发、构建与验证 |
| [plan-deploy-layout.md](docs/plan-deploy-layout.md) | 草稿 · 打包布局设想 |
| [plan-ui.md](docs/plan-ui.md) | 草稿 · 界面规划 |

权威文档描述系统**当前的真实行为**；改代码时必须在同一个提交里同步对应文档。`status:` 与 `plan-` 前缀标识尚未落地的想法。

> ⚠️ **当前例外**：`schema.md`、`api.md`、`model.md` 已按新设计改完，但代码尚未跟进（字段改名、标签收敛成一列、路由改为只用 GET/POST），并且现在**编译不过**。在实现落地前，这三篇描述的是目标而非现状 —— 差异清单见 [docs/README.md](docs/README.md)。

## 许可

AGPL-3.0，见 [LICENSE](LICENSE)。
