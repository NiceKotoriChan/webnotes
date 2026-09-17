# webnotes 项目长期备忘

- 技术栈：客户端 Vue 3 + Tailwind v4 + Tiptap（`client/`），服务端 Go + gin + modernc sqlite（`server/`）；每个仓库一个目录 `<data>/<uuid>/`，内含 `data.db` + `assets/`。
- **配置一律硬编码**在 `server/config.go`（端口、数据目录、图标集、检查间隔、下载源）；不用 .env / 环境变量 / 命令行参数。`DataDir`/`IconDir` 是**相对 `server/`** 的相对路径（`../data`），启动会校验工作目录，必须在 `server/` 里运行。
- 图标集由**后端托管**：`GET|HEAD /icons/mdi.json`，本地副本 `data/icons/mdi.json`（可删，启动重下），启动查一次 + 每 24h 一次。该路由由 `main.go` 挂 `gin.WrapH`，**不属于 api 包**。
- **docs 平铺在 `docs/`，不分层**（用户明确要求，曾按 spec/guide/notes 分三层已拆掉）。权威性写在**文首**：「与代码同步」= 权威；`plan-*.md` + `status:` = 草稿。索引在根 `README.md` 与 `docs/README.md`，5 条写文档规矩在后者。
- **schema 的真值源是 `docs/schema.md` 本身**（2026-09-17 用户手工替换，是唯一「文档为准、代码跟随」的一篇）；`store.RepoSchema` 是它的实现，只多 `IF NOT EXISTS`。其余文档以代码为准。
- 写文档铁律：引用文件写**仓库相对全路径**（`client/src/api.ts`）；列出的路径必须真实存在 —— 历史上出现过 `WysiwygEditor.svelte`、`docs/data.sql` 这类悬空引用，比没有文档更误导。
- 命名/风格：时间戳一律 Unix ms；JSON 字段 snake_case；注释与日志用中文；迁移必须幂等。

## 新 schema（已定稿，代码未跟进）

`notes`：`id, parent_id, title, content, created_at, updated_at, deleted_at, tags, icon`。要点：

1. **`data` → `content`**、**`date` → `created_at` + `updated_at`**（旧 date 回填 created_at）。
2. **标签不再是实体**（用户 2026-09-17 确认按字面，**不单独建表、不要 id**）：删掉 `tags` 与 `note_tags` 两张表，收敛成 `notes.tags` 一个 **JSON 数组**列。代价是标签无 id、重命名/删除要**全表改写**、按标签筛选全表扫；得到的是不再需要关联表。已验证：`json_each` 可用、`json_each(NULL)` 不报错、`json_group_array` 空集返回 `'[]'`、匹配是精确的。
3. `assets` 不变（`id, date, name, mime, size, status`），`date` 是唯一还叫 date 的时间列 —— **有意保留，不统一成 `created_at`**（用户 2026-09-17 定；附件不存在创建/修改之分）。
4. 索引 `idx_notes_parent` + `idx_notes_created`；所有列表必须 `ORDER BY <字段> DESC, id ASC`（兜底列，否则同毫秒笔记顺序会跳）。
5. 触发器 `notes_au` 用 **`AFTER UPDATE OF title, content`**，避免只改标签/刷 updated_at/标软删时白重建 FTS。已实测验证。
6. **不做 schema 迁移**（用户 2026-09-17 定）：落地时直接删掉 `data/` 下的仓库目录重建（本轮丢弃 2 仓库 / 23 笔记 / 1 附件），`schema.md` 的迁移章已改成「建库：不提供迁移」。`store.migrateRepo` **整段删除** —— 它里面 `columnExists(notes,"content")` → 把 content 改回 data 的逻辑在新 schema 下**判据恒真**，留着会把新库改坏。代价：以后改 schema 老库不自动升级，手动处理。

## API：只用 GET + POST（用户定的）

