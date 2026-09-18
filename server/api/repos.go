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
		badRequest(c, "invalid json")
		return
	}
	info, err := s.store.CreateRepo(body.Name)
	if err != nil {
		internal(c, err)
		return
	}
	c.JSON(http.StatusCreated, info)
}

// POST /api/repos/:repo  body: { name }  改名（只动索引，目录与链接不变）
func (s *Server) renameRepo(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "invalid json")
		return
	}
	info, err := s.store.RenameRepo(c.Param("repo"), body.Name)
	if errors.Is(err, store.ErrNotFound) {
		notFound(c, "repo not found")
		return
	}
	if err != nil {
		internal(c, err)
		return
	}
	c.JSON(http.StatusOK, info)
}

// POST /api/repos/:repo/delete  删除整个仓库目录
func (s *Server) deleteRepo(c *gin.Context) {
	id := c.Param("repo")
	if err := s.store.DeleteRepo(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			notFound(c, "repo not found")
			return
		}
		internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}
