## index.db schema

```sql
PRAGMA journal_mode = WAL;

-- 附件元数据 + 上传状态：sha256 主键即文件名，存储路径 assets/<前2位>/<第3-4位>/<id>
-- 单表状态机：(空) → uploading → ready → deleting → (DELETE 行)
-- assets 表里的都是完整内容，不包含 tmp 文件
CREATE TABLE assets (
    id     TEXT PRIMARY KEY,                -- sha256
    name   TEXT NOT NULL,                  -- 原始文件名（Content-Disposition 下载名）
    mime   TEXT NOT NULL,
    size   INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'uploading'
           CHECK (status IN ('uploading','ready','deleting')),
    ctime  INTEGER NOT NULL                -- Unix 毫秒
);
```

## repo db schema

```sql
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

-- 笔记：正文直接存库，保存 = 单事务，无文件/数据库一致性问题
CREATE TABLE notes (
    id      TEXT PRIMARY KEY,                -- UUIDv4
    title   TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',        -- Markdown 正文
    ctime   INTEGER NOT NULL,                -- Unix 毫秒
    mtime   INTEGER NOT NULL
);
CREATE INDEX idx_notes_mtime ON notes(mtime DESC);

-- 全文检索：外部内容表，触发器自动同步；trigram 分词支持中文
CREATE VIRTUAL TABLE notes_fts USING fts5(
    title,
    content,
    content = 'notes',
    content_rowid = 'rowid',
    tokenize = 'trigram'
);

CREATE TRIGGER notes_ai AFTER INSERT ON notes BEGIN
    INSERT INTO notes_fts(rowid, title, content)
    VALUES (new.rowid, new.title, new.content);
END;

CREATE TRIGGER notes_ad AFTER DELETE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content)
    VALUES ('delete', old.rowid, old.title, old.content);
END;

CREATE TRIGGER notes_au AFTER UPDATE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content)
    VALUES ('delete', old.rowid, old.title, old.content);
    INSERT INTO notes_fts(rowid, title, content)
    VALUES (new.rowid, new.title, new.content);
END;

-- 标签：扁平结构，UUID 主键
CREATE TABLE tags (
    id      TEXT PRIMARY KEY,                -- UUIDv4
    name    TEXT NOT NULL UNIQUE
);

CREATE TABLE note_tags (
    note_id  TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    tag_id   TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);
CREATE INDEX idx_note_tags_tag ON note_tags(tag_id);

-- 笔记间引用：独立边表，用户显式维护，不解析/不干涉正文
-- source 引用 target；反向链接 = WHERE target_id = ?
CREATE TABLE refs (
    source_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    target_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    ctime     INTEGER NOT NULL,                -- Unix 毫秒
    PRIMARY KEY (source_id, target_id),
    CHECK (source_id != target_id)
);
CREATE INDEX idx_refs_target ON refs(target_id);
```

FTS 查询示例：

```sql
SELECT n.* FROM notes_fts f
JOIN notes n ON n.rowid = f.rowid
WHERE notes_fts MATCH ?
ORDER BY f.rank;
```

索引损坏时重建：`INSERT INTO notes_fts(notes_fts) VALUES ('rebuild');`
