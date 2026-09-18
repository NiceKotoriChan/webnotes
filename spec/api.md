# 接口

Base path `/api`，**21 条路由，方法只有 `GET` 与 `POST`**。响应一律 JSON，错误形如 `{ "error": "<msg>" }`。无鉴权、无版本前缀。

**GET 只读，POST 是唯一写入口** —— 浏览器会预取、缓存、重试 GET，只要 GET 不写数据，这些行为就永远不会误触发一次修改。POST 按路径形状分三种：

| 形态        | 例                       | 语义                                  | 成功码       |
| ----------- | ------------------------ | ------------------------------------- | ------------ |
| 集合路径    | `POST /notes`            | 新建                                  | 201 + 新资源 |
| 资源路径    | `POST /notes/:id`        | 部分更新                              | 200 + 新状态 |
| 资源 + 动词 | `POST /notes/:id/delete` | 动作（动词只有 `delete` / `restore`） | 200 + 回执   |

客户端能写的字段被封闭在 `title` / `content` / `parent_id` / `tags` / `icon` 五个里，写不了 `created_at` / `updated_at` / `deleted_at`。

约定：路径参数写法与 `router.go` 逐字一致（`:repo` / `:id` / `:sha`）；参数放路径或 JSON body，只有列表筛选与附件 `?inline=1` 用 query；**POST 一律回 JSON body，没有 204**，删除类动作回 `{ "id": ... }` 作回执；列表返回**裸数组**，不套信封。

## Repos

| 方法 | 路径                      | 请求体     | 成功         |
| ---- | ------------------------- | ---------- | ------------ |
| GET  | `/api/repos`              | —          | 200 `Repo[]` |
| POST | `/api/repos`              | `{ name }` | 201 `Repo`   |
| POST | `/api/repos/:repo`        | `{ name }` | 200 `Repo`   |
| POST | `/api/repos/:repo/delete` | —          | 200 `{ id }` |

## Notes

| 方法 | 路径                                 | 请求体                                           | 成功         |
| ---- | ------------------------------------ | ------------------------------------------------ | ------------ |
| GET  | `/api/repos/:repo/notes`             | —                                                | 200 `Note[]` |
| POST | `/api/repos/:repo/notes`             | `{ title?, content?, parent_id?, tags?, icon? }` | 201 `Note`   |
| GET  | `/api/repos/:repo/notes/:id`         | —                                                | 200 `Note`   |
| POST | `/api/repos/:repo/notes/:id`         | `{ title?, content?, parent_id?, tags?, icon? }` | 200 `Note`   |
| POST | `/api/repos/:repo/notes/:id/delete`  | `{ permanent? }`                                 | 200 `{ id }` |
| POST | `/api/repos/:repo/notes/:id/restore` | —                                                | 200 `Note`   |
| GET  | `/api/repos/:repo/trash`             | —                                                | 200 `Note[]` |

### `POST /notes/:id`：body 里出现的字段才改

字段的 null 语义各不相同，实现必须能区分**字段缺席**与**显式 null**：

| 字段                | 缺席 | 显式 null         |
| ------------------- | ---- | ----------------- |
| `title` / `content` | 不动 | 400               |
| `parent_id`         | 不动 | 移到根            |
| `tags`              | 不动 | 等价 `[]`（清空） |
| `icon`              | 不动 | 清空              |

- 只有 `parent_id` 出现且非 null 时才做防环检查。一个可更新字段都没出现 → 400。
- `tags` 先规范化再序列化成 JSON 数组（空集写 `[]`）。
- 哪些字段会刷新 `updated_at`，见 [model.md](model.md)。

### 列表查询参数

