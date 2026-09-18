package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// nowMillis 时间戳一律 Unix 毫秒
func nowMillis() int64 {
	return time.Now().UnixMilli()
}

// Note 对外形状，见 spec/model.md
type Note struct {
	ID        string   `json:"id"`
	ParentID  *string  `json:"parent_id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
	DeletedAt *int64   `json:"deleted_at"`
	Tags      []string `json:"tags"`
	Icon      *string  `json:"icon"`
}

const noteCols = `n.id, n.parent_id, n.title, n.content, n.created_at, n.updated_at, n.deleted_at, n.tags, n.icon`

// GET /api/repos/:repo/notes?q=&tag=&parent_id=&limit=&offset=
func (s *Server) listNotes(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	q := c.Query("q")
	tag := c.Query("tag")
	parentID, hasParent := c.GetQuery("parent_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// trigram 至少要 3 个字符才切得出词，短查询降级 LIKE
	useFTS := utf8.RuneCountInString(q) >= 3

	joins := strings.Builder{}
	conds := []string{"n.deleted_at IS NULL"}
	args := []any{}
	switch {
	case useFTS:
		// MATCH 的左边必须是 FTS 表本身，**不能写表别名** —— `f MATCH ?` 会被当成
		// 一个叫 f 的列，报 "no such column: f"。所以这里不给 notes_fts 起别名。
		joins.WriteString(`JOIN notes_fts ON notes_fts.rowid = n.rowid `)
		conds = append(conds, "notes_fts MATCH ?")
		args = append(args, q)
	case q != "":
		conds = append(conds, "(n.title LIKE ? OR n.content LIKE ?)")
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	if tag != "" {
		// 精确匹配，全表扫（标签没有表）
		conds = append(conds, "EXISTS (SELECT 1 FROM json_each(n.tags) WHERE json_each.value = ?)")
		args = append(args, tag)
	}
	if hasParent {
		if parentID == "" {
			conds = append(conds, "n.parent_id IS NULL")
		} else {
			conds = append(conds, "n.parent_id = ?")
			args = append(args, parentID)
		}
	}

	query := `SELECT ` + noteCols + ` FROM notes n ` + joins.String() + `WHERE ` + strings.Join(conds, " AND ")
	if useFTS {
		query += " ORDER BY notes_fts.rank" // 相关度是数据库给的，这里不是展示排序
	}
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		internal(c, err)
		return
	}
	notes, err := scanNotes(rows)
	if err != nil {
		internal(c, err)
		return
	}
	c.JSON(http.StatusOK, notes)
}

// POST /api/repos/:repo/notes  body: { title?, content?, parent_id?, tags?, icon? }
func (s *Server) createNote(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	p, ok := parseNotePatch(c)
	if !ok {
		return
	}

	now := nowMillis()
	n := Note{
		ID:        uuid.NewString(),
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      []string{},
	}
	if p.title != nil {
		n.Title = *p.title
	}
	if p.content != nil {
		n.Content = *p.content
	}
	if p.tags != nil {
		n.Tags = normalizeTags(*p.tags)
	}
	if p.icon != nil {
		n.Icon = p.icon
	}
	if p.parentID != nil {
		if !noteExists(db, *p.parentID) {
			badRequest(c, "parent note not found")
			return
		}
		n.ParentID = p.parentID
	}

	if _, err := db.Exec(
		`INSERT INTO notes (id, parent_id, title, content, created_at, updated_at, tags, icon)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.ParentID, n.Title, n.Content, n.CreatedAt, n.UpdatedAt, encodeTags(n.Tags), n.Icon,
	); err != nil {
		internal(c, err)
		return
	}
	c.JSON(http.StatusCreated, n)
}

// GET /api/repos/:repo/notes/:id
func (s *Server) getNote(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	n, err := fetchNote(db, c.Param("id"))
	if errors.Is(err, sql.ErrNoRows) {
		notFound(c, "note not found")
		return
	}
	if err != nil {
		internal(c, err)
		return
	}
	c.JSON(http.StatusOK, n)
}

