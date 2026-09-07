package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /api/repos/:repo/notes/:id/tags
func (s *Server) listNoteTags(c *gin.Context) {
	db, ok := s.openRepoDB(c)
	if !ok {
		return
	}
	defer db.Close()

	noteID := c.Param("id")
	// 笔记不存在 → 404
	if !noteExists(db, noteID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	rows, err := db.Query(
		`SELECT t.id, t.name FROM tags t
		 JOIN note_tags nt ON nt.tag_id = t.id
		 WHERE nt.note_id = ?
		 ORDER BY t.name`, noteID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	tags := []Tag{}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		tags = append(tags, t)
	}
	c.JSON(http.StatusOK, tags)
}

// POST /api/repos/:repo/notes/:id/tags  body: { tag_id }
func (s *Server) addNoteTag(c *gin.Context) {
	db, ok := s.openRepoDB(c)
	if !ok {
		return
	}
	defer db.Close()

	var body struct {
		TagID string `json:"tag_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	noteID := c.Param("id")
	// note_tags (note_id, tag_id) PRIMARY KEY → 重复关联返 409
	res, err := db.Exec(
		`INSERT INTO note_tags (note_id, tag_id) VALUES (?, ?)`, noteID, body.TagID,
	)
	if err != nil {
		if isUniqueErr(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "already linked"})
			return
		}
		// FK 失败：note 或 tag 不存在
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insert failed"})
		return
	}
	c.Status(http.StatusNoContent)
}

// DELETE /api/repos/:repo/notes/:id/tags/:tag_id
func (s *Server) removeNoteTag(c *gin.Context) {
	db, ok := s.openRepoDB(c)
	if !ok {
		return
	}
	defer db.Close()

	res, err := db.Exec(
		`DELETE FROM note_tags WHERE note_id=? AND tag_id=?`,
		c.Param("id"), c.Param("tag_id"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func noteExists(db *sql.DB, id string) bool {
	var x string
	err := db.QueryRow(`SELECT id FROM notes WHERE id = ?`, id).Scan(&x)
	return !errors.Is(err, sql.ErrNoRows)
}
