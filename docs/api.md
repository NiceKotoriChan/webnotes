# API 契约

> **与代码同步** —— 真相源是 `server/api/router.go` 里的路由表。改路由必须在同一个提交里改本文。

Base path `/api`，共 **19 条**路由，请求方法只有 `GET` 与 `POST`（为什么这么定见下节）。所有响应体为 JSON，错误统一为 `{ "error": "<msg>" }`。时间戳一律 Unix 毫秒；JSON 字段 snake_case；无鉴权、无版本前缀。

## 为什么只有 GET 和 POST

一句话：**GET 只读，POST 是唯一的写入口。**

GET 必须无副作用 —— 浏览器会预取、缓存、重试 GET，代理也会。只要 GET 不写数据，这些行为就永远不会误触发一次修改。POST 承担全部写入，按路径形状分三种：

| 形态 | 例子 | 语义 | 成功码 |
|---|---|---|---|
| 集合路径 | `POST /notes` | 新建一个 | 201 + 新资源 |
| 资源路径 | `POST /notes/:id` | 修改该资源（部分字段） | 200 + 新状态 |
| 资源路径 + 动词 | `POST /notes/:id/delete` | 不可归约为字段修改的动作 | 200 + 回执 |

动词是**封闭集合**，只有 `delete` 与 `restore` 两个。路径最后一段是动词，就说明这不是一次普通更新。

这么定还有个实际收益：能被客户端写入的字段被封闭在 `title` / `content` / `parent_id` / `tags` / `icon` 五个里，客户端**无法**自己去写 `created_at`、`updated_at`、`deleted_at`。

## 约定

- 路径参数沿用 gin 的写法：`:repo` = 仓库 UUID，`:id` = 笔记 UUID，`:sha` = 附件 sha256。**与 `server/api/router.go` 的写法逐字一致**，所以可以互相 grep。
- 参数一律放**路径**或 **JSON body**。只有三处例外使用 query，且都不是业务参数：笔记列表的筛选参数、附件下载的 `?inline=1`。
- **POST 一律返回 JSON body，没有 204。** 删除类动作回 `{ "id": "..." }` 作为回执。客户端的请求封装因此只有一条解析路径，不必为"无响应体"单开分支。
- 列表接口直接返回**裸数组**（`Note[]` / `Repo[]` / `string[]` / `AssetMeta[]`），不套 `{ items, total }` 信封。

## 错误语义

| 码 | 何时出现 |
|---|---|
| 400 | 请求体不是合法 JSON；没有任何可更新字段；`title`/`content` 传 null；`parent_id` 指向不存在的笔记；把笔记移到自身或自己的后代（成环）；`tags` 不是字符串数组；`:sha` 不是合法 sha256；`X-Size` 为负；附件流式校验 sha256 失败 |
| 404 | 仓库不存在（`:repo` 不是已知 UUID 或目录已删）；笔记不存在（含已软删除的）；标签名不存在；回收站里找不到待还原的笔记；附件未 `ready` |
| 409 | 该附件正在被另一个请求上传；删除非 `ready` 状态的附件 |
| 500 | 其余内部错误。**对外只回 `internal error`**，SQL 原文只进服务端日志 |

## Repos

| 方法 | 路径 | 请求体 | 成功 |
|---|---|---|---|
| GET | `/api/repos` | — | 200 `Repo[]` |
| POST | `/api/repos` | `{ name }` | 201 `Repo` |
| POST | `/api/repos/:repo` | `{ name }` | 200 `Repo` |
| POST | `/api/repos/:repo/delete` | — | 200 `{ id }` |

`Repo` = `{ id, name, date }`（见 [model.md](model.md)）。`id` 由服务端生成 UUID，同时是磁盘目录名，**不可变**；改名只改 `repos.json` 里的 `name`，目录与既有链接不动。

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

`Note` = `{ id, parent_id, title, content, created_at, updated_at, deleted_at, tags, icon }`（见 [model.md](model.md)）。`tags` 恒为数组，无标签时是 `[]`。

创建时未提供的字段取默认值：`title` / `content` 为空串，`parent_id` 为根，`tags` 为 `[]`，`icon` 为 null（由前端按标题自动匹配）；`created_at` 与 `updated_at` 都取当前时间，`deleted_at` 为 null。

### 部分更新：`POST /notes/:id`

这一条取代了原先的 `PUT /notes/:id` + `PATCH /notes/:id` + `PATCH /notes/:id/icon` 三条路由 —— 改同一张表的字段不该有三种形状。

