# API 契约

> **目标设计，代码尚未跟进**（现状见根 [README](../README.md)）。真相源将是 `server/api/router.go` 的路由表；改路由要在同一个提交里改本文。

Base path `/api`，**19 条路由，方法只有 `GET` 与 `POST`**。响应一律 JSON，错误形如 `{ "error": "<msg>" }`。时间戳 Unix 毫秒；字段 snake_case；无鉴权、无版本前缀。

**GET 只读，POST 是唯一写入口** —— 浏览器会预取、缓存、重试 GET，只要 GET 不写数据，这些行为就永远不会误触发一次修改。POST 按路径形状分三种：

| 形态 | 例 | 语义 | 成功码 |
|---|---|---|---|
| 集合路径 | `POST /notes` | 新建 | 201 + 新资源 |
| 资源路径 | `POST /notes/:id` | 部分更新 | 200 + 新状态 |
| 资源 + 动词 | `POST /notes/:id/delete` | 动作（动词只有 `delete` / `restore`） | 200 + 回执 |

这样客户端能写的字段被封闭在 `title` / `content` / `parent_id` / `tags` / `icon` 五个里，写不了 `created_at` / `updated_at` / `deleted_at`。

约定：路径参数写法与 `router.go` 逐字一致（`:repo` / `:id` / `:sha`），可以互相 grep；参数放路径或 JSON body，只有列表筛选与附件 `?inline=1` 用 query；**POST 一律回 JSON body，没有 204**，删除类动作回 `{ "id": ... }` 作回执；列表返回**裸数组**，不套 `{ items, total }` 信封。

## 错误码

| 码 | 何时出现 |
|---|---|
| 400 | body 不是合法 JSON；没有任何可更新字段；`title` / `content` 传 null；`parent_id` 不存在或成环；`tags` 不是字符串数组；`:sha` 非法；`X-Size` 为负；上传时 sha256 校验失败 |
| 404 | 仓库 / 笔记 / 标签 / 待还原笔记不存在；附件未 `ready` |
| 409 | 附件正在被另一个请求上传；删除非 `ready` 的附件 |
| 500 | 其余。**对外只回 `internal error`**，SQL 原文只进服务端日志 |

## Repos

| 方法 | 路径 | 请求体 | 成功 |
|---|---|---|---|
| GET | `/api/repos` | — | 200 `Repo[]` |
| POST | `/api/repos` | `{ name }` | 201 `Repo` |
| POST | `/api/repos/:repo` | `{ name }` | 200 `Repo` |
| POST | `/api/repos/:repo/delete` | — | 200 `{ id }` |

`Repo` = `{ id, name, date }`。`id` 由服务端生成 UUID，同时是磁盘目录名，**不可变**；改名只改 `repos.json`，目录与既有链接不动。

## Notes

| 方法 | 路径 | 请求体 | 成功 |
|---|---|---|---|
| GET | `/api/repos/:repo/notes` | — | 200 `Note[]` |
| POST | `/api/repos/:repo/notes` | `{ title?, content?, parent_id?, tags?, icon? }` | 201 `Note` |
| GET | `/api/repos/:repo/notes/:id` | — | 200 `Note` |
| POST | `/api/repos/:repo/notes/:id` | `{ title?, content?, parent_id?, tags?, icon? }` | 200 `Note` |
| POST | `/api/repos/:repo/notes/:id/delete` | `{ permanent? }` | 200 `{ id }` |
| POST | `/api/repos/:repo/notes/:id/restore` | — | 200 `Note` |
| GET | `/api/repos/:repo/trash` | — | 200 `Note[]` |

`Note` = `{ id, parent_id, title, content, created_at, updated_at, deleted_at, tags, icon }`，`tags` 恒为数组。创建时未给的字段取默认值：`title` / `content` 空串，`parent_id` 根，`tags` `[]`，`icon` null；`created_at` 与 `updated_at` 都取当前时间。

### POST /notes/:id 部分更新

这一条取代了原先的 `PUT` + `PATCH` + `PATCH /icon` 三条 —— 改同一张表的字段不该有三种形状。

