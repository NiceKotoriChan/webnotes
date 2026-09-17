# 本地开发与验证

操作指南，不是规格：随环境变化更新。

## 运行

需要 Go 1.26+、Node 20+、pnpm。

```bash
cd server && go run .        # 后端 :8080，必须在 server/ 里运行
cd client && pnpm install && pnpm dev   # 前端 :5173
```

打开 http://localhost:5173 。Vite 把 `/api` 与 `/icons/mdi.json` 代理到 `:8080`（见 `client/vite.config.ts`），所以两端要同时跑。

VS Code 有三个任务：`dev: backend`、`dev: frontend`、`dev: all`。注意 `.vscode/` 被 `.gitignore` 忽略，别人的 clone 里没有这些任务 —— 要共享得先把它从 `.gitignore` 移除。

## 配置

**没有配置文件、环境变量或命令行参数** —— 全部硬编码在 `server/config.go`（`main.go` 会忽略并提示命令行参数）。改配置 = 改常量 + 重启。

| 常量 | 当前值 | 说明 |
|---|---|---|
| `Addr` | `:8080` | HTTP 监听地址。监听所有网卡是**有意如此**（局域网内其他设备可直接打开） |
| `DataDir` | `../data` | 仓库目录与 `repos.json` 的根，相对 `server/` 解析 |
| `IconDir` | `../data/icons` | 图标集存放目录 |
| `IconFile` | `mdi.json` | 对外地址 `/icons/mdi.json` |
| `IconCheckInterval` | `24h` | 图标集更新检查间隔 |
| `IconMinCount` | `1000` | 有效图标集的最小图标数，低于此值视为坏数据 |
| `IconSources` | 3 个镜像 | 按顺序尝试，第一个成功的即采用 |

其余固定行为直接写在代码里：列表 `limit` 默认 100 / 上限 1000、搜索 FTS 阈值 3 字符、图标集单次下载上限 64 MiB、图标集请求超时 60s、SQLite `MaxOpenConns(1)`、SQLite `busy_timeout` 5000ms。

## 两个必知的坑

**1. 后端必须在 `server/` 目录里启动。** `config.go` 的 `DataDir` / `IconDir` 是相对 `server/` 的路径（基准是 `go.mod` 所在处），`resolvePaths()` 会校验工作目录、不对就报错退出（故意的，免得静默在别处建出一个空 `data/`）。编译出的二进制也要从 `server/` 里起。

**2. 本机代理会劫持 127.0.0.1。** 本地验证一律加 `--noproxy '*'`：

```bash
curl --noproxy '*' -s http://localhost:8080/api/repos
```

本机 `/tmp` 是较小的 tmpfs，编译给个大盘目录：`TMPDIR=~/.cache/tmp go build ./...`

## 验证

```bash
cd server && go vet ./...            # 后端静态检查
cd client && pnpm check && pnpm build # 前端类型检查 + 生产构建
```

接口抽查：

```bash
curl --noproxy '*' -s  http://localhost:8080/api/repos       # 200 + []
curl --noproxy '*' -I  http://localhost:8080/icons/mdi.json  # 200 + ETag
```

把上一步的 ETag 带上再请求一次应当得到 `304`（图标集更新的核心链路，动过图标相关代码务必回归）：

```bash
curl --noproxy '*' -I -H 'If-None-Match: "<上一步的 ETag>"' http://localhost:8080/icons/mdi.json
```

> 这两条抽查当前**还跑不通** —— 后端编译不过（`main.go` 用四个参数调用 `api.NewRouter`，签名只收两个），`/icons/mdi.json` 这条路由也还没注册。

## 构建

```bash
cd client && pnpm build              # 产物在 client/dist/（已 gitignore）
cd server && go build -o webnotes .
```

**后端不托管前端静态文件**，`GET /` 由前端 dev server 或外部静态服务器提供。

## 数据目录

`data/` 全是运行时数据（已 gitignore）：`repos.json`（索引，可重建）、`icons/`（可删，启动会重下）、`<uuid>/`（每个仓库一个目录，含 `data.db` 与 `assets/`）。

- **重置**：删掉 `data/` 或某个 UUID 目录即可，`repos.json` 下次启动自动对账重建。
- **本项目不做 schema 迁移** —— 改表结构就是删库重建，见 [schema.md](schema.md)。
- 图标集当前状态看 `data/icons/mdi.json.meta.json`（来源、ETag、sha256、图标数、更新时间），启动日志里也有。
