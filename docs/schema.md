# 数据库 schema

> **与代码同步** —— 本文是 schema 的真相源。实现是 `server/store/store.go` 里的 `RepoSchema` 常量，与本文逐字对应，只多了 `IF NOT EXISTS`（库每次打开都会执行一遍，必须幂等）。**改 schema 先改本文，再同步那个常量**；两者不一致视为 bug。

每个仓库一个自包含的 `data.db`，所在目录与 `assets/` 的关系见 [model.md](model.md)。约定：时间戳一律 Unix 毫秒；`id` 一律 UUID，`assets.id` 例外（它是内容寻址的 sha256）。

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

## notes

| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT | UUID 主键 |
| parent_id | TEXT | 父笔记 id；NULL = 根节点。`ON DELETE CASCADE` 让硬删子树一次到底 |
| title | TEXT | 标题，默认空串 |
| content | TEXT | 正文（Markdown） |
| created_at | INTEGER | 创建时间。**写入之后再不改动** |
| updated_at | INTEGER | 内容最后修改时间。刷新时机见 [api.md](api.md#updated_at-什么时候刷新) |
| deleted_at | INTEGER | 软删时间；NULL = 正常 |
| tags | TEXT | 标签，**JSON 数组**，如 `["rust","api"]` |
| icon | TEXT | mdi 图标名（不含 `mdi:` 前缀）；NULL = 由前端按标题自动匹配 |

### 为什么没有标签表

标签原先是一张 `tags` 表加一张 `note_tags` 关联表，现在收敛成 `notes` 上的一个 JSON 列。取舍写在明处：

- **得到**：一张表说清一件事；"这篇笔记有哪些标签"不用 join；打标签就是改这一行；不再需要维护关联表的一致性。
- **付出**：标签没有稳定 id，**重命名或删除一个标签要全表改写**那一列 JSON。
- **付出**：按标签筛选不能走索引，是全表扫。单用户本地应用、笔记量在几千条以内，实测无关紧要。

两条改写语句（改标签名 / 删标签），都已在本机 modernc sqlite 3.46 上跑通：

```sql
-- 改标签名（参数依次：旧名、新名、新的 updated_at、旧名）
-- 外层 DISTINCT 是为了「合并」：若某行同时有 rust 与 rust-lang，改名后会出现两个 rust-lang
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

四点实现要点：

- **两条都要同时写 `updated_at`，`count` 取 `RowsAffected`。** 标签属于笔记的**内容**，和「改某篇笔记的 tags」是同一种变更，不该因为走哪条路由而给出不同答案。路由语义见 [api.md](api.md#tags)。
- **只匹配未软删的笔记**（`deleted_at IS NULL`）。回收站里的笔记不参与改写、也不出现在 `GET /tags` 的结果里，还原后标签原样回来。
- `EXISTS` 那一段不只是优化，它本身就决定了 404：没有任何未删笔记在用这个标签时 `RowsAffected = 0`。
- 这两条 UPDATE **不会触发 `notes_au`**（它的 `UPDATE OF` 只列了 `title, content`），所以不会白重建一遍 FTS。已实测，`integrity-check` 通过。

五个已验证的行为，实现时可以直接依赖：

1. `json_each(NULL)` 返回 0 行且**不报错**，所以 `tags` 允许是 NULL，筛选与去重都不用包 `COALESCE`。
2. `json_group_array` 在过滤后为空时返回 `'[]'` 而**不是 NULL**，所以删掉最后一个标签不会把列写成 NULL。
3. 匹配是**精确**的：`tag=ru` 不会命中 `rust`（用 `value = ?` 而不是 `LIKE`，正是为此）。
4. `json_group_array(DISTINCT v)` 完成「合并」：`["rust","rust-lang"]` 经改名 rust→rust-lang 得到 `["rust-lang"]`，不是两个。
5. 未删除的行里，`tags` 恒为 `[]`，或缺省时为空串 —— 应用层统一写 `'[]'`，把「没有标签」收敛成一种表示。

应用层的规范化：写入前 trim、丢弃空串、去重；响应里 `tags` **恒为数组**（无标签时是 `[]`，不是 null）。

## assets

| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT | sha256，内容寻址：同一仓库内同内容只存一份 |
| date | INTEGER | 上传时间 |
| name / mime / size | TEXT / TEXT / INTEGER | 上传时的记录，与文件内容无关 |
| status | TEXT | `uploading` / `ready` / `deleting` 状态机，只有 `ready` 对外可见 |

> 命名**有意保留**：`assets.date` 是 `data.db` 里唯一还叫 `date` 的时间列（`notes` 已拆成 `created_at` / `updated_at`；仓库索引 `data/repos.json` 里的 `date` 同样保留）。附件不存在"创建时间 vs 修改时间"的区分，叫 `date` 更贴切，不做统一。

## 索引

| 索引 | 用途 |
|---|---|
| `idx_notes_parent` | 按父节点取直接子节点（树每展开一层都要用） |
| `idx_notes_created` | 列表默认排序 `created_at DESC`（原 `idx_notes_date` 的等价物） |

标签是 JSON 列，**没有索引**，筛选走全表扫。`updated_at` 也暂未建索引，因为默认排序不用它；等 `?sort=updated` 真的用起来再加 `idx_notes_updated`。

## 排序不在表里

**这个库的任何一张表都没有保存行的顺序。** 行按 rowid 物理摆放，`SELECT` 不带 `ORDER BY` 时的返回顺序是未定义的 —— 你看到的每一条顺序都是查询时算出来的。改 `updated_at` 只是改一个整数，没有任何一行被"挪动"。

所以所有列表查询都必须显式写 `ORDER BY`，并且带一个兜底列，保证值相同时顺序也稳定：

```sql
ORDER BY created_at DESC, id ASC
```

没有兜底列时，同一毫秒创建的几条笔记的相对顺序由 SQLite 自行决定，可能每次刷新都不一样 —— 树会莫名其妙地跳动。

## 触发器与 FTS

`notes_fts` 是 fts5 **外部内容表**（`content='notes'`、`content_rowid='rowid'`），自身不存原文，靠三个触发器与 `notes` 同步：

| 触发器 | 时机 | 作用 |
|---|---|---|
| `notes_ai` | AFTER INSERT | 插入索引行 |
| `notes_ad` | AFTER DELETE | 删除索引行（硬删时） |
| `notes_au` | **AFTER UPDATE OF title, content** | 换索引行：先 delete 旧值，再 insert 新值 |

`notes_au` 上的 `UPDATE OF title, content` 是刻意加的，不是可省的装饰。没有它，触发器会对**任何** UPDATE 重建索引 —— 包括只改标签、只刷 `updated_at`、只标 `deleted_at`。而软删是递归的，删一棵子树就会把子树上每条笔记都重索引一遍，纯属白干。

已在本机验证（modernc sqlite 3.46）：

- 只改 `tags` / `deleted_at` / `updated_at` → 触发器不触发、索引不动、`integrity-check` 通过；
- 改 `title` / `content` → 触发器触发，旧词搜不到、新词搜得到；
- 判定依据是**列出现在 SET 里**，不看值是否真的变了。所以 `SET content = <同一个值>` 也会重建一次，这点开销可以接受。

**软删除的笔记仍留在索引里**，靠查询层的 `deleted_at IS NULL` 过滤。这是刻意的：外部内容表一旦部分同步就容易不一致，让触发器无条件维护索引、把过滤放到查询层，比让触发器去猜"这次软删要不要动索引"更安全。

## 建库：不提供迁移

**本项目不做 schema 迁移。** 唯一支持的状态就是本文这一个 schema。

- 落地时直接删掉 `data/` 下已有的仓库目录，启动时按上面的 SQL 建新库。本轮丢弃的数据：2 个仓库、23 条笔记、1 个附件（标签两张表本来就是 0 行）。
- 因此 `store.migrateRepo` **整段删除**，包括它原有的 `content→data` / `ctime→date` / 删 `mtime` 逻辑。
- `RepoSchema` 在每次打开库时都会执行一遍，所以里面每条语句都必须带 `IF NOT EXISTS`。

> **为什么要连旧迁移代码一起删干净**，而不是留着备用：那段逻辑用 `columnExists(db, "notes", "content")` 判断"这是老库"，据此把 `content` 改回 `data`。而在本文 schema 下**这个判据恒为真** —— 新库的目标列名就叫 `content`。于是每次打开库都会试图把列改回去，直接把新库改坏。留着它比删掉危险得多。

代价是明确的：以后若要改 schema，老库不会被自动升级，得手动处理。对单用户本地应用，这个代价换来的是一整块不再需要维护、也不可能再咬人的代码。