- **body 里出现的字段才改，没出现的字段一律不动。** 这是整条路由的关键：客户端不需要"补全自己没改的字段"。
- 原先的 `PUT` 只收 `title` + `data`，**漏传哪个就把哪个写成空**。而正文保存是 600ms 防抖触发的，某次只发 `title` 就会把正文清空。合并成部分更新之后，这个坑从"提防着别踩"变成"不可能踩到"。
- 字段的 null 语义各不相同，所以实现必须能区分**字段缺席**和**显式 null**：

| 字段 | 缺席 | 显式 null | 说明 |
|---|---|---|---|
| `title` | 不动 | 400 | 不能为 null |
| `content` | 不动 | 400 | 不能为 null |
| `parent_id` | 不动 | 移到根 | |
| `tags` | 不动 | 等价 `[]` | 清空标签 |
| `icon` | 不动 | 清空 | 恢复前端按标题自动匹配 |

- 实现上要用 `map[string]json.RawMessage` 逐个 key 判断。**不能用带 `*string` 字段的结构体** —— 指针字段分不出"没传"和"传了 null"，而这恰好正是 `parent_id: null`（移到根）与 `icon: null`（清图标）的语义。
- `tags` 传入后先规范化：逐项 trim、丢掉空串、去重（保留首次出现的顺序），再序列化成 JSON 数组。空集写 `[]` 而不是 `null`。
- 一个可更新字段都没出现 → 400。
- 只有 `parent_id` 出现且非 null 时才做防环检查（沿 `parent_id` 链向上检测）：移到自身或自己的后代 → 400。

### updated_at 什么时候刷新

| 变更的字段 | 刷新 `updated_at` | 归类 |
|---|---|---|
| `title` / `content` / `tags` | 是 | 笔记的**内容** |
| `parent_id` / `icon` | 否 | **结构**与**展示**，不是内容修改 |

理由：`updated_at` 的语义是"内容最后被修改的时间"。如果拖一下笔记、换个图标也刷新它，那么在以 `updated_at` 排序的视图里，一次拖拽就会让笔记跳位，而用户并不认为自己"改了这篇笔记"。

创建时 `created_at` 与 `updated_at` 同时置为当前时间。

### 列表查询参数

列表接口是仅有的使用 query 的地方，且全部是读取的修饰：

| 参数 | 说明 |
|---|---|
| `q` | 全文搜索，匹配 `title` + `content`。**≥ 3 个字符**走 FTS5 trigram 并按 `rank` 排序；**< 3 个字符降级为 `LIKE`**（trigram 对 2 个字符返回空结果、且不报错，所以必须自己降级），此时改按排序字段排 |
| `tag` | 只返回带该标签的笔记。**精确匹配** —— `ru` 不命中 `rust`。实现是 `json_each` + `EXISTS`，全表扫 |
| `parent_id` | 不传 = 全部未删笔记；传**空串** = 只返回根节点（实现落成 `parent_id IS NULL`）；传 id = 该节点的**直接**子节点，不递归 |
| `sort` | `created`（默认）或 `updated` |
| `limit` / `offset` | 默认 `100` / `0`；`limit` 上限 `1000`，越界或非正数一律回落为 100 |

三个筛选参数**可以叠加，关系是 AND**，且都作用在 `deleted_at IS NULL` 这个基集上：

```sql
SELECT n.* FROM notes n
  JOIN notes_fts f ON f.rowid = n.rowid           -- 仅当 q ≥ 3 个字符时才 join
 WHERE n.deleted_at IS NULL
   AND notes_fts MATCH ?                           -- q
   AND EXISTS (SELECT 1 FROM json_each(n.tags)     -- tag
               WHERE value = ?)
   AND n.parent_id IS NULL                         -- parent_id 传空串时
 ORDER BY <排序字段> DESC, id ASC
 LIMIT ? OFFSET ?;
```

排序一律带兜底列：

```sql
ORDER BY created_at DESC, id ASC   -- sort=created（默认）
ORDER BY updated_at DESC, id ASC   -- sort=updated
```

兜底列是必需的 —— 没有它，同一毫秒创建的几条笔记相对顺序由 SQLite 自行决定，可能每次刷新都不一样，树会莫名跳动。

### delete

- 默认：**软删整棵子树**（递归标记 `deleted_at`）。标签是行内数据，跟着一起隐身，还原时原样回来。
- `{ "permanent": true }`：**硬删整棵子树**，不可恢复，级联由外键与触发器完成。
- 两者都返回 `{ "id": "<笔记 id>" }`；目标不存在返回 404（对已软删的笔记再次软删也是 404）。

### restore

