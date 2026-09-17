# 文档索引

文档平铺在本目录下，不分层。**权威性写在每篇文首**，靠这两条一眼可辨：

- 文首有「**与代码同步**」的 = 权威文档，描述系统真实行为，改代码要同时改它；
- 文件名以 `plan-` 开头、文首有 `status:` 的 = 设计草稿，**不保证与实现一致**。

## ⚠️ 当前状态：文档领先于代码

[api.md](api.md)、[model.md](model.md)、[schema.md](schema.md) 三篇已经按新设计改完，但**实现还没跟上**。当前代码与文档的差异至少有这些：

| | 文档（新设计） | 代码（现状） |
|---|---|---|
| 时间字段 | `created_at` + `updated_at` | 单个 `date` |
| 正文字段 | `content` | `data` |
| 标签 | `notes.tags` 一列 JSON | `tags` 表 + `note_tags` 表 |
| 请求方法 | 只用 GET / POST，19 条路由 | `GET`/`POST`/`PUT`/`PATCH`/`DELETE`/`HEAD`，25 条 |

而且代码目前**编译不过** —— `main.go` 用四个参数调用 `api.NewRouter`，而它的签名只收两个（图标集改由后端托管那次重构做了一半）。所以这三篇暂时是"目标"而非"现状"。实现落地后删掉本节。

## 权威文档（与代码同步）

| 文档 | 内容 | 真相源 |
|---|---|---|
| [api.md](api.md) | HTTP 接口契约：方法、路径、请求体、成功/失败码、静态资源与配置清单 | `server/api/router.go` |
| [model.md](model.md) | 实体字段与**行为语义**（排序、软删、递归、防环、状态机） | `server/api/*.go` 的 struct、`server/store/store.go` |
| [schema.md](schema.md) | 表结构、索引、FTS、建库策略与磁盘布局 | **本文自身**（见下面第 2 条规矩） |
| [client.md](client.md) | 前端分层、数据流、编辑器与 PWA 机制、快捷键 | `client/src/` |
| [dev.md](dev.md) | 环境要求、运行与调试、两个必知的环境坑、验证方法、构建、数据目录 | 无（操作指南，随环境漂移） |

## 设计草稿（不保证与实现一致）

| 文档 | status | 内容 |
|---|---|---|
| [plan-deploy-layout.md](plan-deploy-layout.md) | 未实现 | 安装与打包布局设想（`/usr/share/webnotes`），含与现状的差异对照 |
| [plan-ui.md](plan-ui.md) | 部分未实现 | 活动栏按钮与各面板的规划，含"规划 vs 现状"对照 |

## 写文档的规矩

1. **文档里出现的文件路径、函数名、路由、字段名必须真实存在。** 提交前核对一遍 —— 本项目历史上出现过指向 `src/components/WysiwygEditor.svelte`、`docs/data.sql` 这类已不存在目标的引用，它们比没有文档更误导人。
2. **谁是真值源，写在文首，别靠猜。** [api.md](api.md) 与 [model.md](model.md) 以代码为准；[schema.md](schema.md) 反过来 —— 它是 schema 的真值源，`store.RepoSchema` 是它的实现版本（只多了 `IF NOT EXISTS`）。改 schema 先改文档再改常量。
3. **权威文档只写"是什么"，不写"打算怎么做"。** 后者写成 `plan-*.md`。
4. **草稿一旦落地，内容并入权威文档并删除原文件** —— 不要同一件事留两份。
5. 新增一篇文档前，先确认它不是已有文档的一节。