// POST /api/repos/:repo/notes/:id  body 里出现的字段才改，null 语义见 spec/api.md
func (s *Server) updateNote(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	p, ok := parseNotePatch(c)
	if !ok {
		return
	}
	id := c.Param("id")

	sets := []string{}
	args := []any{}
	refresh := false // 只有 title / content / tags 会刷新 updated_at

	if p.title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *p.title)
		refresh = true
	}
	if p.content != nil {
		sets = append(sets, "content = ?")
		args = append(args, *p.content)
		refresh = true
	}
	if p.tags != nil {
		sets = append(sets, "tags = ?")
		args = append(args, encodeTags(*p.tags))
		refresh = true
	}
	if p.iconSet {
		sets = append(sets, "icon = ?")
		args = append(args, p.icon)
	}
	if p.parentSet {
		if p.parentID != nil {
			pid := *p.parentID
			if pid == id {
				badRequest(c, "cannot move note under itself")
				return
			}
			if !noteExists(db, pid) {
				badRequest(c, "parent note not found")
				return
			}
			if isAncestor(db, pid, id) {
				badRequest(c, "cannot move note under its own descendant")
				return
			}
		}
		sets = append(sets, "parent_id = ?")
		args = append(args, p.parentID)
	}
	if len(sets) == 0 {
		badRequest(c, "no updatable field")
		return
	}
	if refresh {
		sets = append(sets, "updated_at = ?")
		args = append(args, nowMillis())
	}
	args = append(args, id)

	res, err := db.Exec(
		`UPDATE notes SET `+strings.Join(sets, ", ")+` WHERE id = ? AND deleted_at IS NULL`, args...,
	)
	if err != nil {
		internal(c, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		notFound(c, "note not found")
		return
	}
	n, err := fetchNote(db, id)
	if err != nil {
		internal(c, err)
		return
	}
	c.JSON(http.StatusOK, n)
}

