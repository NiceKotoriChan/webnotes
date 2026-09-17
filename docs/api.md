# API

## 约定

- Base path `/api`。`{repo}` = 仓库 UUID，`{id}` = 笔记/标签 UUID，`{sha}` = 附件 sha256。
- **一律 `POST`，只有静态资源是 `GET`**（见文末）。不区分 GET/PUT/PATCH/DELETE，**动作名写在路径最后一段**。
- **参数一律放 JSON body**；路径只承载 id 和动作名，不带查询串。附件上传例外（body 是原始字节）。
- 没有返回值的动作用 `204`；其余 `200`。错误统一 `{ "error": "<msg>" }`。
- 时间戳一律 Unix 毫秒；JSON 字段 snake_case；无鉴权。

```
/api/{repo}

```

rid 仓库
nid
tid

## Repos

```
POST /api/repos/list
POST /api/repos/create?name={}
POST /api/repos/rename?rid={}
POST /api/repos/delete?rid={}
```

## Notes

```
POST /api/notes/list?rid={}
POST /api/notes/create?rid={}
POST /api/notes/update?rid={}&nid={}
POST /api/repos/{repo}/notes/{id}/save   { title, data }                          → Note（自动保存，只动 title+data）
POST /api/repos/{repo}/notes/{id}/move   { parent_id }                            → Note（null = 根；防环）
POST /api/repos/{repo}/notes/{id}/icon   { icon }                                 → 204（null = 恢复自动匹配）
POST /api/repos/{repo}/notes/{id}/delete { permanent? }                           → 204（默认软删除整棵子树）
POST /api/repos/{repo}/notes/{id}/restore                                         → 204（连同子树）
POST /api/repos/{repo}/trash/list
```

**Note**：`{ id, parent_id, title, data, icon, date, deleted_at? }`

`list` 的参数：

| 参数               | 说明                                                                       |
| ------------------ | -------------------------------------------------------------------------- |
| `q`                | 全文搜索（FTS5 trigram，匹配 title + data；< 3 字符降级 LIKE，按 rank 排） |
| `tag_id`           | 按标签过滤                                                                 |
| `parent_id`        | 不传/null = 全部；`""` = 只看根节点；id = 该节点的直接子节点               |
| `limit` / `offset` | 默认 `100` / `0`（limit 上限 1000）                                        |

默认按 `date DESC`。`delete` 为软删除（标记 `deleted_at`），`permanent: true` 才真正级联删子树。

## Tags

```
POST /api/repos/{repo}/tags/list            → [{ id, name }]
POST /api/repos/{repo}/tags/create  { name }→ Tag
POST /api/repos/{repo}/tags/update?id={}&name={}
POST /api/repos/{repo}/tags/{id}/delete     → 204（级联 note_tags）

POST /api/repos/{repo}/notes/{id}/tags/list         → [Tag]
POST /api/repos/{repo}/notes/{id}/tags/add    { tag_id } → 204
POST /api/repos/{repo}/notes/{id}/tags/remove { tag_id } → 204
```

## Assets

```
POST /api/repos/{repo}/assets/list                 → [{ id, name, mime, size, date }]（仅 ready）
POST /api/repos/{repo}/assets/{sha}/exists         → 200 = 已 ready（可秒传）；404 = 没有
POST /api/repos/{repo}/assets/{sha}/upload         → Asset（Headers: X-Name / X-Mime / X-Size；body = 原始字节）
POST /api/repos/{repo}/assets/{sha}/delete         → 204
```

## 静态资源（唯一走 GET 的部分）

浏览器直接拿字节的东西，不套 RPC：

```
GET  /api/repos/{repo}/assets/{sha}   ?inline=1 → 附件本身（附件是内容寻址的不可变文件）
GET  /icons/mdi.json                  全量 MDI 图标集（JSON）
GET  /                                web/ 前端静态文件（见 docs/add.md 的 pack 布局）
```

- 附件、图标都是**不可变**的（sha256 寻址 / 只增不改），所以 GET + 缓存语义天然正确。
- 图标集由后端托管：本地副本 `data/icons/mdi.json`，启动检查一次、之后每 24h 一次，校验通过才原子替换。
