# 数据模型

> **与代码同步** —— 真相源是 `server/api/*.go` 的 struct 与 `server/store/store.go` 的 schema。字段或行为变化时同步本文档。

约定：时间戳一律 Unix 毫秒；JSON 字段 snake_case；图标名为 mdi 名（不含 `mdi:` 前缀）。存储层的列定义见 [schema.md](schema.md)。

## Note（笔记）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | string | UUID，不可变 |
| parent_id | string \| null | 父笔记 id；null = 根节点 |
| title | string | 标题 |
| content | string | 正文（Markdown） |
| created_at | number | 创建时间。**写入后再不改动**，即原 `date` |
| updated_at | number | 内容最后修改时间。刷新时机见下 |
| deleted_at | number \| null | 软删时间；null = 正常（回收站） |
| tags | string[] | 标签名数组。**恒为数组**，无标签时是 `[]` |
| icon | string \| null | 自定义图标名（mdi）；null = 由前端按标题自动匹配 |

### 行为语义

字段定义之外，这些行为直接影响界面表现，是最容易误解的部分：

- **`created_at` 不会被编辑刷新。** 而树与列表默认按 `created_at DESC` 排序，所以"改过的笔记不会冒泡到顶部"是**预期行为**，不是 bug。
- **`updated_at` 只在内容变更时刷新**：`title` / `content` / `tags` 的写入会刷新它；`parent_id`（移动）与 `icon`（换图标）不会。理由是 `updated_at` 表示"内容最后被修改的时间"，而拖拽和换图标是结构与展示层面的动作 —— 让它们刷新时间戳，用户在 `?sort=updated` 视图里就会因为一次拖拽看到笔记跳位。
- **软删除是递归的**：删除一个节点会连同其整棵子树一起标记 `deleted_at`，再对已是删除状态的节点重复删除会得到 404。标签是行内数据，跟着一起隐身，还原时原样回来。
- **还原同样是递归的**：若原父节点仍在回收站里，该笔记会被挂到根节点 —— 避免出现"父在回收站、子在树里"的悬空状态。
- **移动要防环**：服务端沿 `parent_id` 链向上检测，拒绝把节点移到自身或自己的后代。
- **排序必须带兜底列**（`ORDER BY <字段> DESC, id ASC`）：同一毫秒创建的笔记若没有兜底列，相对顺序由 SQLite 自行决定，可能每次刷新都不同。
- **`icon` 为 null 时由前端自动匹配**：按标题的关键词与扩展名映射到 mdi 图标名（`client/src/lib/autoIcon.ts`）。

## 标签不是实体

这是这次数据模型里最大的一处变化，值得单独说清：**标签只是 `notes.tags` 这个 JSON 数组里的字符串，库里没有标签表，标签没有 id。**

由此推出几条必须接受的后果：

- **没有"新建标签"这个操作**。创建标签就是给某篇笔记写上这个字符串。
- **标签列表是从笔记派生的**（扫全部未删笔记的 `tags` 去重）。所以**一篇笔记都不用的标签根本不存在** —— 不存在"孤儿标签"，也就没有"清理未使用标签"这件事要做。
- **重命名或删除标签是全表改写**（语句见 [schema.md](schema.md#为什么没有标签表)），影响的笔记数就是响应里的 `count`。两条语句都会**刷新受影响笔记的 `updated_at`** —— 标签是内容的一部分，和"改某篇笔记的 tags"是同一种变更，不该因为走哪条路由而给出不同答案。回收站里的笔记不参与改写。
- **改名撞上已存在的名字时两个标签合并**，而不是报错。这符合字符串语义：合并后就是同名的那些笔记共用一个标签。
- **按标签筛选不能走索引**。单用户本地应用、几千条笔记的量级下无关紧要；如果哪天量级上来了，再考虑把标签拆回关联表。

跨仓库不共享：每个仓库的 `data.db` 各有一份自己的标签集合。

## Repo（仓库）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | string | UUID，即磁盘目录名，不可变 |
| name | string | 显示名，可改（改名不动目录与链接） |
| date | number | 创建时间 |

一个仓库对应一个自包含目录 `<data>/<uuid>/`，内含 `data.db`（笔记与附件元数据）与 `assets/`（附件文件）。`data/repos.json` 只是索引，**目录才是真相源** —— 启动时 `store.reconcile` 双向对账：目录有而索引无则补一条（名字回退"未命名"），索引有而目录无则丢弃。

> **已知缺陷**：`reconcile` 补的那条索引没有 `date`（零值），也不会回填旧索引里缺失的 `date`。当前 `data/repos.json` 两个仓库的 `date` 都是 `0`，于是"按创建时间排序"形同虚设。修法是补索引时用目录的创建时间、或直接 `time.Now()`。

## Asset（附件）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | string | sha256，内容寻址（同一仓库内同内容自动去重，只存一份） |
| name | string | 文件名 |
| mime | string | MIME |
| size | number | 字节数 |
| date | number | 上传时间 |
| status | `uploading` \| `ready` \| `deleting` | 上传状态机 |

**`status` 只在 `ready` 时可见**：列表接口与下载都只认 `ready`，`uploading` / `deleting` 是崩溃恢复用的中间态。启动时 `asset.Reconcile` 逐仓库对账：清掉孤儿 tmp 文件、`uploading` 但落盘文件已就位则置 `ready`、`deleting` 则重删文件并清行。

附件的 `name` / `mime` 是**上传时**的记录，与文件内容无关；`id` 才是内容的身份。因此同一份内容以不同文件名上传两次只会存一份，第二次命中秒传，`name` / `mime` 保持第一次的值。

秒传时客户端不再自己拼元数据 —— 先 `GET /assets/:sha/meta` 拿服务端的权威记录（见 [api.md](api.md#meta为什么不用-head)），因此"界面显示的文件名"与库里记录始终一致。

## 变更历史

| 变更 | 说明 |
|---|---|
| 删去 `notes.mtime` | 早于当前形态 |
| `notes.content` → `data` → **`content`** | 绕了一圈回到 `content`。旧迁移逻辑随之整段删除，见 [schema.md](schema.md#建库不提供迁移) |
| `notes.ctime` → `date` → **`created_at` + `updated_at`** | 单列拆成两列，旧值回填 `created_at` |
| `tags` 表 + `note_tags` 表 → **`notes.tags` 一列 JSON** | 标签失去实体身份，见上文 |
| `assets.ctime` → `date` | 附件仍用 `date`，未随 notes 拆分 |

**本项目不做 schema 迁移**：`store.migrateRepo` 已整段删除，唯一支持的状态就是当前 schema，旧库直接重建。理由与代价见 [schema.md](schema.md#建库不提供迁移)。
