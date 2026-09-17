---
status: 未实现
---

# 安装与打包布局设想

> **这是设计草稿，不是现状。** 原文件名为 `docs/add.md`，正文只有一段布局草图；保留下来是因为它表达了对最终发布形态的意图，落地后应并入 `docs/spec/` 并删除本篇。
> 当前的目录布局与运行方式见根 `README.md` 与 [`schema.md`](schema.md)。

## 设想

设想安装到 `/usr/share/webnotes`，整体布局为：

```
/usr/share/webnotes
├── web/      存放前端静态代码
├── data/     存放笔记数据
└── icon/
    ├── mdi.json
    └── auto_mdi.json     记录默认与自定义图标的映射关系
```

## 与现状的差异

| 设想 | 现状 |
|---|---|
| 安装在 `/usr/share/webnotes` | 直接在仓库目录里运行，数据在相对 `server/` 的 `../data`（见 `server/config.go`） |
| 后端托管 `web/` 并提供 `GET /` | **后端不管静态文件**，`GET /` 目前由前端 dev server 或外部静态服务器提供 |
| `icon/auto_mdi.json` 持久化图标映射 | 图标映射没有持久化，在前端 `client/src/lib/autoIcon.ts` 里以规则表（正则 → 图标名）实现；只有用户在笔记上手动选定的图标会存进 `notes.icon` |
| `data/` 平铺 | 已实现且更细：每仓库一个 UUID 自包含目录，外加 `repos.json` 索引 |
| 未提及图标集更新 | 已实现：后端托管 `mdi.json`，启动下载 + 每 24h 检查更新，见 [`api.md`](api.md) |

## 落地时需要决定的事

- 前端产物（`client/dist/`）是由后端直接托管，还是继续分离部署？
- 安装到系统目录后，`DataDir` 指向哪里？（当前是硬编码的相对路径 + 工作目录校验，安装场景需要换一种解析方式）
- 是否需要卸载脚本与数据保留/清理策略？