还原该笔记**及其整棵子树**。若原父节点仍处于删除状态，该笔记会被挂到根节点 —— 避免"父在回收站、子在树里"的悬空状态。目标不在回收站里返回 404。返回还原后的 `Note`。还原**不刷新** `updated_at`。

### GET /trash

只列出"顶层"被删笔记（自身 `deleted_at` 非空且父节点未被删），按 `deleted_at DESC`。子节点不单独出现，因为它们会随父节点一起还原。

## Tags

**标签不是实体，是 `notes.tags` 列里的字符串**（见 [schema.md](schema.md)）。由此推出三条，读下面的接口时别指望它们是别的东西：

- **没有"新建标签"接口** —— 创建标签就是给某篇笔记写上这个字符串。
- **`GET /tags` 是派生查询，不是真值源。** 真值在每条笔记的 `tags` 里；这个接口只是把全部未删笔记的标签扫出来去重。因此**没有任何笔记在用的标签不会出现在结果里**。
- **调用方可以自己维护一份。** 既然标签能从已加载的笔记本地算出来，就不必每次变更后都回头请求它 —— 本地解析、需要时再拉一次即可。接入方（前端）的具体实现不在本文范围，这里只定接口角色：**它是可再生的派生结果，不是需要保持同步的状态。**

| 方法 | 路径 | 请求体 | 成功 |
|---|---|---|---|
| GET | `/api/repos/:repo/tags` | — | 200 `string[]` |
| POST | `/api/repos/:repo/tags/rename` | `{ from, to }` | 200 `{ count }` |
| POST | `/api/repos/:repo/tags/delete` | `{ name }` | 200 `{ count }` |

- `GET` 扫全部**未删**笔记的 `tags` 去重、按名升序返回裸字符串数组。排序是字节序（SQLite 默认 collation），**中文标签不是拼音序**。
- 改名与删除是**全表改写**：一条 UPDATE 扫过所有未删笔记，改写含有该标签的行的 JSON 列。`count` 就是受影响的笔记数（`RowsAffected`）。
- **两者都会刷新受影响笔记的 `updated_at`。** 标签属于笔记内容，与 `POST /notes/:id` 改 `tags` 是同一种变更 —— 不该因为走哪条路由而给出不同答案。
- `rename` 的 `to` 已经存在时，**两个标签合并**（同一行内去重），不报错、不返回 409。
- `from` / `name` 不存在 → **404**。判据是"没有任何未删笔记在用这个标签"，不是"标签表里没有这一行" —— 因为没有标签表。
- 已软删的笔记**不参与**改写，也不出现在 `GET` 的结果里；还原后标签原样回来。