规则：**GET 只读；POST 唯一写入口**，三种形态 —— 集合路径=新建(201)、资源路径=修改(200)、资源路径+动词=动作(200)。动词是封闭集合 `{delete, restore}`。**19 条路由**（原 25 条），**取消 204**（POST 一律回 JSON，删除类回 `{"id":...}`）。详见 `docs/api.md`。

1. notes 三条更新路由合并成 `POST /notes/:id` 部分更新，body 白名单 `{title,content,parent_id,tags,icon}`，用 `map[string]json.RawMessage` 判 key 存在性（`*string` 分不出「缺席」与「显式 null」）。`title/content` 传 null → 400；`parent_id:null` = 移到根；`tags:null` = 清空；`icon:null` = 清图标。
2. `updated_at` 只在 `title`/`content`/`tags` 变更时刷新；`parent_id`/`icon` 不刷新（拖拽、换图标不该让笔记在 updated 排序里跳位）。
3. `HEAD /assets/:sha` → **`GET /assets/:sha/meta`**（404=没有，200=真实元数据），顺带修掉客户端秒传时拼假 AssetMeta 的问题。
4. 标签三条：`GET /tags`（distinct 派生）+ `POST /tags/rename` + `POST /tags/delete`；改名撞名即合并，不报 409。
5. engine 归 `main`：`api.NewRouter(s,a)` → **`api.Register(r gin.IRouter, s, a)`**，只注册 `/api`、不 import icons。**这一改顺带修掉 P0 编译错误**（`main.go:59` 四参 vs `router.go:20` 两参）。备选：保留 `NewRouter(s,a)` 名字 + main 补两行。
6. 新增 `server/api/respond.go`：`fail(c,err)`（500，细节只进日志）+ `failMsg` + `bind`，消灭约 40 处样板并停止回显 SQL 原文。
7. 取消 204 后 `client/src/api.ts` 的 `request()` 少一条分支。
8. **`GET /tags` 是派生查询，不是真值源** —— 真值在每条笔记的 `tags` 里，该接口只把未删笔记的标签扫出去重。所以**调用方可以自己维护一份**（用户原话：标签解析不用每次都请求后端，前端可自己维护；以后可能加保存按钮/重新拉取 —— 前端的事本轮不做）。**没有任何笔记在用的标签不存在于结果里**，因此没有「新建标签」和「清理未使用标签」这两件事。
9. **标签改名/删除是两条全表改写 UPDATE，且必须同时写 `updated_at`**（我拍板：`tags` 属于内容，不能因为走哪条路由就和改单篇笔记的 `tags` 给出不同答案）；`count` = `RowsAffected`，0 行 → 404；只作用于未软删笔记；`json_group_array(DISTINCT v)` 负责「撞名即合并」；这两条 UPDATE **不触发 `notes_au`**（已实测）。
10. `?sort=updated` 用户说**暂时不管**（文档保留，前端如用需加 `idx_notes_updated`）。

## 明确不做（别回头再提）

不加 `/api/v1`；不抽 `internal/repo` 层（同包内合并 `scanNotes`/`fetchNote` → `scanNote` 即可）；不加鉴权。**已定：`Addr` 保持 `:8080`（监听所有网卡是有意的，不收紧到 `127.0.0.1`）；`assets.date` 保持 `date`；不做 schema 迁移。**

## 现状与踩坑

- **代码编译不过**（2026-09-17，未修）：`server/main.go:59` 四参调用 `api.NewRouter`，签名只收两参。根因是图标集托管重构只做了一半。文档已领先于代码，差异清单在 `docs/README.md`。
- 环境：本机 /tmp 是 10 MB tmpfs，`go build`/`go run` 要 `TMPDIR=/home/kotori/.cache/tmp`；本地验证用 `curl --noproxy '*'`（主机代理劫持 127.0.0.1）；`rm` 会把文件移进仓库根 `.Trash-0/`（已 gitignore）。
- 沙箱会把**被预览过的文件**单独 bind-mount，导致该文件无法 `mv`（Device or resource busy）——遇到就 Write 重建 + 删原文件。同一文件的多个 Edit 不能并行发（后写的基于旧快照，会覆盖前一处）。
