package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) listRepos(c *gin.Context) {
	ids, err := s.store.ListRepos()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ids)
}

// POST /api/repos  body: { "id": "work" }
// 创建空 db 文件 + 初始化 schema；已存在返 409
func (s *Server) createRepo(c *gin.Context) {
	var body struct {
		ID string `json:"id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id := body.ID
	if !validRepoID(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repo id"})
		return
	}
	if exists, err := repoExists(s.store.RepoPath(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "repo exists"})
		return
	}
	db, err := s.store.OpenRepo(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	db.Close()
	c.Status(http.StatusNoContent)
}

func (s *Server) deleteRepo(c *gin.Context) {
	repo := c.Param("repo")
	if err := s.store.DeleteRepo(repo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// validRepoID 禁止路径穿越字符与扩展名
func validRepoID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	if strings.ContainsAny(id, `/\..`) {
		return false
	}
	return true
}

func repoExists(path string) (bool, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	// 三个后缀任一存在即视为 repo 已存在
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if _, err := os.Stat(abs + suffix); err == nil {
			return true, nil
		}
	}
	return false, nil
}