// POST /api/repos/:repo/notes/:id/delete  body: { permanent? }
func (s *Server) deleteNote(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	var body struct {
		Permanent bool `json:"permanent"`
	}
	if err := c.ShouldBindJSON(&body); err != nil && !errors.Is(err, io.EOF) {
		badRequest(c, "invalid json")
		return
	}
	id := c.Param("id")

	var (
		res sql.Result
		err error
	)
	if body.Permanent {
		// 硬删：外键 ON DELETE CASCADE 带走整棵子树，FTS 由触发器清
		res, err = db.Exec(`DELETE FROM notes WHERE id = ?`, id)
	} else {
		// 软删整棵子树；已软删的节点重复软删会匹配到 0 行 → 404
		res, err = db.Exec(`
			WITH RECURSIVE subtree(id) AS (
				SELECT id FROM notes WHERE id = ? AND deleted_at IS NULL
				UNION ALL
				SELECT n.id FROM notes n JOIN subtree s ON n.parent_id = s.id
			)
			UPDATE notes SET deleted_at = ? WHERE id IN (SELECT id FROM subtree)`,
			id, nowMillis())
	}
	if err != nil {
		internal(c, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		notFound(c, "note not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

// POST /api/repos/:repo/notes/:id/restore  还原该笔记及整棵子树（不刷新 updated_at）
func (s *Server) restoreNote(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	id := c.Param("id")
	var parent sql.NullString
	err := db.QueryRow(
		`SELECT parent_id FROM notes WHERE id = ? AND deleted_at IS NOT NULL`, id,
	).Scan(&parent)
	if errors.Is(err, sql.ErrNoRows) {
		notFound(c, "note not found in trash")
		return
	}
	if err != nil {
		internal(c, err)
		return
	}

	if _, err := db.Exec(`
		WITH RECURSIVE subtree(id) AS (
			SELECT id FROM notes WHERE id = ?
			UNION ALL
			SELECT n.id FROM notes n JOIN subtree s ON n.parent_id = s.id
		)
		UPDATE notes SET deleted_at = NULL WHERE id IN (SELECT id FROM subtree)`, id); err != nil {
		internal(c, err)
		return
	}

	// 原父节点还在回收站 → 挂到根，避免「父在回收站、子在树里」
	if parent.Valid {
		var stillDeleted int64
		if err := db.QueryRow(
			`SELECT deleted_at IS NOT NULL FROM notes WHERE id = ?`, parent.String,
		).Scan(&stillDeleted); err != nil {
			internal(c, err)
			return
		}
		if stillDeleted != 0 {
			if _, err := db.Exec(`UPDATE notes SET parent_id = NULL WHERE id = ?`, id); err != nil {
				internal(c, err)
				return
			}
		}
	}

	n, err := fetchNote(db, id)
	if err != nil {
		internal(c, err)
		return
	}
	c.JSON(http.StatusOK, n)
}

// GET /api/repos/:repo/trash  只列「顶层」被删笔记，按 deleted_at DESC
func (s *Server) listTrash(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT ` + noteCols + ` FROM notes n
		WHERE n.deleted_at IS NOT NULL
		  AND (n.parent_id IS NULL OR NOT EXISTS (
		      SELECT 1 FROM notes p WHERE p.id = n.parent_id AND p.deleted_at IS NOT NULL
		  ))
		ORDER BY n.deleted_at DESC`)
	if err != nil {
		internal(c, err)
		return
	}
	notes, err := scanNotes(rows)
	if err != nil {
		internal(c, err)
		return
	}
	c.JSON(http.StatusOK, notes)
}

// notePatch POST /notes 与 POST /notes/:id 的 body。字段缺席与显式 null 语义不同，
// 所以按「出现即记录」解析 —— 见 spec/api.md 的那张表。
type notePatch struct {
	title     *string   // 缺席不动；null 不合法
	content   *string   // 同上
	parentSet bool      // 出现即记
	parentID  *string   // nil + parentSet = 移到根
	tags      *[]string // nil = 不动；null 等价 []
	iconSet   bool
	icon      *string // nil + iconSet = 清空
}

func parseNotePatch(c *gin.Context) (*notePatch, bool) {
	dec := json.NewDecoder(c.Request.Body)
	var raw map[string]json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		badRequest(c, "invalid json")
		return nil, false
	}

	p := &notePatch{}
	for key, val := range raw {
		isNull := string(val) == "null"
		switch key {
		case "title", "content":
			if isNull {
				badRequest(c, key+" cannot be null")
				return nil, false
			}
			var v string
			if err := json.Unmarshal(val, &v); err != nil {
				badRequest(c, key+" must be a string")
				return nil, false
			}
			if key == "title" {
				p.title = &v
			} else {
				p.content = &v
			}
		case "parent_id":
			p.parentSet = true
			if isNull {
				continue
			}
			var v string
			if err := json.Unmarshal(val, &v); err != nil {
				badRequest(c, "parent_id must be a string or null")
				return nil, false
			}
			p.parentID = &v
		case "tags":
			if isNull {
				empty := []string{}
				p.tags = &empty
				continue
			}
			var v []string
			if err := json.Unmarshal(val, &v); err != nil {
				badRequest(c, "tags must be an array of strings")
				return nil, false
			}
			p.tags = &v
		case "icon":
			p.iconSet = true
			if isNull {
				continue
			}
			var v string
			if err := json.Unmarshal(val, &v); err != nil {
				badRequest(c, "icon must be a string or null")
				return nil, false
			}
			p.icon = &v
		}
	}
	return p, true
}

// normalizeTags trim、丢空串、去重，保持原顺序
func normalizeTags(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]bool, len(in))
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// encodeTags 落库：空集写 []（不是 null）
func encodeTags(tags []string) string {
	b, err := json.Marshal(normalizeTags(tags))
	if err != nil {
		return "[]"
	}
	return string(b)
}

// parseTags 读列：NULL、空串、坏 JSON 都当无标签
func parseTags(raw sql.NullString) []string {
	if !raw.Valid || raw.String == "" {
		return []string{}
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw.String), &tags); err != nil {
		return []string{}
	}
	return normalizeTags(tags)
}

func scanNotes(rows *sql.Rows) ([]Note, error) {
	defer rows.Close()
	out := []Note{}
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *n)
	}
	return out, rows.Err()
}

// scanner 让 scanNote 同时吃 *sql.Row 与 *sql.Rows
type scanner interface{ Scan(dest ...any) error }

func scanNote(row scanner) (*Note, error) {
	var (
		n       Note
		parent  sql.NullString
		deleted sql.NullInt64
		tags    sql.NullString
		icon    sql.NullString
	)
	if err := row.Scan(&n.ID, &parent, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt, &deleted, &tags, &icon); err != nil {
		return nil, err
	}
	if parent.Valid {
		n.ParentID = &parent.String
	}
	if deleted.Valid {
		n.DeletedAt = &deleted.Int64
	}
	if icon.Valid {
		n.Icon = &icon.String
	}
	n.Tags = parseTags(tags)
	return &n, nil
}

// fetchNote 取一条**未删**笔记；不存在返回 sql.ErrNoRows
func fetchNote(db *sql.DB, id string) (*Note, error) {
	row := db.QueryRow(`SELECT `+noteCols+` FROM notes n WHERE n.id = ? AND n.deleted_at IS NULL`, id)
	return scanNote(row)
}

// noteExists 笔记存在且未被软删
func noteExists(db *sql.DB, id string) bool {
	var x string
	err := db.QueryRow(`SELECT id FROM notes WHERE id = ? AND deleted_at IS NULL`, id).Scan(&x)
	return !errors.Is(err, sql.ErrNoRows)
}

// isAncestor 判断 ancestor 是否为 start 的祖先（沿 parent 链向上能否走到 ancestor）
func isAncestor(db *sql.DB, start, ancestor string) bool {
	cur := start
	for i := 0; i < 10000; i++ {
		if cur == ancestor {
			return true
		}
		var p sql.NullString
		if err := db.QueryRow(`SELECT parent_id FROM notes WHERE id = ?`, cur).Scan(&p); err != nil || !p.Valid {
			return false
		}
		cur = p.String
	}
	return false
}
