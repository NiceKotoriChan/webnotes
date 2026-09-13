// Package asset 操作单个仓库内的 assets/ 目录与 data.db 中的 assets 表。
// 附件仓库内私有：所有方法都显式传入该仓库的 *sql.DB 与目录路径。
package asset

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
)

type Store struct{}

func New() *Store { return &Store{} }

var (
	ErrShaMismatch = errors.New("sha256 mismatch")
	ErrNotReady    = errors.New("asset not ready")
)

// Metadata 单条附件元数据
type Metadata struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Mime  string `json:"mime"`
	Size  int64  `json:"size"`
	Ctime int64  `json:"ctime"`
}

// ReadyPath 落盘路径 <dir>/assets/ab/cd/<sha>
func (a *Store) ReadyPath(dir, id string) string {
	return filepath.Join(dir, "assets", id[:2], id[2:4], id)
}

// TmpPath 上传中转路径 <dir>/assets/tmp/<sha>
func (a *Store) TmpPath(dir, id string) string {
	return filepath.Join(dir, "assets", "tmp", id)
}

// Status 返回 status；空串表示不存在
func (a *Store) Status(db *sql.DB, id string) (string, error) {
	var st string
	err := db.QueryRow(`SELECT status FROM assets WHERE id = ?`, id).Scan(&st)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return st, err
}

// InsertUploading 插入 uploading 行；冲突（已存在）返回 false
func (a *Store) InsertUploading(db *sql.DB, id, name, mime string, size, ctime int64) (bool, error) {
	res, err := db.Exec(
		`INSERT INTO assets (id, name, mime, size, status, ctime)
		 VALUES (?, ?, ?, ?, 'uploading', ?)`,
		id, name, mime, size, ctime,
	)
	if err != nil {
		return false, nil // PRIMARY KEY 冲突
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// Save 流式接收 body：写 tmp → 校验 sha256 → rename → 置 ready
func (a *Store) Save(db *sql.DB, dir, id, name, mime string, size int64, r io.Reader) error {
	tmpPath := a.TmpPath(dir, id)
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

	readyPath := a.ReadyPath(dir, id)
	if err := os.MkdirAll(filepath.Dir(readyPath), 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, readyPath); err != nil { // tmp 与分片目录同在 assets/ 下，同文件系统原子 rename
		return err
	}
	_, err = db.Exec(`UPDATE assets SET status='ready' WHERE id = ?`, id)
	return err
}

// Remove 删除：仅 status='ready' 时允许；删文件 + 删行
func (a *Store) Remove(db *sql.DB, dir, id string) error {
	res, err := db.Exec(
		`UPDATE assets SET status='deleting' WHERE id = ? AND status='ready'`, id,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotReady
	}
	if err := os.Remove(a.ReadyPath(dir, id)); err != nil && !os.IsNotExist(err) {
		return err
	}
	_, err = db.Exec(`DELETE FROM assets WHERE id = ?`, id)
	return err
}

// CleanupUploading 删 uploading 行（sha 校验失败后清理）
func (a *Store) CleanupUploading(db *sql.DB, id string) {
	db.Exec(`DELETE FROM assets WHERE id = ? AND status='uploading'`, id)
}

// List 列出该仓库全部 ready 附件，按上传时间倒序
func (a *Store) List(db *sql.DB) ([]Metadata, error) {
	rows, err := db.Query(
		`SELECT id, name, mime, size, ctime FROM assets WHERE status='ready' ORDER BY ctime DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Metadata{}
	for rows.Next() {
		var m Metadata
		if err := rows.Scan(&m.ID, &m.Name, &m.Mime, &m.Size, &m.Ctime); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Metadata 返回单条元数据；不存在返回 (nil, nil)
func (a *Store) Metadata(db *sql.DB, id string) (*Metadata, error) {
	var m Metadata
	err := db.QueryRow(
		`SELECT id, name, mime, size, ctime FROM assets WHERE id = ?`, id,
	).Scan(&m.ID, &m.Name, &m.Mime, &m.Size, &m.Ctime)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &m, err
}

// Reconcile 启动对账（单仓库）：清孤儿 tmp、uploading→ready、deleting 重删/清行
func (a *Store) Reconcile(db *sql.DB, dir string) error {
	// 1. 清孤儿 tmp：无记录或已 ready 的 tmp 文件删除
	tmpDir := filepath.Join(dir, "assets", "tmp")
	if entries, err := os.ReadDir(tmpDir); err == nil {
		for _, e := range entries {
			id := e.Name()
			st, err := a.Status(db, id)
			if err != nil || st == "" || st == "ready" {
				os.Remove(filepath.Join(tmpDir, id))
			}
		}
	}

	// 2. uploading 但 ready 文件已就位 → 置 ready
	for _, id := range a.idsByStatus(db, "uploading") {
		if info, err := os.Stat(a.ReadyPath(dir, id)); err == nil && !info.IsDir() {
			db.Exec(`UPDATE assets SET status='ready' WHERE id = ?`, id)
		}
	}

	// 3. deleting：文件在则重删，文件已删则清行
	for _, id := range a.idsByStatus(db, "deleting") {
		if err := os.Remove(a.ReadyPath(dir, id)); err != nil && !os.IsNotExist(err) {
			continue
		}
		db.Exec(`DELETE FROM assets WHERE id = ?`, id)
	}
	return nil
}

func (a *Store) idsByStatus(db *sql.DB, status string) []string {
	rows, err := db.Query(`SELECT id FROM assets WHERE status = ?`, status)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}
