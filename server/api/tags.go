package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GET /api/repos/:repo/tags
func (s *Server) listTags(c *gin.Context) {
	db, ok := s.openRepoDB(c)
	if !ok {
		return
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, name FROM tags ORDER BY name`)
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

// POST /api/repos/:repo/tags  body: { name }
// UNIQUE 冲突返 409
func (s *Server) createTag(c *gin.Context) {
	db, ok := s.openRepoDB(c)
	if !ok {
		return
	}
	defer db.Close()

	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t := Tag{ID: uuid.NewString(), Name: body.Name}
	if _, err := db.Exec(
		`INSERT INTO tags (id, name) VALUES (?, ?)`, t.ID, t.Name,
	); err != nil {
		// SQLite UNIQUE 冲突错误串里包含 "UNIQUE"
		if isUniqueErr(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "tag name exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, t)
}

// PUT /api/repos/:repo/tags/:id  body: { name }
func (s *Server) renameTag(c *gin.Context) {
	db, ok := s.openRepoDB(c)
	if !ok {
		return
	}
	defer db.Close()

	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := db.Exec(`UPDATE tags SET name=? WHERE id=?`, body.Name, c.Param("id"))
	if err != nil {
		if isUniqueErr(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "tag name exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}
	c.JSON(http.StatusOK, Tag{ID: c.Param("id"), Name: body.Name})
}

// DELETE /api/repos/:repo/tags/:id
// 级联 note_tags（ON DELETE CASCADE）
func (s *Server) deleteTag(c *gin.Context) {
	db, ok := s.openRepoDB(c)
	if !ok {
		return
	}
	defer db.Close()

	res, err := db.Exec(`DELETE FROM tags WHERE id=?`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// isUniqueErr 判断是否 UNIQUE 约束冲突（modernc 错误信息含 "UNIQUE constraint failed"）
func isUniqueErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE")
}
