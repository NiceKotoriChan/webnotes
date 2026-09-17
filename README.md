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
# 终端 1：后端 :8080（必须在 server/ 里运行）
cd server && go run .

# 终端 2：前端 :5173
cd client && pnpm install && pnpm dev
```

打开 http://localhost:5173 。运行细节与两个必知的环境坑见 [docs/dev.md](docs/dev.md)。

## 配置

**没有配置文件、环境变量或命令行参数** —— 全部硬编码在 `server/config.go`。改配置 = 改常量 + 重启。清单见 [docs/dev.md](docs/dev.md#配置)。

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
│   └── src/             api.ts / components/ / composables/ / lib/
├── data/                运行时数据（gitignore）：仓库目录、repos.json、icons/
├── docs/                文档（见下）
└── AGENTS.md            给 AI 助手的项目规则
```

## 文档

`docs/` 一篇文章只讲一层，按设计层次命名；权威性看文首。

| 文档 | 层次 | 内容 |
|---|---|---|
| [schema.md](docs/schema.md) | 数据库逻辑 | 表结构、列、索引、FTS 与触发器、标签改写语句、建库策略（**schema 的真相源**） |
| [model.md](docs/model.md) | 后端对象逻辑 | `Repo` / `Note` / `AssetMeta` 与行为语义：软删递归、还原、防环、时间戳刷新、附件状态机 |
| [api.md](docs/api.md) | 接口逻辑 | HTTP 契约：只用 GET / POST 的 19 条路由、错误码、查询参数、静态资源 |
| [ui.md](docs/ui.md) | 视图逻辑 | 前端分层、数据流、编辑器与 PWA 机制、快捷键 |
| [dev.md](docs/dev.md) | 操作指南 | 运行、配置清单、两个必知的环境坑、验证与构建、数据目录 |

> ⚠️ **当前状态：文档领先于代码。** `schema.md` / `model.md` / `api.md` 描述的是已定稿的新设计，代码尚未跟进（时间字段拆 `created_at` + `updated_at`、`data` → `content`、标签从两张表收敛成 `notes.tags` 一列、路由改为只用 GET/POST 的 19 条、删掉 `migrateRepo`），而且现在**编译不过**（`main.go` 用四个参数调用 `api.NewRouter`，签名只收两个）。落地后删掉本节。[ui.md](docs/ui.md) 与 [dev.md](docs/dev.md) 描述当前状态，落地时要一起改。

## 许可

AGPL-3.0，见 [LICENSE](LICENSE)。
