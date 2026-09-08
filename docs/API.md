## 约定
;
- Base path: `/api`
- 时间戳：Unix 毫秒（INTEGER）
- 字段命名：snake_case
- 错误响应：HTTP 状态码 + `{ "error": "<msg>" }`
- 无鉴权（本地/局域部署）

## Repos

```
GET    /api/repos                    列出所有 repo
POST   /api/repos                    创建 repo        body: { id }
DELETE /api/repos/{repo}            删除 repo（含 db 文件）
```

## Notes

```
GET    /api/repos/{repo}/notes                       列出 ?q=&tag_id=&limit=&offset=
POST   /api/repos/{repo}/notes                       创建  body: { title, content }   → { id, title, content, ctime, mtime }
GET    /api/repos/{repo}/notes/{id}                  获取
PUT    /api/repos/{repo}/notes/{id}                  更新  body: { title, content }（mtime=now，ctime 不变）
DELETE /api/repos/{repo}/notes/{id}                  删除（级联 note_tags、refs、FTS）
```

`?q=` 走 FTS5 trigram 匹配，按 rank 排序。

## Tags

```
GET    /api/repos/{repo}/tags                        列出
POST   /api/repos/{repo}/tags                        创建  body: { name }              → { id, name }
PUT    /api/repos/{repo}/tags/{id}                   重命名  body: { name }
DELETE /api/repos/{repo}/tags/{id}                   删除（级联 note_tags）
```

## Note-Tag 关联

```
GET    /api/repos/{repo}/notes/{id}/tags             列出笔记的标签
POST   /api/repos/{repo}/notes/{id}/tags             添加  body: { tag_id }
DELETE /api/repos/{repo}/notes/{id}/tags/{tag_id}    移除
```

## Refs（笔记间引用）

```
GET    /api/repos/{repo}/notes/{id}/refs             出边引用（本笔记引用了谁）
GET    /api/repos/{repo}/notes/{id}/backrefs         入边引用（谁引用了本笔记，即反向链接）
POST   /api/repos/{repo}/notes/{id}/refs             创建出边  body: { target_id }   ctime=now
DELETE /api/repos/{repo}/notes/{id}/refs/{target_id} 删除出边
```

`source != target` 由 schema CHECK 约束；自环返回 400。

## Assets（附件）

```
HEAD   /api/assets/{sha256}                          检查是否 ready
       200 ready           404 不存在或非 ready
POST   /api/assets/{sha256}                          上传
       Headers: X-Name, X-Mime, X-Size
       Body:    raw 二进制（流式写 assets/tmp/<sha256>）
       服务端流式重算 sha256 比对 path 一致才 rename
       201 { id, name, mime, size, ctime }          409 已存在/上传中
GET    /api/assets/{sha256}                          下载
       Content-Type: <mime>
       Content-Disposition: attachment; filename*=UTF-8''<urlencoded name>
       ?inline=1 改用 inline
DELETE /api/assets/{sha256}                          删除
       status!='ready' 返回 409 拒绝
       204 成功
```

并发与断点续传不处理：第二个客户端 POST 同一 sha256 会撞 PRIMARY KEY，返回 409，前端报错"上传中，请稍后重试"。
