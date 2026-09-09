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
	ID       string  `json:"id"`
	ParentID *string `json:"parent_id"`
	Title    string  `json:"title"`
	Content  string  `json:"content"`
	CTime    int64   `json:"ctime"`
	MTime    int64   `json:"mtime"`
}

const noteCols = `n.id, n.parent_id, n.title, n.content, n.ctime, n.mtime`

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
	conds := []string{}
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
	}
	if _, err := db.Exec(
		`INSERT INTO notes (id, parent_id, title, content, ctime, mtime) VALUES (?, ?, ?, ?, ?, ?)`,
		n.ID, n.ParentID, n.Title, n.Content, n.CTime, n.MTime,
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
		`UPDATE notes SET title=?, content=?, mtime=? WHERE id=?`,
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
		`UPDATE notes SET parent_id=?, mtime=? WHERE id=?`,
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

// DELETE /api/repos/:repo/notes/:id  级联删除整棵子树、note_tags、FTS（外键 + 触发器）
func (s *Server) deleteNote(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	res, err := db.Exec(`DELETE FROM notes WHERE id = ?`, c.Param("id"))
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

func scanNotes(rows *sql.Rows) ([]Note, error) {
	defer rows.Close()
	out := []Note{}
	for rows.Next() {
		var n Note
		var parent sql.NullString
		if err := rows.Scan(&n.ID, &parent, &n.Title, &n.Content, &n.CTime, &n.MTime); err != nil {
			return nil, err
		}
		if parent.Valid {
			n.ParentID = &parent.String
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func fetchNote(db *sql.DB, id string) (*Note, error) {
	var n Note
	var parent sql.NullString
	err := db.QueryRow(
		`SELECT `+noteCols+` FROM notes n WHERE n.id = ?`, id,
	).Scan(&n.ID, &parent, &n.Title, &n.Content, &n.CTime, &n.MTime)
	if err != nil {
		return nil, err
	}
	if parent.Valid {
		n.ParentID = &parent.String
	}
	return &n, nil
}

func noteExists(db *sql.DB, id string) bool {
	var x string
	err := db.QueryRow(`SELECT id FROM notes WHERE id = ?`, id).Scan(&x)
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
