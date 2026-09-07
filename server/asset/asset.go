// Package asset 操作 assets/ 目录与 index.db 中的 assets 表
package asset

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"webnotes/server/store"
)

type Store struct {
	dataDir string
	indexDB *sql.DB
}

func New(s *store.Store) *Store {
	return &Store{dataDir: s.DataDir, indexDB: s.IndexDB}
}

// ReadyPath 返回 sha256 落盘路径 assets/ab/cd/<sha256>
func (a *Store) ReadyPath(id string) string {
	return filepath.Join(a.dataDir, "assets", id[:2], id[2:4], id)
}

func (a *Store) TmpPath(id string) string {
	return filepath.Join(a.dataDir, "assets", "tmp", id)
}

// Status 返回 id 的 status；空串表示不存在
func (a *Store) Status(id string) (string, error) {
	var st string
	err := a.indexDB.QueryRow(`SELECT status FROM assets WHERE id = ?`, id).Scan(&st)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return st, err
}

// InsertUploading 插入 uploading 行；冲突（已存在）返回 false
func (a *Store) InsertUploading(id, name, mime string, size int64, ctime int64) (bool, error) {
	res, err := a.indexDB.Exec(
		`INSERT INTO assets (id, name, mime, size, status, ctime)
		 VALUES (?, ?, ?, ?, 'uploading', ?)`,
		id, name, mime, size, ctime,
	)
	if err != nil {
		return false, nil // UNIQUE 冲突
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// Save 流式接收 body：写 tmp → 校验 sha256 → rename → 置 ready
func (a *Store) Save(id, name, mime string, size int64, r io.Reader) error {
	tmpPath := a.TmpPath(id)
	if err := os.MkdirAll(filepath.Dir(tmpPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), r); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != id {
		os.Remove(tmpPath)
		return ErrShaMismatch
	}

	readyPath := a.ReadyPath(id)
	if err := os.MkdirAll(filepath.Dir(readyPath), 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, readyPath); err != nil {
		return err
	}
	_, err = a.indexDB.Exec(`UPDATE assets SET status='ready' WHERE id = ?`, id)
	return err
}

// Remove 删除：仅 status='ready' 时允许；删文件 + 删行
func (a *Store) Remove(id string) error {
	res, err := a.indexDB.Exec(
		`UPDATE assets SET status='deleting' WHERE id = ? AND status='ready'`, id,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotReady
	}
	if err := os.Remove(a.ReadyPath(id)); err != nil && !os.IsNotExist(err) {
		return err
	}
	_, err = a.indexDB.Exec(`DELETE FROM assets WHERE id = ?`, id)
	return err
}

// CleanupUploading 删除 uploading 状态的行（用于 sha 校验失败后清理）
func (a *Store) CleanupUploading(id string) {
	a.indexDB.Exec(`DELETE FROM assets WHERE id = ? AND status='uploading'`, id)
}

// EnsureSameDir 检查 tmp 与 assets 在同一文件系统（rename 原子性前置条件）
func (a *Store) EnsureSameDir() error {
	tmpSt := syscall.Stat_t{}
	readySt := syscall.Stat_t{}
	if err := syscall.Stat(filepath.Join(a.dataDir, "assets", "tmp"), &tmpSt); err != nil {
		return err
	}
	if err := syscall.Stat(a.dataDir, &readySt); err != nil {
		return err
	}
	if tmpSt.Dev != readySt.Dev {
		return errors.New("assets/tmp and assets/ must be on the same filesystem")
	}
	return nil
}

// Reconcile 启动时对账：清孤儿 tmp、uploading→ready、deleting 重删/清行
func (a *Store) Reconcile() error {
	// 1. 清 tmp 目录：删除不在 index.db 里的 tmp 文件
	tmpDir := filepath.Join(a.dataDir, "assets", "tmp")
	if entries, err := os.ReadDir(tmpDir); err == nil {
		for _, e := range entries {
			id := e.Name()
			st, err := a.Status(id)
			if err != nil || st == "" || st == "ready" {
				os.Remove(filepath.Join(tmpDir, id))
			}
			// uploading 状态保留 tmp 文件
		}
	}

	// 2. status='uploading' 但 ready 文件已就位 → 置 ready
	rows, err := a.indexDB.Query(`SELECT id FROM assets WHERE status='uploading'`)
	if err != nil {
		return err
	}
	var uploadingIDs []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		uploadingIDs = append(uploadingIDs, id)
	}
	rows.Close()
	for _, id := range uploadingIDs {
		if info, err := os.Stat(a.ReadyPath(id)); err == nil && !info.IsDir() {
			a.indexDB.Exec(`UPDATE assets SET status='ready' WHERE id = ?`, id)
		}
	}

	// 3. status='deleting' 但文件仍在 → 重试删除；文件已删 → 删行
	rows, err = a.indexDB.Query(`SELECT id FROM assets WHERE status='deleting'`)
	if err != nil {
		return err
	}
	var deletingIDs []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		deletingIDs = append(deletingIDs, id)
	}
	rows.Close()
	for _, id := range deletingIDs {
		if err := os.Remove(a.ReadyPath(id)); err != nil && !os.IsNotExist(err) {
			continue
		}
		a.indexDB.Exec(`DELETE FROM assets WHERE id = ?`, id)
	}
	return nil
}

// Metadata 返回单条 assets 元数据
type Metadata struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Mime  string `json:"mime"`
	Size  int64  `json:"size"`
	Ctime int64  `json:"ctime"`
}

func (a *Store) Metadata(id string) (*Metadata, error) {
	var m Metadata
	err := a.indexDB.QueryRow(
		`SELECT id, name, mime, size, ctime FROM assets WHERE id = ?`, id,
	).Scan(&m.ID, &m.Name, &m.Mime, &m.Size, &m.Ctime)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &m, err
}

var (
	ErrShaMismatch = errors.New("sha256 mismatch")
	ErrNotReady    = errors.New("asset not ready")
)
