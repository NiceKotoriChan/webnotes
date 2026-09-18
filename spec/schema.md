# 数据库

```sql
CREATE TABLE assets (
    id     TEXT PRIMARY KEY,
    date   INTEGER NOT NULL,
    name   TEXT NOT NULL,
    mime   TEXT NOT NULL,
    size   INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'uploading'
           CHECK (status IN ('uploading','ready','deleting'))
);

CREATE TABLE notes (
    id         TEXT PRIMARY KEY,
    parent_id  TEXT REFERENCES notes(id) ON DELETE CASCADE,

    title   TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',

    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    deleted_at  INTEGER,

    tags    TEXT,
    icon    TEXT
);
CREATE INDEX idx_notes_parent ON notes(parent_id);
CREATE INDEX idx_notes_created ON notes(created_at DESC);

CREATE VIRTUAL TABLE notes_fts USING fts5(
    title, content,
    content = 'notes', content_rowid = 'rowid',
    tokenize = 'trigram'
);
CREATE TRIGGER notes_ai AFTER INSERT ON notes BEGIN
    INSERT INTO notes_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
END;
CREATE TRIGGER notes_ad AFTER DELETE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES ('delete', old.rowid, old.title, old.content);
END;
CREATE TRIGGER notes_au AFTER UPDATE OF title, content ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES ('delete', old.rowid, old.title, old.content);
    INSERT INTO notes_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
END;
```

约定：时间戳一律 Unix 毫秒；`id` 一般是 UUID，`assets.id` 例外（内容寻址的 sha256）。

## notes

| 列         | 类型            | 说明                                                           |
| ---------- | --------------- | -------------------------------------------------------------- |
| id         | TEXT            | UUID 主键，不可变                                              |
| parent_id  | TEXT \| null    | 父笔记；null = 根节点。硬删时 `ON DELETE CASCADE` 一次删完子树 |
| title      | TEXT            | 标题，默认空串                                                 |
| content    | TEXT            | 正文（Markdown）                                               |
| created_at | INTEGER         | 创建时间，写入后再不改动                                       |
| updated_at | INTEGER         | 内容最后修改时间；刷新规则见 [model.md](model.md)              |
| deleted_at | INTEGER \| null | 软删时间；null = 正常                                          |
| tags       | TEXT            | 标签，**JSON 数组**如 `["rust","api"]`；无标签写 `[]`          |
| icon       | TEXT \| null    | mdi 图标名（不含 `mdi:`）；null = 由前端按标题匹配             |

## assets

| 列                 | 类型                  | 说明                                                             |
| ------------------ | --------------------- | ---------------------------------------------------------------- |
| id                 | TEXT                  | sha256，内容寻址：同仓库内同内容只存一份                         |
| date               | INTEGER               | 上传时间                                                         |
| name / mime / size | TEXT / TEXT / INTEGER | **上传时**的记录，与文件内容无关                                 |
| status             | TEXT                  | `uploading` / `ready` / `deleting` 三态，见 [model.md](model.md) |

附件落盘路径 `assets/ab/cd/<sha>`（取 sha 前四位分两级），写入期的临时文件在 `assets/tmp/`。

`notes_fts` 是 fts5 **外部内容表**（自身不存原文），靠上面三个触发器同步。其中 **`notes_au` 的 `UPDATE OF title, content` 是必需的**，不是装饰：没有它，触发器会对任何 UPDATE 重建索引 —— 包括只改标签、只刷 `updated_at`、只标软删，而软删是递归的，删一棵子树就把子树上每条笔记重索引一遍。判定依据是**列出现在 SET 里**，不看值是否真的变。

**软删的笔记仍留在索引里**，靠查询层的 `deleted_at IS NULL` 过滤。让触发器无条件维护索引、把过滤放在查询层，比让触发器去猜"这次软删要不要动索引"更安全。

搜索：**≥3 字符**用 `notes_fts MATCH`，结果按 `rank`（相关度）返回；**<3 字符降级 `LIKE '%q%'`** —— trigram 至少要 3 个字符才切得出词，短于此不报错但恒定空结果（中英文同样如此）。
