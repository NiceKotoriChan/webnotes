## 约定

- Base path：`/api`；路径里的 `{repo}` 是仓库 UUID（不可变）
- 时间戳：Unix 毫秒；字段命名：snake_case
- 错误：HTTP 状态码 + `{ "error": "<msg>" }`；无鉴权（本地/局域部署）

## Repos

```
GET    /api/repos                      列出          → [{ id, name, ctime }]
POST   /api/repos                      创建  { name }         → { id, name, ctime }
                                       （服务端生成 UUID，建目录 + data.db）
PATCH  /api/repos/{repo}               改名  { name }         → { id, name, ctime }
                                       （只改 repos.json，目录/链接不动）
DELETE /api/repos/{repo}               删除（整个仓库目录，含 db 与附件）
```

## Notes

笔记为树形结构，`parent_id` 为 NULL 即根节点。

```
GET    /api/repos/{repo}/notes         列出  ?q=&tag_id=&parent_id=&limit=&offset=
POST   /api/repos/{repo}/notes         创建  { title, content, parent_id? }
                                       → { id, parent_id, title, content, ctime, mtime }
GET    /api/repos/{repo}/notes/{id}    获取
PUT    /api/repos/{repo}/notes/{id}    更新  { title, content }（mtime=now，ctime 不变）
PATCH  /api/repos/{repo}/notes/{id}    移动  { parent_id }（parent_id=null 移到根）
                                       服务端沿 parent 链防环；成环返回 400
PATCH  /api/repos/{repo}/notes/{id}/icon  设置自定义图标  { icon }（icon=null 恢复自动匹配）
DELETE /api/repos/{repo}/notes/{id}    删除（默认软删除整棵子树进回收站；?permanent=1 彻底删除）
POST   /api/repos/{repo}/notes/{id}/restore  还原（连同子树；父级仍删除则移到根）
GET    /api/repos/{repo}/trash        回收站（列出被删除的顶层笔记）
```

- `?q=` 走 FTS5 trigram（匹配 title + content，按 rank 排序；查询词需 ≥3 字符）
- `?parent_id=` 只返回该节点的直接子笔记；不传则列出全部
- 删除为软删除（标记 deleted_at），可从回收站还原；?permanent=1 才真正级联删子树（不二次确认）

## Tags

```
GET    /api/repos/{repo}/tags          列出
POST   /api/repos/{repo}/tags          创建  { name }   → { id, name }
PUT    /api/repos/{repo}/tags/{id}     重命名 { name }
DELETE /api/repos/{repo}/tags/{id}     删除（级联 note_tags）
```

## Note-Tag 关联

```
GET    /api/repos/{repo}/notes/{id}/tags             列出笔记的标签
POST   /api/repos/{repo}/notes/{id}/tags             添加  { tag_id }
DELETE /api/repos/{repo}/notes/{id}/tags/{tag_id}    移除
```

## Assets（附件，仓库内私有）

```
GET    /api/repos/{repo}/assets           列出全部 ready 附件 → [{ id, name, mime, size, ctime }]
HEAD   /api/repos/{repo}/assets/{sha256}   检查是否 ready
       200 ready        404 不存在或非 ready
POST   /api/repos/{repo}/assets/{sha256}   上传
       Headers: X-Name, X-Mime, X-Size
       Body:    raw 二进制（流式写本仓库 assets/tmp/<sha256>）
       服务端流式重算 sha256，与 path 一致才 rename
       201 { id, name, mime, size, ctime }      409 已存在/上传中
GET    /api/repos/{repo}/assets/{sha256}   下载
       Content-Type: <mime>
       Content-Disposition: attachment; filename*=UTF-8''<urlencoded name>
       ?inline=1 改用 inline（用于正文内嵌图片）
DELETE /api/repos/{repo}/assets/{sha256}   删除
       status!='ready' 返回 409；成功 204
```

附件按 sha256 在仓库内去重；正文引用 `/api/repos/{repo}/assets/{sha256>`，
因 repo 用不可变 UUID，仓库改名后历史链接仍有效。
并发与断点续传不处理：同 sha256 第二个上传者撞主键返回 409，前端提示"上传中，请稍后重试"。
