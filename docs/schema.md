# 数据库

> **目标设计，代码尚未跟进**（现状见根 [README](../README.md)）。本文件是 schema 的真相源：**先改这里，再同步 `server/store/store.go` 的 `RepoSchema`** —— 那个常量与下面的 SQL 逐字一致，只多 `IF NOT EXISTS`（每次打开库都会执行一遍）。两者不一致视为 bug。

每个仓库一个自包含目录 `<data>/<uuid>/`，内含 `data.db`（SQLite）与 `assets/`。连接固定 `MaxOpenConns(1)`、`busy_timeout 5000ms`。

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

| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT | UUID 主键，不可变 |
| parent_id | TEXT \| null | 父笔记；null = 根节点。硬删时 `ON DELETE CASCADE` 一次删完子树 |
| title | TEXT | 标题，默认空串 |
| content | TEXT | 正文（Markdown） |
| created_at | INTEGER | 创建时间，写入后再不改动 |
| updated_at | INTEGER | 内容最后修改时间；刷新规则见 [model.md](model.md) |
| deleted_at | INTEGER \| null | 软删时间；null = 正常 |
| tags | TEXT | 标签，**JSON 数组**如 `["rust","api"]`；无标签写 `[]` |
| icon | TEXT \| null | mdi 图标名（不含 `mdi:`）；null = 由前端按标题匹配 |

## assets

| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT | sha256，内容寻址：同仓库内同内容只存一份 |
| date | INTEGER | 上传时间 |
| name / mime / size | TEXT / TEXT / INTEGER | **上传时**的记录，与文件内容无关 |
| status | TEXT | `uploading` / `ready` / `deleting` 三态，见 [model.md](model.md) |

附件落盘路径 `assets/ab/cd/<sha>`（取 sha 前四位分两级），写入期的临时文件在 `assets/tmp/`。

> `assets.date` 有意保留 `date`（附件没有"创建 vs 修改"之分），`repos.json` 里的 `date` 同理，不做统一。

## 索引与 FTS

| 索引 | 用途 |
|---|---|
| `idx_notes_parent` | 取直接子节点（树每展开一层都要用） |
| `idx_notes_created` | 默认排序 `created_at DESC` |

标签是 JSON 列，**不能走索引**（`json_each` + `EXISTS` 全表扫）。单机、几千条笔记的量级下无所谓。`updated_at` 暂不建索引 —— 默认排序不用它，`?sort=updated` 真用起来再加 `idx_notes_updated`。

`notes_fts` 是 fts5 **外部内容表**（自身不存原文），靠上面三个触发器同步。其中 **`notes_au` 的 `UPDATE OF title, content` 是必需的**，不是装饰：没有它，触发器会对任何 UPDATE 重建索引 —— 包括只改标签、只刷 `updated_at`、只标软删，而软删是递归的，删一棵子树就把子树上每条笔记重索引一遍。判定依据是**列出现在 SET 里**，不看值是否真的变。

**软删的笔记仍留在索引里**，靠查询层的 `deleted_at IS NULL` 过滤。让触发器无条件维护索引、把过滤放在查询层，比让触发器去猜"这次软删要不要动索引"更安全。

## 顺序不存在表里

表是堆存储，**没有任何顺序被保存** —— 看到的每条顺序都是查询时用 `ORDER BY` 现算出来的，编辑笔记不会移动任何一行。所以所有列表查询必须显式排序，且必须带兜底列：

```sql
ORDER BY created_at DESC, id ASC   -- 默认
ORDER BY updated_at DESC, id ASC   -- ?sort=updated
```

兜底列不能省：没有它，同一毫秒创建的笔记相对顺序由 SQLite 自行决定，树会莫名跳动。

## 标签：两条全表改写

标签没有表，改名 / 删除就是扫过全部未删笔记改写 JSON 列：

```sql
-- 改标签名（参数依次：旧名、新名、新的 updated_at、旧名）
-- 外层 DISTINCT 用于「合并」：某行同时有 rust 与 rust-lang 时，改名后只会剩一个
UPDATE notes SET
    tags = (SELECT json_group_array(DISTINCT v) FROM (
                SELECT CASE WHEN value = ? THEN ? ELSE value END AS v
                FROM json_each(notes.tags))),
    updated_at = ?
  WHERE deleted_at IS NULL
    AND EXISTS (SELECT 1 FROM json_each(notes.tags) WHERE value = ?);

-- 删标签（参数依次：标签名、新的 updated_at、标签名）
UPDATE notes SET
    tags = (SELECT json_group_array(value) FROM json_each(notes.tags) WHERE value <> ?),
    updated_at = ?
  WHERE deleted_at IS NULL
    AND EXISTS (SELECT 1 FROM json_each(notes.tags) WHERE value = ?);
```

三点实现要点：

- **两条都要同时写 `updated_at`**：标签是笔记内容的一部分，与"改某篇笔记的 tags"是同一种变更，不该因为走哪条路由而给出不同答案。
- **只匹配未软删的笔记**。`EXISTS` 那一段同时决定 404：没有任何未删笔记在用该标签时 `RowsAffected = 0`。
- **两条都不触发 `notes_au`**（它的 `UPDATE OF` 只列了 `title, content`），不会白重建 FTS。

已在 modernc sqlite 3.46 上实测：`json_each(NULL)` 返回 0 行且不报错（所以 `tags` 可为 NULL）；`json_group_array` 过滤后为空返回 `'[]'` 而不是 NULL（删掉最后一个标签不会毁掉该列）；匹配是精确的（`ru` 不命中 `rust`）；`DISTINCT` 完成撞名合并。

## 建库：不提供迁移

**本项目不做 schema 迁移。** 唯一支持的状态就是本文件的这一个 schema；改表结构 = 删掉 `data/` 下的仓库目录，启动时重建。

因此 `store.migrateRepo` **整段删除、不留备用**（代码尚未跟进）。留着比删掉危险：它用 `columnExists(notes, "content")` 判断"这是老库"，据此把 `content` 改回 `data`。而在本 schema 下**这个判据恒为真**（目标列名就叫 `content`），每次打开库都会试图改回去，直接把新库改坏。

代价：以后改 schema，老库不会自动升级，得手动处理。单用户本地应用，这个代价换来的是一整块不再需要维护的代码。