两条改写语句与实测结论见 [schema.md](schema.md#为什么没有标签表)。

## Assets

| 方法 | 路径 | 请求体 | 成功 |
|---|---|---|---|
| GET | `/api/repos/:repo/assets` | — | 200 `AssetMeta[]`（仅 ready，按 date 倒序） |
| GET | `/api/repos/:repo/assets/:sha` | — | 附件本体 |
| GET | `/api/repos/:repo/assets/:sha/meta` | — | 200 `AssetMeta` / 404 |
| POST | `/api/repos/:repo/assets/:sha` | 原始字节 | 201 `AssetMeta`（新上传）/ 200 `AssetMeta`（已存在） |
| POST | `/api/repos/:repo/assets/:sha/delete` | — | 200 `{ id }` |

`AssetMeta` = `{ id, name, mime, size, date }`，`id` 即 sha256。

### meta：为什么不用 HEAD

原先用 `HEAD /assets/:sha`（200 = 已存在）探测"这份内容在不在"。它有两个问题：`HEAD` 不在 GET/POST 这个约束里；而且它只有一个状态码，**客户端拿不到真实元数据**，于是秒传分支只能自己拼一个 `AssetMeta`（拿本地文件名冒充服务端记录），造成"界面显示的文件名"和库里实际记录不一致。

`GET /assets/:sha/meta` 一次解决两件事：404 = 没有；200 + 真实元数据 = 有，且 `name` / `mime` / `size` 都是库里的权威值。

### 上传（POST）

body 是**原始字节**，元信息走请求头：

| 头 | 说明 |
|---|---|
| `X-Name` | 文件名。必传，同时用于按扩展名推断 MIME |
| `X-Mime` | MIME。留空时按 `X-Name` 的扩展名推断，再退回 `application/octet-stream` |
| `X-Size` | 字节数。负值返回 400 |

流程是**幂等**的：

1. 该 sha 已 `ready` → 直接 **200 + 已有元数据**（秒传，一个字节都不传）
2. 插入 `status='uploading'` 行；主键冲突说明另一个请求正在上传 → **409**
3. 流式写入 `assets/tmp/<sha>`，边写边算 sha256
4. 校验通过 → 原子 `rename` 到 `assets/ab/cd/<sha>`、置 `ready` → **201 + 元数据**
5. 校验失败 → 删除 tmp 与记录行 → **400**

第 1 步和第 4 步返回的都是**服务端的真实元数据**，客户端不需要拼任何东西。

### 下载（GET）

`?inline=1` 返回 `Content-Disposition: inline`（供正文内嵌图片使用）；否则为 `attachment`，文件名按 RFC 5987 编码 UTF-8。

## 静态资源

浏览器直接取字节的内容不套上面的 JSON 约定：

| 方法 | 路径 | 说明 |
|---|---|---|
| GET / HEAD | `/icons/mdi.json` | 全量 MDI 图标集（JSON） |
| GET | `/api/repos/:repo/assets/:sha` | 附件本体（内容寻址，不可变） |
| GET | `/` | 前端静态文件 —— 当前由前端 dev server 或外部静态服务器提供，**后端不托管**（见 [dev.md](dev.md)） |

这条 `/icons/mdi.json` 由 `main.go` 直接挂在 engine 上，不在 `api` 包里 —— `api` 只注册 `/api` 下的路由。

### 图标集

**设计取舍**：图标集**不划分"有效/无效"**，而是全量一次性下发给前端，由前端缓存并离线可用。服务端的校验只用于判断"这份下载是不是坏数据"，不用于对前端做过滤。

后端托管该文件：本地副本 `data/icons/mdi.json`，启动时**仅在缺失时**下载一次，之后每 24h 后台检查一次更新。更新遵循"拿到更好的才替换"：

1. 按 `IconSources` 顺序尝试下载源，第一个成功的即采用。
2. 校验为合法 JSON（含 `prefix` 与 `icons`，且至少一个图标有 `body`）、且图标数 ≥ `IconMinCount` 才接受。
3. 内容有变化才原子替换本地副本（tmp → fsync → rename），并落盘 `<file>.meta.json` 记录来源、ETag、sha256、图标数与更新时间。
4. 任何一步失败都保留现有副本 —— 绝不因为一次坏响应把可用的图标集写坏。

响应带 `ETag`（内容 sha256）与 `Cache-Control: no-cache`，命中 `If-None-Match` 回 304，因此前端每次校验只花一个 304 就能跟上更新。

## 不做的事

明确不做，免得以后有人到处找它们 —— 这些是取舍，不是待办：

| 不做 | 为什么 |
|---|---|
| `/api/v1` 版本前缀 | 单机服务、前后端同仓同发，版本号只增加噪声 |
| 鉴权 | 只在本机与局域网使用；`Addr` 监听所有网卡是有意的 |
| 批量接口 | 唯一的批量操作是标签改名/删除，已有专用路由 |
| `{ items, total }` 分页信封与 count 接口 | 笔记量级在几千条内，一次拉全量即可；信封只多一层解析 |
| 并发控制（`ETag` / `If-Match` / 版本号） | 单用户，同名冲突不值得引入乐观锁协议 |
| 服务端推送 | 没有多端并发场景；图标集的更新由前端每次带 `If-None-Match` 主动校验（见下文） |

## 配置

**没有配置文件、环境变量或命令行参数** —— 全部硬编码在 `server/config.go`（`main.go` 会忽略并提示命令行参数）。

| 常量 | 当前值 | 说明 |
|---|---|---|
| `Addr` | `:8080` | HTTP 监听地址 |
| `DataDir` | `../data` | 仓库目录与 `repos.json` 的根（相对 `server/` 解析） |
| `IconDir` | `../data/icons` | 图标集存放目录 |
| `IconFile` | `mdi.json` | 对外地址 `/icons/mdi.json` |
| `IconCheckInterval` | `24h` | 图标集更新检查间隔 |
| `IconMinCount` | `1000` | 有效图标集的最小图标数，低于此值视为坏数据 |
| `IconSources` | 3 个镜像 | 按顺序尝试，第一个成功的即采用（后两个是可达性备份） |

其余固定行为直接写在代码里：笔记列表 `limit` 默认 100 / 上限 1000、搜索 FTS 阈值 3 字符、图标集单次下载上限 64 MiB、图标集请求超时 60s、SQLite 连接池 `MaxOpenConns(1)`、SQLite `busy_timeout` 5000ms。

`Addr` 监听所有网卡（`0.0.0.0:8080`），同网段可访问。这是**有意如此**（局域网内其他设备可直接打开），不收紧到 `127.0.0.1`。
