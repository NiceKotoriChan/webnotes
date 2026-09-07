// Package store 管理 index.db 与 repos/ 下的 repo db
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	DataDir string
	IndexDB *sql.DB
}

func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dataDir, "repos"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "assets", "tmp"), 0o755); err != nil {
		return nil, err
	}
	indexDB, err := openDB(filepath.Join(dataDir, "index.db"))
	if err != nil {
		return nil, err
	}
	s := &Store{DataDir: dataDir, IndexDB: indexDB}
	if err := s.ensureIndexSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

func openDB(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

const indexSchema = `
CREATE TABLE IF NOT EXISTS assets (
    id     TEXT PRIMARY KEY,
    name   TEXT NOT NULL,
    mime   TEXT NOT NULL,
    size   INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'uploading'
           CHECK (status IN ('uploading','ready','deleting')),
    ctime  INTEGER NOT NULL
);`

const RepoSchema = `
CREATE TABLE IF NOT EXISTS notes (
    id      TEXT PRIMARY KEY,
    title   TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    ctime   INTEGER NOT NULL,
    mtime   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_notes_mtime ON notes(mtime DESC);

CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
    title, content,
    content = 'notes', content_rowid = 'rowid',
    tokenize = 'trigram'
);
CREATE TRIGGER IF NOT EXISTS notes_ai AFTER INSERT ON notes BEGIN
    INSERT INTO notes_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
END;
CREATE TRIGGER IF NOT EXISTS notes_ad AFTER DELETE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES ('delete', old.rowid, old.title, old.content);
END;
CREATE TRIGGER IF NOT EXISTS notes_au AFTER UPDATE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES ('delete', old.rowid, old.title, old.content);
    INSERT INTO notes_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
END;

CREATE TABLE IF NOT EXISTS tags (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS note_tags (
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    tag_id  TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);
CREATE INDEX IF NOT EXISTS idx_note_tags_tag ON note_tags(tag_id);

CREATE TABLE IF NOT EXISTS refs (
    source_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    target_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    ctime     INTEGER NOT NULL,
    PRIMARY KEY (source_id, target_id),
    CHECK (source_id != target_id)
);
CREATE INDEX IF NOT EXISTS idx_refs_target ON refs(target_id);`

func (s *Store) ensureIndexSchema() error {
	_, err := s.IndexDB.Exec(indexSchema)
	return err
}

// RepoPath 返回 repos/<id>.db
func (s *Store) RepoPath(id string) string {
	return filepath.Join(s.DataDir, "repos", id+".db")
}

// OpenRepo 打开（必要时创建）repo db 并确保 schema
func (s *Store) OpenRepo(id string) (*sql.DB, error) {
	db, err := openDB(s.RepoPath(id))
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(RepoSchema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// ListRepos 扫描 repos/ 目录下的 *.db
func (s *Store) ListRepos() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.DataDir, "repos"))
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if name := e.Name(); filepath.Ext(name) == ".db" {
			ids = append(ids, name[:len(name)-3])
		}
	}
	return ids, nil
}

// DeleteRepo 删除 repo db 文件（含 WAL/SHM）
func (s *Store) DeleteRepo(id string) error {
	base := s.RepoPath(id)
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := os.Remove(base + suffix); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
