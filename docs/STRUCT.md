个人笔记软件
Client-Server，PWA，SQLite
后端 Go gin 框架，前端 TypeScript + Svelte + Vite

每个 repo 一个 db，笔记正文存 db 内，无文件/数据库一致性问题
tags 扁平，refs 独立边表（显式维护，不解析正文）
附件 sha256 内容寻址，全局共享去重，元数据存 index.db

## 目录结构

```
repos/
├── linux.db                -- repo: linux（文件名即 repo id，启动扫描 *.db）
├── linux.db-wal
├── linux.db-shm
├── work.db                 -- repo: work
├── work.db-wal
└── work.db-shm
assets/
└── ab/
    └── cd/
        └── abcd1234...     -- sha256 全文，64 hex，无扩展名
index.db                    -- 全局 assets 注册表
index.db-wal
index.db-shm
```

## 附件写入流程

1. 客户端算 sha256，查 assets 表
   - status='ready' → 秒传，返回 sha256
   - 不存在 → INSERT status='uploading'
2. 客户端上传，服务端流式写入 assets/tmp/<sha256>
3. fsync
4. rename → assets/ab/cd/<sha256>（原子落盘，无半截文件）
5. index.db update status='ready'
6. 返回 sha256，前端在 Markdown 正文里引用 /api/assets/<sha256>

崩溃恢复：

- 步骤 1 INSERT 后崩 → assets 有 uploading 记录，无文件 → 对账清行
- 步骤 4 后、5 前崩 → 文件已就位，assets 仍 uploading → 对账置 ready
- 步骤 2-4 中途崩 → tmp 残留 → 启动时清理 assets/tmp/

## 附件删除流程

1. index.db update status='deleting' WHERE id=? AND status='ready'（受影响行数=0 时拒绝）
2. 删除 assets/ab/cd/<sha256>
3. index.db delete 行

崩溃恢复：

- 步骤 1 后崩 → 文件仍在，status='deleting' → 对账重新删除或置回 ready
- 步骤 2 后、3 前崩 → 文件已删，assets 仍有 deleting 记录 → 对账清行

访问：/api/assets/<sha256>
（name 用于 Content-Disposition 下载名，查找只认 sha256）

## 备份

- 所有 db（index.db + repos/\*.db）用 sqlite backup API（或 VACUUM INTO）在线备份，
  不要直接 cp 正在写的库
- assets 目录一并复制；内容不可变，复制中途无一致性问题