| 参数               | 说明                                                                                                                               |
| ------------------ | ---------------------------------------------------------------------------------------------------------------------------------- |
| `q`                | 匹配 `title` + `content`。FTS5 trigram，相关度顺序由数据库给出（**<3 字符降级 `LIKE`**），实现见 [schema.md](schema.md)             |
| `tag`              | 只返回带该标签的笔记。**精确匹配**（`ru` 不命中 `rust`），实现是全表扫                                                             |
| `parent_id`        | 不传 = 全部未删笔记；传**空串** = 只根节点；传 id = 该节点的**直接**子节点                                                         |
| `limit` / `offset` | 默认 `100` / `0`，`limit` 上限 `1000`，越界或非正数回落为 100                                                                      |

三个筛选参数**叠加关系是 AND**，基集是 `deleted_at IS NULL`。**接口不出排序参数**：除 `q` 由数据库按相关度给顺序外，结果一律按**数据库默认顺序**（不写 `ORDER BY`）返回，`limit` / `offset` 也照这个顺序切片。文档树在发送后由前端自己排。

### delete / restore / trash

- `delete`：默认**软删整棵子树**；`{ "permanent": true }` 则硬删整棵子树，不可恢复。都返回 `{ "id" }`；目标不存在（含对已软删笔记再次软删）返回 404。
- `restore`：还原该笔记**及其整棵子树**，不在回收站里返回 404。返回还原后的 `Note`，**不刷新** `updated_at`。
- `GET /trash`：只列「顶层」被删笔记（自身已删且父节点未删），按 `deleted_at DESC`。

## Tags

标签不是实体（见 [model.md](model.md)），所以**没有「新建标签」接口**；`GET /tags` 是派生查询，没有任何笔记在用的标签不会出现在结果里。

| 方法 | 路径                           | 请求体         | 成功            |
| ---- | ------------------------------ | -------------- | --------------- |
| GET  | `/api/repos/:repo/tags`        | —              | 200 `string[]`  |
| POST | `/api/repos/:repo/tags/rename` | `{ from, to }` | 200 `{ count }` |
| POST | `/api/repos/:repo/tags/delete` | `{ name }`     | 200 `{ count }` |

- `count` 是受影响的笔记数；`from` / `name` 不存在 → 404（判据是没有任何未删笔记在用这个标签）。
- `rename` 的 `to` 已存在时**两个标签合并**，不报错、不返回 409。
- 已软删的笔记不参与改写；`GET` 按名升序，排序是字节序。

## Assets

| 方法 | 路径                                  | 请求体   | 成功                                        |
| ---- | ------------------------------------- | -------- | ------------------------------------------- |
| GET  | `/api/repos/:repo/assets`             | —        | 200 `AssetMeta[]`（仅 ready，按 date 倒序） |
| GET  | `/api/repos/:repo/assets/:sha`        | —        | 附件本体                                    |
| GET  | `/api/repos/:repo/assets/:sha/meta`   | —        | 200 `AssetMeta` / 404                       |
| POST | `/api/repos/:repo/assets/:sha`        | 原始字节 | 201（新上传）/ 200（已存在）                |
| POST | `/api/repos/:repo/assets/:sha/delete` | —        | 200 `{ id }`                                |

**用 `meta` 而不是 `HEAD`**：`HEAD` 只有一个状态码，客户端拿不到真实元数据，秒传分支只能自己拼一个 `AssetMeta`。`GET .../meta` 一次解决两件事 —— 404 = 没有；200 + 服务端权威元数据 = 有。

**上传**是幂等的：body 为原始字节，元信息走请求头 `X-Name`（必传，也用于按扩展名推断 MIME）、`X-Mime`（留空则按扩展名推断，再退回 `application/octet-stream`）、`X-Size`（负值 400）。

1. 该 sha 已 `ready` → 直接 **200 + 已有元数据**（秒传，一个字节都不传）
2. 插入 `status='uploading'` 行；主键冲突说明另一个请求正在上传 → **409**
3. 流式写入临时文件并边写边算 sha256；通过则原子 `rename` 并置 `ready` → **201 + 元数据**；失败则删临时文件与记录行 → **400**

**下载**：`?inline=1` 返回 `Content-Disposition: inline`（供正文内嵌图片），否则为 `attachment`，文件名按 RFC 5987 编码 UTF-8。

