# webnotes 项目长期备忘

- 技术栈：客户端 Vue 3 + Tailwind v4 + Tiptap（`client/`），服务端 Go + gin + modernc sqlite（`server/`）；每个仓库一个目录 `<data>/<uuid>/`，内含 `data.db` + `assets/`。
- **配置一律硬编码**：全部集中在 `server/config.go`（端口、数据目录、图标集目录/文件名、检查间隔、下载源）。**不使用** .env / 环境变量 / 命令行参数。改配置 = 改常量 + 重启。
- 路径约定：`DataDir`/`IconDir` 是**相对 `server/`** 的相对路径（`../data`、`../data/icons`），因为 go.mod 在 server/。相对路径启动时会校验工作目录，必须在 server/ 里运行（`go run .` / VS Code `dev: backend` 任务 cwd=server）。
- 图标集由**后端托管**：`GET|HEAD /icons/mdi.json`，本地副本 `data/icons/mdi.json`（可删，启动会重下），启动查一次 + 每 24h 一次；前端不再自带 mdi.json，dev 靠 vite proxy 转发该路径，SW 对它走网络优先。
- 文档分工：`docs/api.md`（接口 + 配置表）、`docs/model.md`（数据模型）、`docs/schema.md`（SQL + 目录结构）、`docs/client.md`（客户端；其中 Svelte 5 / `.svelte` 的描述与现状不符，实为 Vue + `.vue`，待改）。
- 命名/风格：时间戳一律 Unix ms；JSON 字段 snake_case；注释与日志用中文；迁移逻辑要幂等（`store.migrateRepo`）。
- 环境注意：本机 /tmp 是 10 MB tmpfs，`go build`/`go run` 需 `TMPDIR=<大盘目录>`；本地服务验证用 `curl --noproxy '*'`（主机代理会劫持 127.0.0.1）；`rm` 会把文件移进仓库根的 `.Trash-0/`（已 gitignore）。
