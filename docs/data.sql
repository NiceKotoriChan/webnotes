PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
-- 附件：sha256 即主键/文件名，存 assets/<前2位>/<3-4位>/<id>
-- 状态机：uploading → ready → deleting → 删除行；表内只登记完整文件
CREATE TABLE assets (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    mime TEXT NOT NULL,
    size INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'uploading' CHECK (status IN ('uploading', 'ready', 'deleting')),
    ctime INTEGER NOT NULL
);
-- 笔记：树形结构，id 为 UUIDv7，parent_id 为 NULL 即根节点；正文直接存库
-- 删除默认软删除（deleted_at 标记）进回收站，可还原；?permanent=1 才硬删子树
-- icon 为自定义图标名（lucide，不含前缀），NULL 用前端自动匹配
CREATE TABLE notes (
    id TEXT PRIMARY KEY,
    parent_id TEXT REFERENCES notes(id) ON DELETE CASCADE,
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    ctime INTEGER NOT NULL,
    mtime INTEGER NOT NULL,
    deleted_at INTEGER,
    icon TEXT
);
CREATE INDEX idx_notes_parent ON notes(parent_id);
CREATE INDEX idx_notes_mtime ON notes(mtime DESC);
-- 标签：id 为 UUIDv7，name 为唯一
-- 删标签会级联删除所有关联笔记
CREATE TABLE tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);
CREATE TABLE note_tags (
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);
CREATE INDEX idx_note_tags_tag ON note_tags(tag_id);
-- 全文检索：外部内容表，触发器自动同步；trigram 分词支持中文（查询词需 ≥3 字符）
CREATE VIRTUAL TABLE notes_fts USING fts5(
    title,
    content,
    content = 'notes',
    content_rowid = 'rowid',
    tokenize = 'trigram'
);
CREATE TRIGGER notes_ai
AFTER
INSERT ON notes BEGIN
INSERT INTO notes_fts(rowid, title, content)
VALUES (new.rowid, new.title, new.content);
END;
CREATE TRIGGER notes_ad
AFTER DELETE ON notes BEGIN
INSERT INTO notes_fts(notes_fts, rowid, title, content)
VALUES ('delete', old.rowid, old.title, old.content);
END;
CREATE TRIGGER notes_au
AFTER
UPDATE ON notes BEGIN
INSERT INTO notes_fts(notes_fts, rowid, title, content)
VALUES ('delete', old.rowid, old.title, old.content);
INSERT INTO notes_fts(rowid, title, content)
VALUES (new.rowid, new.title, new.content);
END;
-- 搜索：SELECT n.* FROM notes_fts f JOIN notes n ON n.rowid = f.rowid
--       WHERE notes_fts MATCH ? ORDER BY f.rank;
-- 重建：INSERT INTO notes_fts(notes_fts) VALUES ('rebuild');