## 错误码

| 码  | 何时出现                                                                                                                                                                 |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 400 | body 不是合法 JSON；没有任何可更新字段；`title` / `content` 传 null；`parent_id` 不存在或成环；`tags` 不是字符串数组；`:sha` 非法；`X-Size` 为负；上传时 sha256 校验失败；规则不是合法 JSON、缺 `rules` 数组，或某条规则缺 `match`／`icon` |
| 404 | 仓库 / 笔记 / 标签 / 待还原笔记不存在；附件未 `ready`                                                                                                                    |
| 409 | 附件正在被另一个请求上传；删除非 `ready` 的附件                                                                                                                          |
| 500 | 其余。**对外只回 `internal error`**，SQL 原文只进服务端日志                                                                                                              |

## 图标规则

规则文件分两份：**默认规则只读，自定义规则可改**。后端在启动时用默认规则灌一份自定义规则的初值，所以它一开始就等于默认规则；前端改的始终是自定义规则，要撤销就整个回退。

| 方法 | 路径                     | 请求体        | 成功               |
| ---- | ------------------------ | ------------- | ------------------ |
| POST | `/api/icons/rules`       | 完整规则 JSON | 200 规范化后的规则 |
| POST | `/api/icons/rules/reset` | —             | 200 默认规则       |

- **读走静态路径**（`/icons/mdi_rules_custom.json`，见下节），写只有这两条。规则没有 id、不是集合，所以不做增量补丁：`POST /api/icons/rules` 是**整体覆盖**，前端读出来、改完整体回传。
- 落盘前做结构校验：必须是 JSON 对象、带 `rules` 数组、每条 `match` 与 `icon` 非空。**正则本身不校验** —— 规则最终由前端用 JS 正则执行，Go 的方言更窄，拿标准库去卡会把合法的前端规则误判成非法。
- 写盘是原子的（同目录临时文件 + `rename`），校验不过就一个字节都不落盘。
- `reset` 用默认规则覆盖自定义规则；把自定义规则文件删掉再重启，是同一效果。

## 静态资源

| 方法       | 路径                            | 说明                              |
| ---------- | ------------------------------- | --------------------------------- |
| GET / HEAD | `/icons/mdi.json`               | 全量 MDI 图标集（JSON）           |
| GET / HEAD | `/icons/mdi_rules_default.json` | 默认图标规则                      |
| GET / HEAD | `/icons/mdi_rules_custom.json`  | 自定义图标规则（唯一可写的一份）  |

图标集全量一次性下发，由前端缓存；后端持本地副本，启动时缺失才下载，之后每 24h 后台检查。响应带 `ETag` 与 `Cache-Control: no-cache`，命中 `If-None-Match` 回 304；规则文件同理，但每次请求现读盘，改完立刻生效。

**为什么有规则文件**：mdi 的图标名很难从笔记标题直接匹配出来，所以要有一份规则把标题映射到图标名 —— `mdi_rules_default.json` 是内置的默认规则，`mdi_rules_custom.json` 供自定义。两份文件同构，**后端只托管与校验、不执行匹配**（匹配由前端在 `icon` 为 null 时做，见 [schema.md](schema.md)）。

```json
{
  "fallback": "file-document-outline",
  "rules": [
    { "match": "\\.md$", "icon": "file-document-outline" },
    { "match": "linux|内核", "icon": "linux" }
  ]
}
```

- `rules` **按数组顺序匹配，第一条命中即停**；`match` 是大小写不敏感的**正则**，匹配对象是笔记标题（含扩展名，所以扩展名写成 `\\.go$` 这种形式）。`icon` 是不带 `mdi:` 前缀的图标名。
- `fallback`：全部规则都没命中时的兜底图标，取自定义规则里的那份，它没写就用默认规则的。
- 先跑 custom，未命中再跑 default —— 自定义规则天然压过默认规则，想改掉某条默认匹配，在 custom 里写一条更靠前、能先命中的规则即可。
