// Package api 提供 HTTP 路由
package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"webnotes/server/asset"
	"webnotes/server/icons"
	"webnotes/server/store"
)

type Server struct {
	store  *store.Store
	assets *asset.Store
	rules  *icons.Rules
}

func NewRouter(s *store.Store, a *asset.Store, ru *icons.Rules) *gin.Engine {
	r := gin.Default()
	srv := &Server{store: s, assets: a, rules: ru}

	g := r.Group("/api")
	{
		// Repos
		g.GET("/repos", srv.listRepos)
		g.POST("/repos", srv.createRepo)
		g.PATCH("/repos/:repo", srv.renameRepo)
		g.DELETE("/repos/:repo", srv.deleteRepo)

		// Notes（树形）
		g.GET("/repos/:repo/notes", srv.listNotes)
		g.POST("/repos/:repo/notes", srv.createNote)
		g.GET("/repos/:repo/notes/:id", srv.getNote)
		g.PUT("/repos/:repo/notes/:id", srv.updateNote)
		g.PATCH("/repos/:repo/notes/:id", srv.moveNote)
		g.PATCH("/repos/:repo/notes/:id/icon", srv.setNoteIcon)
		g.DELETE("/repos/:repo/notes/:id", srv.deleteNote)
		g.POST("/repos/:repo/notes/:id/restore", srv.restoreNote)
		g.GET("/repos/:repo/trash", srv.listTrash)

		// Tags
		g.GET("/repos/:repo/tags", srv.listTags)
		g.POST("/repos/:repo/tags", srv.createTag)
		g.PUT("/repos/:repo/tags/:id", srv.renameTag)
		g.DELETE("/repos/:repo/tags/:id", srv.deleteTag)

		// Note-Tag 关联
		g.GET("/repos/:repo/notes/:id/tags", srv.listNoteTags)
		g.POST("/repos/:repo/notes/:id/tags", srv.addNoteTag)
		g.DELETE("/repos/:repo/notes/:id/tags/:tag_id", srv.removeNoteTag)

		// Assets（仓库内私有）
		g.GET("/repos/:repo/assets", srv.listAssets)
		g.HEAD("/repos/:repo/assets/:sha", srv.headAsset)
		g.POST("/repos/:repo/assets/:sha", srv.uploadAsset)
		g.GET("/repos/:repo/assets/:sha", srv.getAsset)
		g.DELETE("/repos/:repo/assets/:sha", srv.deleteAsset)

		// 图标规则（全局，不属任何仓库；读走 /icons/mdi_rules_custom.json）
		g.POST("/icons/rules", srv.saveRules)
		g.POST("/icons/rules/reset", srv.resetRules)
	}
	return r
}

// openRepo 打开路径参数指定的仓库，返回 db 与目录；失败已写响应
func (s *Server) openRepo(c *gin.Context) (*sql.DB, string, bool) {
	repo := c.Param("repo")
	db, err := s.store.OpenRepo(repo)
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "repo not found"})
		return nil, "", false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, "", false
	}
	return db, s.store.RepoDir(repo), true
}
