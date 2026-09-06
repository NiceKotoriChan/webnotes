每个 repo 一个数据库文件（repos/<repo>.db），schema 如下：

附件无元数据库：MIME 响应时嗅探，size 用 os.Stat，文件是否存在即去重依据。

```sql
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

-- 笔记：正文直接存库，保存 = 单事务，无文件/数据库一致性问题
CREATE TABLE notes (
    id      TEXT PRIMARY KEY,                -- UUIDv7
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

-- 标签：扁平结构，name 即主键
CREATE TABLE tags (
    name TEXT PRIMARY KEY
);

CREATE TABLE note_tags (
    note_id  TEXT NOT NULL REFERENCES notes(id)  ON DELETE CASCADE,
    tag_name TEXT NOT NULL REFERENCES tags(name) ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY (note_id, tag_name)
);
CREATE INDEX idx_note_tags_tag ON note_tags(tag_name);

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

-- 附件关联：sha256 对应全局 assets/ 目录下的内容寻址文件
-- （附件无元数据库，不做 FK 约束，孤儿文件由全局 GC 回收）
CREATE TABLE note_assets (
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    sha256  TEXT NOT NULL,
    PRIMARY KEY (note_id, sha256)
);
CREATE INDEX idx_note_assets_sha ON note_assets(sha256);
```

FTS 查询示例：

```sql
SELECT n.* FROM notes_fts f
JOIN notes n ON n.rowid = f.rowid
WHERE notes_fts MATCH ?
ORDER BY f.rank;
```

索引损坏时重建：`INSERT INTO notes_fts(notes_fts) VALUES ('rebuild');`
