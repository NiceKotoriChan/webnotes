# 数据模型 (Model)

约定：时间戳一律 Unix 毫秒；JSON 字段 snake_case；图标名为 mdi 名（不含 `mdi:` 前缀）。

## Note（笔记）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | string | UUID，不可变 |
| parent_id | string \| null | 父笔记 id；null = 根节点（树形结构） |
| title | string | 标题 |
| data | string | 正文（Markdown，即原 `content`） |
| icon | string \| null | 自定义图标名（mdi）；null = 按标题自动匹配 |
| date | number | 创建时间（Unix ms，即原 `ctime`） |
| deleted_at | number \| null | 软删除时间；null = 正常（回收站） |

关系：**笔记 ↔ 标签（多对多）**，经 `note_tags` 关联（笔记对象通过 `GET /notes/:id/tags` 取到它的标签）。

变更：**删去 `mtime`**；`content` → `data`；`ctime` → `date`。标签结构不变（仍为独立表 + 关联表）。

排序：树/列表按 `date` 倒序（最新的排最上）。

## Repo（仓库）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | string | UUID，即磁盘目录名，不可变 |
| name | string | 显示名 |
| date | number | 创建时间（原 `ctime`） |

## Tag（标签）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | string | UUID |
| name | string | 标签名，唯一 |

## note_tags（笔记-标签关联）

| 字段 | 类型 | 说明 |
|---|---|---|
| note_id | string | 笔记 id（级联删除） |
| tag_id | string | 标签 id（级联删除） |

主键 `(note_id, tag_id)`。

## Asset（附件）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | string | sha256，内容寻址（仓库内去重） |
| name | string | 文件名 |
| mime | string | MIME |
| size | number | 字节数 |
| date | number | 创建时间（原 `ctime`） |
| status | 'uploading' \| 'ready' \| 'deleting' | 上传状态机 |
