package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Note struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	CTime   int64  `json:"ctime"`
	MTime   int64  `json:"mtime"`
}

// openRepoDB 打开 repo db，404 时返错（由 caller 处理）
func (s *Server) openRepoDB(c *gin.Context) (*sql.DB, bool) {
	repo := c.Param("repo")
	db, err := s.store.OpenRepo(repo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, false
	}
	return db, true
}

// GET /api/repos/:repo/notes?q=&tag_id=&limit=&offset=
func (s *Server) listNotes(c *gin.Context) {
	db, ok := s.openRepoDB(c)
	if !ok {
		return
	}
	defer db.Close()

	q := c.Query("q")
	tagID := c.Query("tag_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	// trigram tokenizer 需要 ≥3 个 Unicode 字符才能匹配
	// 短查询降级到 LIKE，避免 2 字搜索返空
	useFTS := q != "" && utf8.RuneCountInString(q) >= 3
	var (
		rows *sql.Rows
		err  error
	)
	switch {
	case useFTS && tagID != "":
		rows, err = db.Query(`
			SELECT n.id, n.title, n.content, n.ctime, n.mtime
			FROM notes_fts f
			JOIN notes n ON n.rowid = f.rowid
			JOIN note_tags nt ON nt.note_id = n.id
			WHERE notes_fts MATCH ? AND nt.tag_id = ?
			ORDER BY f.rank
			LIMIT ? OFFSET ?`, q, tagID, limit, offset)
	case useFTS:
		rows, err = db.Query(`
			SELECT n.id, n.title, n.content, n.ctime, n.mtime
			FROM notes_fts f
			JOIN notes n ON n.rowid = f.rowid
			WHERE notes_fts MATCH ?
			ORDER BY f.rank
			LIMIT ? OFFSET ?`, q, limit, offset)
	case q != "" && tagID != "":
		like := "%" + q + "%"
		rows, err = db.Query(`
			SELECT n.id, n.title, n.content, n.ctime, n.mtime
			FROM notes n
			JOIN note_tags nt ON nt.note_id = n.id
			WHERE (n.title LIKE ? OR n.content LIKE ?) AND nt.tag_id = ?
			ORDER BY n.mtime DESC
			LIMIT ? OFFSET ?`, like, like, tagID, limit, offset)
	case q != "":
		like := "%" + q + "%"
		rows, err = db.Query(`
			SELECT id, title, content, ctime, mtime
			FROM notes
			WHERE title LIKE ? OR content LIKE ?
			ORDER BY mtime DESC
			LIMIT ? OFFSET ?`, like, like, limit, offset)
	case tagID != "":
		rows, err = db.Query(`
			SELECT n.id, n.title, n.content, n.ctime, n.mtime
			FROM notes n
			JOIN note_tags nt ON nt.note_id = n.id
			WHERE nt.tag_id = ?
			ORDER BY n.mtime DESC
			LIMIT ? OFFSET ?`, tagID, limit, offset)
	default:
		rows, err = db.Query(`
			SELECT id, title, content, ctime, mtime
			FROM notes
			ORDER BY mtime DESC
			LIMIT ? OFFSET ?`, limit, offset)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.CTime, &n.MTime); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		notes = append(notes, n)
	}
	c.JSON(http.StatusOK, notes)
}

// POST /api/repos/:repo/notes  body: { title, content }
func (s *Server) createNote(c *gin.Context) {
	db, ok := s.openRepoDB(c)
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
	now := time.Now().UnixMilli()
	n := Note{
		ID:      uuid.NewString(),
		Title:   body.Title,
		Content: body.Content,
		CTime:   now,
		MTime:   now,
	}
	if _, err := db.Exec(
		`INSERT INTO notes (id, title, content, ctime, mtime) VALUES (?, ?, ?, ?, ?)`,
		n.ID, n.Title, n.Content, n.CTime, n.MTime,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, n)
}

// GET /api/repos/:repo/notes/:id
func (s *Server) getNote(c *gin.Context) {
	db, ok := s.openRepoDB(c)
	if !ok {
		return
	}
	defer db.Close()

	var n Note
	err := db.QueryRow(
		`SELECT id, title, content, ctime, mtime FROM notes WHERE id = ?`, c.Param("id"),
	).Scan(&n.ID, &n.Title, &n.Content, &n.CTime, &n.MTime)
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

// PUT /api/repos/:repo/notes/:id  body: { title, content }
func (s *Server) updateNote(c *gin.Context) {
	db, ok := s.openRepoDB(c)
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
	now := time.Now().UnixMilli()
	res, err := db.Exec(
		`UPDATE notes SET title=?, content=?, mtime=? WHERE id=?`,
		body.Title, body.Content, now, c.Param("id"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	c.JSON(http.StatusOK, Note{
		ID:      c.Param("id"),
		Title:   body.Title,
		Content: body.Content,
		MTime:   now,
	})
}

// DELETE /api/repos/:repo/notes/:id
// 级联 note_tags、refs（source/target 两侧）、FTS 由触发器清理
func (s *Server) deleteNote(c *gin.Context) {
	db, ok := s.openRepoDB(c)
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