- **body 里出现的字段才改，没出现的字段一律不动。** 客户端不必"补全自己没改的字段"。
- 原先的 `PUT` 漏传哪个字段就把哪个写成空，而正文保存是 600ms 防抖触发的 —— 只发 `title` 就会清空正文。改成部分更新后，这个坑不可能再踩。
- 字段的 null 语义各不相同，实现必须能区分**字段缺席**与**显式 null**：

| 字段 | 缺席 | 显式 null |
|---|---|---|
| `title` / `content` | 不动 | 400 |
| `parent_id` | 不动 | 移到根 |
| `tags` | 不动 | 等价 `[]`（清空） |
| `icon` | 不动 | 清空，恢复前端自动匹配 |

- 实现用 `map[string]json.RawMessage` 逐 key 分派。**不能用带 `*string` 字段的结构体** —— 指针分不出"没传"和"传了 null"，而这正是 `parent_id: null` 与 `icon: null` 的语义。
- 只有 `parent_id` 出现且非 null 时才做防环检查。一个可更新字段都没出现 → 400。
- `tags` 先规范化：trim、丢空串、去重，再序列化成 JSON 数组（空集写 `[]`）。

### updated_at 只在内容变更时刷新

`title` / `content` / `tags` 会刷新；`parent_id`（移动）与 `icon`（换图标）**不刷新** —— 它们是结构与展示层面的动作。否则在按 `updated_at` 排序的视图里，一次拖拽就会让笔记跳位，而用户并不认为自己改了这篇笔记。

### 列表查询参数

| 参数 | 说明 |
|---|---|
| `q` | 匹配 `title` + `content`。**≥3 字符**走 FTS5 trigram 并按 `rank` 排；**<3 字符降级 `LIKE`**（trigram 对 2 字符返回空结果且不报错，必须自己降级），此时改按排序字段排 |
| `tag` | 只返回带该标签的笔记。**精确匹配**（`ru` 不命中 `rust`），实现是 `json_each` + `EXISTS`，全表扫 |
| `parent_id` | 不传 = 全部未删笔记；传**空串** = 只根节点（落成 `parent_id IS NULL`）；传 id = 该节点的**直接**子节点 |
| `sort` | `created`（默认）或 `updated` |
| `limit` / `offset` | 默认 `100` / `0`，`limit` 上限 `1000`，越界或非正数回落为 100 |

三个筛选参数**叠加关系是 AND**，基集是 `deleted_at IS NULL`。排序一律带兜底列：

```sql
ORDER BY created_at DESC, id ASC   -- sort=created（默认）
ORDER BY updated_at DESC, id ASC   -- sort=updated
```

### delete / restore / trash

- `delete`：默认**软删整棵子树**（递归标 `deleted_at`）；`{ "permanent": true }` 则硬删整棵子树，不可恢复。都返回 `{ "id" }`；目标不存在（含对已软删笔记再次软删）返回 404。
- `restore`：还原该笔记**及其整棵子树**。若原父节点仍在删除状态，该笔记挂到根 —— 避免"父在回收站、子在树里"的悬空状态。不在回收站里返回 404。返回还原后的 `Note`，**不刷新** `updated_at`。
- `GET /trash`：只列"顶层"被删笔记（自身已删且父节点未删），按 `deleted_at DESC`。子节点随父一起还原，不单独出现。

## Tags

**标签是 `notes.tags` 里的字符串，不是实体**（见 [model.md](model.md)）。三条前提，读下面的接口时别指望它们是别的东西：

- **没有"新建标签"接口** —— 创建标签就是给某篇笔记写上这个字符串。
- **`GET /tags` 是派生查询，不是真值源。** 它只把未删笔记的标签扫出去重，所以**没有任何笔记在用的标签不会出现在结果里**。
- **调用方可以自己维护一份。** 它是可再生的派生结果，不是需要保持同步的状态 —— 本地解析、需要时再拉一次即可。

