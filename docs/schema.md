# 数据库 Schema

每个仓库一个自包含目录 `<root>/<uuid>/`，内含 `data.db`（元数据）与 `assets/`（附件文件）。

## SQL

```sql
CREATE TABLE notes (
    id         TEXT PRIMARY KEY,
    parent_id  TEXT REFERENCES notes(id) ON DELETE CASCADE,
    title      TEXT NOT NULL DEFAULT '',
    data       TEXT NOT NULL DEFAULT '',
    icon       TEXT,
    date       INTEGER NOT NULL,
    deleted_at INTEGER
);
CREATE INDEX idx_notes_parent ON notes(parent_id);
CREATE INDEX idx_notes_date   ON notes(date DESC);

CREATE VIRTUAL TABLE notes_fts USING fts5(
    title, data,
    content = 'notes', content_rowid = 'rowid',
    tokenize = 'trigram'
);
-- 触发器：INSERT/DELETE/UPDATE 同步 notes ↔ notes_fts（内容取 title + data）

CREATE TABLE tags (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);
CREATE TABLE note_tags (
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    tag_id  TEXT NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);
CREATE INDEX idx_note_tags_tag ON note_tags(tag_id);

CREATE TABLE assets (
    id     TEXT PRIMARY KEY,
    name   TEXT NOT NULL,
    mime   TEXT NOT NULL,
    size   INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'uploading'
           CHECK (status IN ('uploading','ready','deleting')),
    date   INTEGER NOT NULL
);
```

## 说明

- 无 `mtime`、无 `ctime`（统一为 `date`）。
- 标签结构不变：独立 `tags` 表 + `note_tags` 关联表（多对多）。
- 附件 sha256 内容寻址、两级分片存储；`repos.json` 仅索引、目录为真相源 —— 沿用原设计。

## 目录结构与运维

```
webnotes/
├── repos.json                  -- 全局仓库索引（可重建，非真相源）
├── <uuid>/                     -- 目录名 = 仓库 UUID，不可变
│   ├── data.db                 -- 笔记/标签/附件元数据
│   └── assets/                 -- 该仓库私有附件
│       ├── tmp/                -- 上传中转
│       └── ab/cd/<sha256>      -- 内容寻址，两级分片
```

- `repos.json` 仅索引，启动对账：目录为真相源（目录有/json 无 → 补；json 有/目录无 → 删）。
- 附件上传：客户端算 sha256 → `HEAD` 秒传判断 → 流式写 `tmp/` 边写边校验 → `rename` 落盘 → `status=ready`；崩溃恢复靠状态机（uploading/ready/deleting）+ 启动对账。
- 备份：`data.db` 用 sqlite backup API / `VACUUM INTO`；`repos.json` 一并复制；`assets/` 直接复制（内容不可变）。
