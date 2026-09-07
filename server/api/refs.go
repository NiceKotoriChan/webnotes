package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Ref struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	CTime    int64  `json:"ctime"`
}

// GET /api/repos/:repo/notes/:id/refs — 出边引用（本笔记引用了谁）
func (s *Server) listRefs(c *gin.Context) {
	db, ok := s.openRepoDB(c)
	if !ok {
		return
	}
	defer db.Close()

	noteID := c.Param("id")
	if !noteExists(db, noteID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	rows, err := db.Query(
		`SELECT source_id, target_id, ctime FROM refs WHERE source_id = ? ORDER BY ctime DESC`,
		noteID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	refs := []Ref{}
	for rows.Next() {
		var r Ref
		if err := rows.Scan(&r.SourceID, &r.TargetID, &r.CTime); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		refs = append(refs, r)
	}
	c.JSON(http.StatusOK, refs)
}

// GET /api/repos/:repo/notes/:id/backrefs — 入边引用（谁引用了本笔记，即反向链接）
func (s *Server) listBackrefs(c *gin.Context) {
	db, ok := s.openRepoDB(c)
	if !ok {
		return
	}
	defer db.Close()

	noteID := c.Param("id")
	if !noteExists(db, noteID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	rows, err := db.Query(
		`SELECT source_id, target_id, ctime FROM refs WHERE target_id = ? ORDER BY ctime DESC`,
		noteID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	refs := []Ref{}
	for rows.Next() {
		var r Ref
		if err := rows.Scan(&r.SourceID, &r.TargetID, &r.CTime); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		refs = append(refs, r)
	}
	c.JSON(http.StatusOK, refs)
}

// POST /api/repos/:repo/notes/:id/refs  body: { target_id }
// 自环（source == target）由 schema CHECK 拒绝，转 400
func (s *Server) createRef(c *gin.Context) {
	db, ok := s.openRepoDB(c)
	if !ok {
		return
	}
	defer db.Close()

	var body struct {
		TargetID string `json:"target_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sourceID := c.Param("id")
	if sourceID == body.TargetID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "self reference not allowed"})
		return
	}
	now := time.Now().UnixMilli()
	res, err := db.Exec(
		`INSERT INTO refs (source_id, target_id, ctime) VALUES (?, ?, ?)`,
		sourceID, body.TargetID, now,
	)
	if err != nil {
		if isUniqueErr(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "ref exists"})
			return
		}
		// FK 失败：source 或 target 不存在
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insert failed"})
		return
	}
	c.JSON(http.StatusCreated, Ref{SourceID: sourceID, TargetID: body.TargetID, CTime: now})
}

// DELETE /api/repos/:repo/notes/:id/refs/:target_id
func (s *Server) deleteRef(c *gin.Context) {
	db, ok := s.openRepoDB(c)
	if !ok {
		return
	}
	defer db.Close()

	res, err := db.Exec(
		`DELETE FROM refs WHERE source_id=? AND target_id=?`,
		c.Param("id"), c.Param("target_id"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "ref not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
