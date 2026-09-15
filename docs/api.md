# API

Base path `/api`；`{repo}` 为仓库 UUID；无鉴权；时间戳 Unix ms；错误统一 `{ "error": "<msg>" }`。

## Repos

```
GET    /api/repos              列出        → [{ id, name, date }]
POST   /api/repos              创建  { name }
PATCH  /api/repos/{repo}       改名  { name }
DELETE /api/repos/{repo}       删除（整个仓库目录）
```

## Notes

```
GET    /api/repos/{repo}/notes                ?q=&tag_id=&parent_id=&limit=&offset=
POST   /api/repos/{repo}/notes                { title, data, parent_id?, icon? }
GET    /api/repos/{repo}/notes/{id}
PUT    /api/repos/{repo}/notes/{id}           { title, data }（自动保存，只动 title+data）
PATCH  /api/repos/{repo}/notes/{id}           { parent_id }（移动；null=根；防环）
PATCH  /api/repos/{repo}/notes/{id}/icon      { icon }（null=恢复自动匹配）
DELETE /api/repos/{repo}/notes/{id}           ?permanent=1（默认软删除进回收站）
POST   /api/repos/{repo}/notes/{id}/restore   还原（连同子树）
GET    /api/repos/{repo}/trash               回收站
```

**Note 对象**：`{ id, parent_id, title, data, icon, date, deleted_at? }`

- `?q=` 走 FTS5 trigram（匹配 title + data，≥3 字符，按 rank 排）
- `?tag_id=` 按标签过滤
- 默认按 `date DESC`
- 删除为软删除（标记 deleted_at），`?permanent=1` 才真正级联删子树

## Tags

```
GET    /api/repos/{repo}/tags                   列出          → [{ id, name }]
POST   /api/repos/{repo}/tags                   创建  { name }
PUT    /api/repos/{repo}/tags/{id}              重命名 { name }
DELETE /api/repos/{repo}/tags/{id}              删除（级联 note_tags）

GET    /api/repos/{repo}/notes/{id}/tags        列出笔记的标签
POST   /api/repos/{repo}/notes/{id}/tags        添加  { tag_id }
DELETE /api/repos/{repo}/notes/{id}/tags/{tag_id}  移除
```

## Assets

```
GET    /api/repos/{repo}/assets          列出 ready 附件 → [{ id, name, mime, size, date }]
HEAD   /api/repos/{repo}/assets/{sha}    检查是否 ready（秒传判断）
POST   /api/repos/{repo}/assets/{sha}    上传（Headers: X-Name/X-Mime/X-Size）
GET    /api/repos/{repo}/assets/{sha}    下载  ?inline=1 内联
DELETE /api/repos/{repo}/assets/{sha}    删除
```
