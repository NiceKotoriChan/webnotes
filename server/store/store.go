// Package store 管理 webnotes/ 根目录：repos.json 索引与各仓库自包含目录。
// 每个仓库是 <root>/<uuid>/ 目录，内含 data.db 与私有 assets/。
package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// ErrNotFound 仓库或记录不存在
var ErrNotFound = errors.New("not found")

// RepoInfo repos.json 中的一条索引
type RepoInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	CTime int64  `json:"ctime"`
}

type Store struct {
	Root  string
	mu    sync.Mutex
	repos []RepoInfo
}

func New(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	s := &Store{Root: root}
	if err := s.reconcile(); err != nil {
		return nil, err
	}
	return s, nil
}

// reconcile 启动对账：目录为真相源，repos.json 仅索引。
// 目录有/json 无 → 补（name 回退"未命名"）；json 有/目录无 → 丢弃。
func (s *Store) reconcile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	indexed := map[string]RepoInfo{}
	if data, err := os.ReadFile(s.indexPath()); err == nil {
		var list []RepoInfo
		if json.Unmarshal(data, &list) == nil {
			for _, r := range list {
				indexed[r.ID] = r
			}
		}
	}

	entries, _ := os.ReadDir(s.Root)
	merged := []RepoInfo{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		if _, err := uuid.Parse(id); err != nil {
			continue // 只认 UUID 目录
		}
		if _, err := os.Stat(filepath.Join(s.Root, id, "data.db")); err != nil {
			continue
		}
		if info, ok := indexed[id]; ok {
			merged = append(merged, info)
		} else {
			merged = append(merged, RepoInfo{ID: id, Name: "未命名"})
		}
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].CTime < merged[j].CTime })
	s.repos = merged
	return s.saveLocked()
}

// saveLocked 原子重写 repos.json（tmp → fsync → rename）。调用方需持锁。
func (s *Store) saveLocked() error {
	data, err := json.MarshalIndent(s.repos, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.indexPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if f, err := os.Open(tmp); err == nil {
		f.Sync()
		f.Close()
	}
	return os.Rename(tmp, s.indexPath())
}

func (s *Store) indexPath() string { return filepath.Join(s.Root, "repos.json") }

// RepoDir 返回仓库目录 <root>/<id>
func (s *Store) RepoDir(id string) string { return filepath.Join(s.Root, id) }

func (s *Store) ListRepos() []RepoInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]RepoInfo, len(s.repos))
	copy(out, s.repos)
	return out
}

// CreateRepo 生成 UUID、建目录与 data.db、写入索引
func (s *Store) CreateRepo(name string) (*RepoInfo, error) {
	id := uuid.NewString()
	dir := s.RepoDir(id)
	if err := os.MkdirAll(filepath.Join(dir, "assets", "tmp"), 0o755); err != nil {
		return nil, err
	}
	db, err := openDB(filepath.Join(dir, "data.db"))
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	if _, err := db.Exec(RepoSchema); err != nil {
		db.Close()
		os.RemoveAll(dir)
		return nil, err
	}
	db.Close()

	info := RepoInfo{ID: id, Name: name, CTime: time.Now().UnixMilli()}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repos = append(s.repos, info)
	if err := s.saveLocked(); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	return &info, nil
}

// RenameRepo 只改索引里的 name，目录与链接不动
func (s *Store) RenameRepo(id, name string) (*RepoInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.repos {
		if s.repos[i].ID == id {
			s.repos[i].Name = name
			if err := s.saveLocked(); err != nil {
				return nil, err
			}
			r := s.repos[i]
			return &r, nil
		}
	}
	return nil, ErrNotFound
}

// DeleteRepo 删除整个仓库目录并从索引移除
func (s *Store) DeleteRepo(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrNotFound
	}
	if err := os.RemoveAll(s.RepoDir(id)); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.repos[:0]
	for _, r := range s.repos {
		if r.ID != id {
			kept = append(kept, r)
		}
	}
	s.repos = kept
	return s.saveLocked()
}

// OpenRepo 打开已有仓库的 data.db 并确保 schema 与迁移；仓库不存在返回 ErrNotFound（不会新建空库）
func (s *Store) OpenRepo(id string) (*sql.DB, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrNotFound
	}
	dbPath := filepath.Join(s.RepoDir(id), "data.db")
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(RepoSchema); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrateRepo(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func openDB(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// SQLite 单写者；串行连接避免同文件多连接争锁
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

// columnExists 判断表是否已有某列
func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid       int
			name      string
			typ       string
			notnull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// migrateRepo 迁移旧库：按需补列（幂等）。触发器逻辑不变，无需重建。
func migrateRepo(db *sql.DB) error {
	for _, col := range []struct{ name, ddl string }{
		{"deleted_at", `ALTER TABLE notes ADD COLUMN deleted_at INTEGER`},
		{"icon", `ALTER TABLE notes ADD COLUMN icon TEXT`},
	} {
		has, err := columnExists(db, "notes", col.name)
		if err != nil {
			return err
		}
		if !has {
			if _, err := db.Exec(col.ddl); err != nil {
				return err
			}
		}
	}
	return nil
}

// RepoSchema 每个仓库 data.db 的结构，与 docs/data.sql 保持一致
const RepoSchema = `
CREATE TABLE IF NOT EXISTS assets (
    id     TEXT PRIMARY KEY,
    name   TEXT NOT NULL,
    mime   TEXT NOT NULL,
    size   INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'uploading'
           CHECK (status IN ('uploading','ready','deleting')),
    ctime  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS notes (
    id         TEXT PRIMARY KEY,
    parent_id  TEXT REFERENCES notes(id) ON DELETE CASCADE,
    title      TEXT NOT NULL DEFAULT '',
    content    TEXT NOT NULL DEFAULT '',
    ctime      INTEGER NOT NULL,
    mtime      INTEGER NOT NULL,
    deleted_at INTEGER,
    icon       TEXT
);
CREATE INDEX IF NOT EXISTS idx_notes_parent ON notes(parent_id);
CREATE INDEX IF NOT EXISTS idx_notes_mtime  ON notes(mtime DESC);

CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
    title, content,
    content = 'notes', content_rowid = 'rowid',
    tokenize = 'trigram'
);
` + ftsTriggers + `
CREATE TABLE IF NOT EXISTS tags (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS note_tags (
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    tag_id  TEXT NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);
CREATE INDEX IF NOT EXISTS idx_note_tags_tag ON note_tags(tag_id);`

// ftsTriggers 维护 notes ↔ notes_fts 同步（无条件）。软删除的笔记仍在索引中，
// 由查询层的 deleted_at IS NULL 过滤，避免 FTS5 外部内容表同步不一致。
const ftsTriggers = `
CREATE TRIGGER IF NOT EXISTS notes_ai AFTER INSERT ON notes BEGIN
    INSERT INTO notes_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
END;
CREATE TRIGGER IF NOT EXISTS notes_ad AFTER DELETE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES ('delete', old.rowid, old.title, old.content);
END;
CREATE TRIGGER IF NOT EXISTS notes_au AFTER UPDATE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES ('delete', old.rowid, old.title, old.content);
    INSERT INTO notes_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
END;`
