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
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    deleted_at      INTEGER
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