| 方法 | 路径 | 请求体 | 成功 |
|---|---|---|---|
| GET | `/api/repos/:repo/tags` | — | 200 `string[]` |
| POST | `/api/repos/:repo/tags/rename` | `{ from, to }` | 200 `{ count }` |
| POST | `/api/repos/:repo/tags/delete` | `{ name }` | 200 `{ count }` |

- 改名与删除是**全表改写**（语句见 [schema.md](schema.md)），`count` 就是受影响的笔记数；**两者都会刷新受影响笔记的 `updated_at`** —— 标签属于内容，与改某篇笔记的 `tags` 是同一种变更，不该因为走哪条路由而给出不同答案。
- `from` / `name` 不存在 → 404，判据是"没有任何未删笔记在用这个标签"（因为没有标签表）。
- `rename` 的 `to` 已存在时**两个标签合并**，不报错、不返回 409。
- 已软删的笔记不参与改写，也不出现在 `GET` 结果里，还原后标签原样回来。
- `GET` 按名升序，排序是**字节序**，中文标签不是拼音序。

## Assets

| 方法 | 路径 | 请求体 | 成功 |
|---|---|---|---|
| GET | `/api/repos/:repo/assets` | — | 200 `AssetMeta[]`（仅 ready，按 date 倒序） |
| GET | `/api/repos/:repo/assets/:sha` | — | 附件本体 |
| GET | `/api/repos/:repo/assets/:sha/meta` | — | 200 `AssetMeta` / 404 |
| POST | `/api/repos/:repo/assets/:sha` | 原始字节 | 201 `AssetMeta`（新上传）/ 200（已存在） |
| POST | `/api/repos/:repo/assets/:sha/delete` | — | 200 `{ id }` |

`AssetMeta` = `{ id, name, mime, size, date }`，`id` 即 sha256。

**为什么用 `meta` 而不是 `HEAD`**：`HEAD` 只有一个状态码，客户端拿不到真实元数据，于是秒传分支只能自己拼一个 `AssetMeta`（拿本地文件名冒充服务端记录），造成界面显示的文件名与库里记录不一致。`GET .../meta` 一次解决两件事 —— 404 = 没有；200 + 库里的权威元数据 = 有。

**上传**是幂等的：body 为原始字节，元信息走请求头 `X-Name`（必传，也用于按扩展名推断 MIME）、`X-Mime`（留空则按扩展名推断，再退回 `application/octet-stream`）、`X-Size`（负值 400）。

1. 该 sha 已 `ready` → 直接 **200 + 已有元数据**（秒传，一个字节都不传）
2. 插入 `status='uploading'` 行；主键冲突说明另一个请求正在上传 → **409**
3. 流式写入 `assets/tmp/<sha>` 并边写边算 sha256；通过则原子 `rename` 到 `assets/ab/cd/<sha>` 并置 `ready` → **201 + 元数据**；失败则删 tmp 与记录行 → **400**

第 1 步和第 3 步返回的都是服务端的真实元数据，客户端不需要拼任何东西。

**下载**：`?inline=1` 返回 `Content-Disposition: inline`（供正文内嵌图片），否则为 `attachment`，文件名按 RFC 5987 编码 UTF-8。

## 静态资源

| 方法 | 路径 | 说明 |
|---|---|---|
| GET / HEAD | `/icons/mdi.json` | 全量 MDI 图标集（JSON）。由 `main.go` 直接挂在 engine 上，**不在 `api` 包里**（`api` 只注册 `/api` 下的路由） |

图标集**不做"有效/无效"划分**，全量一次性下发，由前端缓存并离线可用。后端持有本地副本 `data/icons/mdi.json`：启动时仅在缺失时下载一次，之后每 24h 后台检查。更新遵循"拿到更好的才替换" —— 按 `IconSources` 顺序尝试，校验为合法 JSON 且图标数 ≥ `IconMinCount` 才接受，内容有变化才原子替换；任何一步失败都保留现有副本。响应带 `ETag`（内容 sha256）与 `Cache-Control: no-cache`，命中 `If-None-Match` 回 304。

附件下载见上文 Assets 的 `GET /api/repos/:repo/assets/:sha`。**前端静态文件不由后端托管** —— `GET /` 由前端 dev server 或外部静态服务器提供。
