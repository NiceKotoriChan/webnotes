# 本地开发与验证

> 操作指南，不是规格：随环境变化更新，不承诺与代码逐行对应。

## 环境要求

| 依赖 | 版本 |
|---|---|
| Go | 1.26+（见 `server/go.mod`） |
| Node | 20+ |
| pnpm | 较新版本即可（仓库使用 `pnpm-lock.yaml`） |

## 运行

```bash
# 终端 1：后端，监听 :8080
cd server && go run .

# 终端 2：前端，监听 :5173
cd client && pnpm install && pnpm dev
```

打开 http://localhost:5173 。Vite 会把 `/api` 与 `/icons/mdi.json` 代理到 `http://localhost:8080`（见 `client/vite.config.ts`），所以两端必须同时运行。

VS Code 里准备了三个任务，可直接从命令面板运行：

| 任务 | 作用 |
|---|---|
| `dev: backend` | `go run .`，cwd 为 `server/` |
| `dev: frontend` | `pnpm dev`，cwd 为 `client/` |
| `dev: all` | 并行启动上面两个 |

> `.vscode/` 目前被 `.gitignore` 忽略，所以这几个任务在别人的 clone 里不存在。要共享需要把它从 `.gitignore` 移除。

## 两个必须知道的坑

### 1. 后端必须在 `server/` 目录里启动

`server/config.go` 里的 `DataDir` / `IconDir` 是**相对 `server/`** 的相对路径（`../data`、`../data/icons`），基准是 `go.mod` 所在目录。`resolvePaths()` 会校验工作目录，目录不对时直接报错退出 —— 这是故意的，避免静默在别处建出一个空 `data/` 来。

所以：`cd server && go run .`，编译出的二进制也要从 `server/` 里启动。

### 2. 本机代理会劫持 127.0.0.1

本地验证一律加 `--noproxy '*'`：

```bash
curl --noproxy '*' -s http://localhost:8080/api/repos
```

另外编译中间产物较大时注意 `/tmp` 容量（本机 `/tmp` 是较小的 tmpfs）：

```bash
TMPDIR=~/.cache/tmp go build ./...
```

## 验证

```bash
# 后端：编译 + 静态检查
cd server && go vet ./...

# 前端：类型检查 + 生产构建
cd client && pnpm check && pnpm build
```

接口抽查：

```bash
curl --noproxy '*' -s   http://localhost:8080/api/repos                  # 期望 200 + []
curl --noproxy '*' -I   http://localhost:8080/icons/mdi.json             # 期望 200 + ETag
```

把上一步拿到的 ETag 带上再请求一次，应当得到 `304`（这条链路是图标集更新的核心，改动图标相关代码后务必回归）：

```bash
curl --noproxy '*' -I -H 'If-None-Match: "<上一步的 ETag>"' http://localhost:8080/icons/mdi.json
```

接口契约见 [api.md](api.md)。

## 构建

```bash
cd client && pnpm build      # 产物在 client/dist/（已在 .gitignore 中）
cd server && go build -o webnotes .
```

**后端目前不托管前端静态文件**，`GET /` 由前端 dev server 或外部静态服务器提供。"把前端打包进后端一起分发"的设想记录在 [plan-deploy-layout.md](plan-deploy-layout.md)，尚未实现。

## 数据目录

运行时数据都在 `data/`（已 gitignore，与源码分离）：

```
data/
├── repos.json              仓库索引（可重建，非真相源）
├── icons/                  mdi.json + mdi.json.meta.json（可删，启动会重下）
└── <uuid>/                 每个仓库一个目录
    ├── data.db             SQLite
    └── assets/             附件（tmp/ 与 ab/cd/<sha>）
```

- **重置**：删掉 `data/` 或其中某个 UUID 目录即可，`repos.json` 下次启动会自动对账重建。
- **看图标集状态**：`data/icons/mdi.json.meta.json` 记录了当前副本的来源、ETag、sha256、图标数与更新时间；启动日志里也会打一行。
- 目录结构与建库 schema 见 [schema.md](schema.md)。**本项目不做 schema 迁移** —— 改表结构就是删库重建，见该文的「建库：不提供迁移」。
