个人笔记软件。Client-Server，PWA，SQLite。
后端 Go gin，前端 TypeScript + Svelte + Vite。

每个仓库一个自包含目录：笔记/标签/附件元数据存 `data.db`，附件文件在同目录 `assets/`。
笔记树形组织（`parent_id`），标签扁平；附件 sha256 内容寻址、仓库内私有去重。
schema 见 [data.sql](./data.sql)。

## 目录结构

```
webnotes/
├── repos.json                  -- 全局仓库索引（可重建，非真相源）
├── a3f9c2e1-…/                 -- 目录名 = repo UUID，不可变
│   ├── data.db                 -- 笔记/标签/附件元数据
│   ├── data.db-wal / -shm
│   └── assets/                 -- 该仓库私有附件
│       ├── tmp/                -- 上传中转
│       └── ab/cd/abcd1234…     -- sha256 全文（64 hex），两级分片，无扩展名
└── b7d1e0f4-…/
    ├── data.db …
    └── assets/ …
```

repos.json：

```json
[{ "id": "a3f9c2e1-…", "name": "Linux 学习笔记", "ctime": 1725900000000 }]
```

- **id**：repo UUID，即目录名，创建后不变。API 与正文附件链接都用它
  （`/api/repos/<id>/assets/<sha256>`），改名不影响历史链接。
- **name**：可变显示名，改名 = 原子重写 repos.json（写 tmp → fsync → rename）。
- 真相源是 `webnotes/*/` 目录，repos.json 仅索引，启动对账：
  目录有/json 无 → 补（name 回退"未命名"）；json 有/目录无 → 删；文件丢 → 扫目录重建。

## 附件上传

附件表在各仓库 `data.db` 内，状态机 `uploading → ready → deleting → 删行`。

1. 客户端算 sha256，查本仓库 assets 表
   - `ready` → 秒传，直接返回
   - 不存在 → INSERT `uploading`
2. 流式写入 `assets/tmp/<sha256>`，边写边算 sha256 校验
3. fsync
4. rename → `assets/ab/cd/<sha256>`（同文件系统原子落盘）
5. UPDATE 该行 `status='ready'`
6. 前端正文引用 `/api/repos/<repo>/assets/<sha256>`

崩溃恢复（启动对账，均在本仓库 db + assets/ 内）：

- INSERT 后崩 → 有 uploading 行、无文件 → 清行
- rename 后、UPDATE 前崩 → 文件就位、仍 uploading → 置 ready
- 写 tmp 中途崩 → tmp 残留 → 清空 `assets/tmp/`

## 附件删除

1. UPDATE `status='deleting' WHERE id=? AND status='ready'`（0 行则拒绝）
2. 删 `assets/ab/cd/<sha256>`
3. DELETE 该行

崩溃恢复：

- 步骤 1 后崩 → 文件在、status=deleting → 重删或置回 ready
- 步骤 2 后、3 前崩 → 文件已删、行还在 → 清行

访问：`GET /api/repos/<repo>/assets/<sha256>`；name 仅用于 Content-Disposition 下载名。

## 备份

- 各仓库 `data.db` 用 sqlite backup API（或 `VACUUM INTO`）在线备份，勿直接 cp 正在写的库
- `repos.json` 一并复制（可重建，但保留可省去 name 回退）
- `assets/` 直接复制；内容不可变（sha256 寻址），复制中途无一致性问题
