个人笔记软件
Client-Server 架构，网页支持 PWA，SQLite 数据库
后端 Go，前端 SPA：TypeScript + Svelte + Vite

tags 扁平结构
refs 笔记间相互引用（独立边表，显式维护，不解析正文）
笔记扁平，无父子层级

每个 repo 一个 db
每个笔记（含正文）存 db 里
附件内容寻址，不可变，全局共享（跨 repo 去重）
附件不建元数据库：文件系统即注册表，sha256 即文件名

## 目录结构

```
repos/
├── notes.db                -- repo: notes（文件名即 repo id，启动扫描 *.db）
├── notes.db-wal
├── notes.db-shm
└── work.db                 -- repo: work
assets/
└── ab/
    └── cd/
        └── abcd1234...     -- sha256 全文，64 hex，无扩展名
```

- 附件分片：取 sha256 前 4 个 hex 字符，分两级各 2 个
- 文件无扩展名，无元数据库：MIME 响应时嗅探（Go http.DetectContentType），size 用 os.Stat
- 文件存在即存在，sha256 已落盘则直接复用
- 删除 repo：删掉对应 db 文件后跑一次 GC，其附件自动回收

## 内容指纹：SHA-256

- Go 标准库 crypto/sha256，无第三方依赖
- 抗碰撞强度远超个人笔记场景；Git / S3 / Docker 同系列
- 不用 MD5 / SHA-1；BLAKE3 更快但需第三方库，此规模无必要

## 附件写入流程

1. 前端上传文件
2. 服务端流式接收：边写 assets 目录下的临时文件（.tmp-<随机>）边算 sha256
3. 目标文件 assets/ab/cd/<sha256> 已存在 → 丢弃临时文件，直接复用（跨 repo 去重）
4. 否则 fsync 后 rename 到目标路径（原子落盘，无半截文件）
5. note_assets 关联与笔记保存在 repo db 的同一事务里提交
   （第 4、5 步之间崩溃只产生孤儿文件，由 GC 回收）

正文引用：/api/assets/<sha256>/<filename>
（filename 仅用于 URL 可读和下载名，查找只认 sha256）

## 附件 GC

附件不可变，删笔记只级联删所在 repo 的 note_assets 关联。启动时/后台：

1. 打开 repos/ 下所有 db，汇总 note_assets 中全部 sha256（引用集合）
2. 扫描 assets/ 目录，不在引用集合中的文件 → 删除
3. 临时文件残留（_.tmp-_）启动时直接清理

删除 repo = 删除其 db 文件后跑一次 GC。

## 备份

- repos/\*.db 用 sqlite backup API（或 VACUUM INTO）在线备份，不要直接 cp 正在写的库
- assets 目录一并复制；内容不可变，复制中途无一致性问题
