// Package api 提供 HTTP 路由与接口实现。
// 接口契约（19 条路由、POST 的三种形态、错误码）见 spec/api.md。
package api

import (
	"database/sql"
	"errors"

	"github.com/gin-gonic/gin"

	"webnotes/server/asset"
	"webnotes/server/store"
)

type Server struct {
	store  *store.Store
	assets *asset.Store
}

func NewRouter(s *store.Store, a *asset.Store) *gin.Engine {
	r := gin.Default()
	srv := &Server{store: s, assets: a}

	g := r.Group("/api")
	{
		// Repos
		g.GET("/repos", srv.listRepos)
		g.POST("/repos", srv.createRepo)
		g.POST("/repos/:repo", srv.renameRepo)
		g.POST("/repos/:repo/delete", srv.deleteRepo)

		// Notes
		g.GET("/repos/:repo/notes", srv.listNotes)
		g.POST("/repos/:repo/notes", srv.createNote)
		g.GET("/repos/:repo/notes/:id", srv.getNote)
		g.POST("/repos/:repo/notes/:id", srv.updateNote)
		g.POST("/repos/:repo/notes/:id/delete", srv.deleteNote)
		g.POST("/repos/:repo/notes/:id/restore", srv.restoreNote)
		g.GET("/repos/:repo/trash", srv.listTrash)

		// Tags（派生，没有标签表）
		g.GET("/repos/:repo/tags", srv.listTags)
		g.POST("/repos/:repo/tags/rename", srv.renameTag)
		g.POST("/repos/:repo/tags/delete", srv.deleteTag)

		// Assets（仓库内私有）
		g.GET("/repos/:repo/assets", srv.listAssets)
		g.GET("/repos/:repo/assets/:sha", srv.getAsset)
		g.GET("/repos/:repo/assets/:sha/meta", srv.getAssetMeta)
		g.POST("/repos/:repo/assets/:sha", srv.uploadAsset)
		g.POST("/repos/:repo/assets/:sha/delete", srv.deleteAsset)
	}
	return r
}

// openRepo 打开路径参数指定的仓库，返回 db 与目录；失败已写响应
func (s *Server) openRepo(c *gin.Context) (*sql.DB, string, bool) {
	repo := c.Param("repo")
	db, err := s.store.OpenRepo(repo)
	if errors.Is(err, store.ErrNotFound) {
		notFound(c, "repo not found")
		return nil, "", false
	}
	if err != nil {
		internal(c, err)
		return nil, "", false
	}
	return db, s.store.RepoDir(repo), true
}
