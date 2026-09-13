package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Note struct {
	ID        string  `json:"id"`
	ParentID  *string `json:"parent_id"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	CTime     int64   `json:"ctime"`
	MTime     int64   `json:"mtime"`
	DeletedAt *int64  `json:"deleted_at,omitempty"`
	Icon      *string `json:"icon,omitempty"`
}

const noteCols = `n.id, n.parent_id, n.title, n.content, n.ctime, n.mtime, n.deleted_at, n.icon`

// GET /api/repos/:repo/notes?q=&tag_id=&parent_id=&limit=&offset=
func (s *Server) listNotes(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	q := c.Query("q")
	tagID := c.Query("tag_id")
	_, hasParent := c.GetQuery("parent_id")
	parentID := c.Query("parent_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	// trigram 需 ≥3 字符；短查询降级 LIKE
	useFTS := q != "" && utf8.RuneCountInString(q) >= 3

	joins := strings.Builder{}
	conds := []string{"n.deleted_at IS NULL"}
	args := []any{}
	if useFTS {
		joins.WriteString(`JOIN notes_fts f ON f.rowid = n.rowid `)
		conds = append(conds, "notes_fts MATCH ?")
		args = append(args, q)
	} else if q != "" {
		conds = append(conds, "(n.title LIKE ? OR n.content LIKE ?)")
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	if tagID != "" {
		joins.WriteString(`JOIN note_tags nt ON nt.note_id = n.id `)
		conds = append(conds, "nt.tag_id = ?")
		args = append(args, tagID)
	}
	if hasParent {
		if parentID == "" {
			conds = append(conds, "n.parent_id IS NULL") // 根节点
		} else {
			conds = append(conds, "n.parent_id = ?")
			args = append(args, parentID)
		}
	}

	order := "n.mtime DESC"
	if useFTS {
		order = "f.rank"
	}
	query := `SELECT ` + noteCols + ` FROM notes n ` + joins.String()
	if len(conds) > 0 {
		query += "WHERE " + strings.Join(conds, " AND ") + " "
	}
	query += "ORDER BY " + order + " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	notes, err := scanNotes(rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, notes)
}

// POST /api/repos/:repo/notes  body: { title, content, parent_id? }
func (s *Server) createNote(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	var body struct {
		Title    string  `json:"title"`
		Content  string  `json:"content"`
		ParentID *string `json:"parent_id"`
		Icon     *string `json:"icon"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.ParentID != nil && !noteExists(db, *body.ParentID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parent note not found"})
		return
	}

	now := time.Now().UnixMilli()
	n := Note{
		ID:       uuid.NewString(),
		ParentID: body.ParentID,
		Title:    body.Title,
		Content:  body.Content,
		CTime:    now,
		MTime:    now,
		Icon:     body.Icon,
	}
	if _, err := db.Exec(
		`INSERT INTO notes (id, parent_id, title, content, ctime, mtime, icon) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.ParentID, n.Title, n.Content, n.CTime, n.MTime, n.Icon,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, n)
}

// PUT /api/repos/:repo/notes/:id  body: { title, content }（mtime=now，ctime/parent 不变）
func (s *Server) updateNote(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := db.Exec(
		`UPDATE notes SET title=?, content=?, mtime=? WHERE id=? AND deleted_at IS NULL`,
		body.Title, body.Content, time.Now().UnixMilli(), c.Param("id"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	n, err := fetchNote(db, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, n)
}

// PATCH /api/repos/:repo/notes/:id  body: { parent_id }  移动节点（parent_id=null 移到根）
func (s *Server) moveNote(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	var body struct {
		ParentID *string `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	noteID := c.Param("id")
	if !noteExists(db, noteID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	if body.ParentID != nil {
		pid := *body.ParentID
		if pid == noteID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot move note under itself"})
			return
		}
		if !noteExists(db, pid) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parent note not found"})
			return
		}
		if isAncestor(db, pid, noteID) { // noteID 是 pid 的祖先 → 移过去成环
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot move note under its own descendant"})
			return
		}
	}

	res, err := db.Exec(
		`UPDATE notes SET parent_id=?, mtime=? WHERE id=? AND deleted_at IS NULL`,
		body.ParentID, time.Now().UnixMilli(), noteID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	n, err := fetchNote(db, noteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, n)
}

// DELETE /api/repos/:repo/notes/:id  软删除整棵子树（默认）；?permanent=1 彻底删除
func (s *Server) deleteNote(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	noteID := c.Param("id")
	permanent := c.Query("permanent") == "1"

	var (
		res sql.Result
		err error
	)
	if permanent {
		// 彻底删除：硬删整棵子树，note_tags 与 FTS 由外键 + 触发器级联
		res, err = db.Exec(`DELETE FROM notes WHERE id = ?`, noteID)
	} else {
		// 软删除：递归标记整棵子树（保留 note_tags 便于还原）
		res, err = db.Exec(`
			WITH RECURSIVE subtree(id) AS (
				SELECT id FROM notes WHERE id = ? AND deleted_at IS NULL
				UNION ALL
				SELECT n.id FROM notes n JOIN subtree s ON n.parent_id = s.id
			)
			UPDATE notes SET deleted_at = ? WHERE id IN (SELECT id FROM subtree)`,
			noteID, time.Now().UnixMilli())
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// POST /api/repos/:repo/notes/:id/restore  还原回收站里的笔记（连同其子树）
func (s *Server) restoreNote(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	noteID := c.Param("id")
	var parent sql.NullString
	err := db.QueryRow(
		`SELECT parent_id FROM notes WHERE id = ? AND deleted_at IS NOT NULL`, noteID,
	).Scan(&parent)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found in trash"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 递归还原整棵子树
	if _, err := db.Exec(`
		WITH RECURSIVE subtree(id) AS (
			SELECT id FROM notes WHERE id = ?
			UNION ALL
			SELECT n.id FROM notes n JOIN subtree s ON n.parent_id = s.id
		)
		UPDATE notes SET deleted_at = NULL WHERE id IN (SELECT id FROM subtree)`, noteID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 若父节点仍被删除，移到根
	if parent.Valid {
		var parentDeleted int
		err := db.QueryRow(
			`SELECT deleted_at IS NOT NULL FROM notes WHERE id = ?`, parent.String,
		).Scan(&parentDeleted)
		if err != nil || parentDeleted != 0 {
			if _, err := db.Exec(`UPDATE notes SET parent_id = NULL WHERE id = ?`, noteID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}

	c.Status(http.StatusNoContent)
}

// PATCH /api/repos/:repo/notes/:id/icon  body: { icon }  设置自定义图标；icon=null 恢复自动匹配
func (s *Server) setNoteIcon(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	var body struct {
		Icon *string `json:"icon"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := db.Exec(
		`UPDATE notes SET icon=? WHERE id=? AND deleted_at IS NULL`,
		body.Icon, c.Param("id"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// GET /api/repos/:repo/trash  回收站：列出被删除的顶层笔记（父节点未删除的）
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	notes, err := scanNotes(rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, notes)
}

func scanNotes(rows *sql.Rows) ([]Note, error) {
	defer rows.Close()
	out := []Note{}
	for rows.Next() {
		var n Note
		var parent sql.NullString
		var deleted sql.NullInt64
		var icon sql.NullString
		if err := rows.Scan(&n.ID, &parent, &n.Title, &n.Content, &n.CTime, &n.MTime, &deleted, &icon); err != nil {
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
		out = append(out, n)
	}
	return out, rows.Err()
}

func fetchNote(db *sql.DB, id string) (*Note, error) {
	var n Note
	var parent sql.NullString
	var deleted sql.NullInt64
	var icon sql.NullString
	err := db.QueryRow(
		`SELECT `+noteCols+` FROM notes n WHERE n.id = ? AND n.deleted_at IS NULL`, id,
	).Scan(&n.ID, &parent, &n.Title, &n.Content, &n.CTime, &n.MTime, &deleted, &icon)
	if err != nil {
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
	return &n, nil
}

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
