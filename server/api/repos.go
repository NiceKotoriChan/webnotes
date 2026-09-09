package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"webnotes/server/store"
)

// GET /api/repos
func (s *Server) listRepos(c *gin.Context) {
	c.JSON(http.StatusOK, s.store.ListRepos())
}

// POST /api/repos  body: { name }  → 服务端生成 UUID
func (s *Server) createRepo(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	info, err := s.store.CreateRepo(body.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, info)
}

// PATCH /api/repos/:repo  body: { name }  改名（不动目录/链接）
func (s *Server) renameRepo(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	info, err := s.store.RenameRepo(c.Param("repo"), body.Name)
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "repo not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// DELETE /api/repos/:repo  删除整个仓库目录
func (s *Server) deleteRepo(c *gin.Context) {
	if err := s.store.DeleteRepo(c.Param("repo")); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "repo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
