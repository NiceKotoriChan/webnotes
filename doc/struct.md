个人笔记软件
Client-Server 架构，网页支持 PWA，Sqlite 数据库
后端go 前端SPA TypeScript + Svelte + Vite；

tags 扁平结构
refs 层级结构

要考虑附件

目录分片分 2级，每级2个hex字符
全局搜索，支持标题
Note ID:
550e8400-e29b-41d4-a716-446655440000

Storage Key:
notes/55/0e/550e8400-e29b-41d4-a716-446655440000.md

```sql
CREATE TABLE notes (
    id    TEXT PRIMARY KEY, -- UUIDv4
    title TEXT NOT NULL,
    ctime INTEGER NOT NULL,
    mtime INTEGER NOT NULL
);

CREATE TABLE tags (
    id    TEXT PRIMARY KEY, -- UUIDv4
    name  TEXT NOT NULL UNIQUE
);

CREATE TABLE note_tags (
    note_id  TEXT NOT NULL,
    tag_id   TEXT NOT NULL,

    PRIMARY KEY (note_id, tag_id),

    FOREIGN KEY (note_id)
        REFERENCES notes(id)
        ON DELETE CASCADE,

    FOREIGN KEY (tag_id)
        REFERENCES tags(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE
);

CREATE INDEX idx_note_tags_tag
    ON note_tags(tag_id);

CREATE TABLE refs (
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    ctime     INTEGER NOT NULL,

    PRIMARY KEY (source_id, target_id),

    FOREIGN KEY (source_id)
        REFERENCES notes(id)
        ON DELETE CASCADE,

    FOREIGN KEY (target_id)
        REFERENCES notes(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_refs_source
    ON refs(source_id);

CREATE INDEX idx_refs_target
    ON refs(target_id);
```

```
repos/
└── notes/
    └── 55/
        └── 0e/
            └── 550e8400-e29b-41d4-a716-446655440000.md
    attachments/
        └── 550e8400-e29b-41d4-a716-446655440000.jpg
```

Content-Type: text/markdown; charset=utf-8
