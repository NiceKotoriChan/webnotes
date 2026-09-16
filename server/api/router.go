// Package api 提供 HTTP 路由
package api

import (
	"database/sql"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"webnotes/server/asset"
	"webnotes/server/icons"
	"webnotes/server/store"
)

type Server struct {
	store  *store.Store
	assets *asset.Store
}

// NewRouter 组装路由。约定：一律 POST，动作名在路径最后一段，参数走 JSON body；
// 只有静态资源（附件、图标集、前端 web/）走 GET。iconPath + ic 是后端托管的图标集。
func NewRouter(s *store.Store, a *asset.Store, ic *icons.Set, iconPath string) *gin.Engine {
	r := gin.Default()
	srv := &Server{store: s, assets: a}

	// 静态资源
	r.GET(iconPath, gin.WrapH(ic))
	r.HEAD(iconPath, gin.WrapH(ic))

	g := r.Group("/api")
	{
		// Repos
		g.POST("/repos/list", srv.listRepos)
		g.POST("/repos/create", srv.createRepo)
		g.POST("/repos/:repo/rename", srv.renameRepo)
		g.POST("/repos/:repo/delete", srv.deleteRepo)

		// Notes（树形）
		g.POST("/repos/:repo/notes/list", srv.listNotes)
		g.POST("/repos/:repo/notes/create", srv.createNote)
		g.POST("/repos/:repo/notes/:id/get", srv.getNote)
		g.POST("/repos/:repo/notes/:id/save", srv.updateNote)
		g.POST("/repos/:repo/notes/:id/move", srv.moveNote)
		g.POST("/repos/:repo/notes/:id/icon", srv.setNoteIcon)
		g.POST("/repos/:repo/notes/:id/delete", srv.deleteNote)
		g.POST("/repos/:repo/notes/:id/restore", srv.restoreNote)
		g.POST("/repos/:repo/trash/list", srv.listTrash)

		// Tags
		g.POST("/repos/:repo/tags/list", srv.listTags)
		g.POST("/repos/:repo/tags/create", srv.createTag)
		g.POST("/repos/:repo/tags/:id/rename", srv.renameTag)
		g.POST("/repos/:repo/tags/:id/delete", srv.deleteTag)

		// Note-Tag 关联
		g.POST("/repos/:repo/notes/:id/tags/list", srv.listNoteTags)
		g.POST("/repos/:repo/notes/:id/tags/add", srv.addNoteTag)
		g.POST("/repos/:repo/notes/:id/tags/remove", srv.removeNoteTag)

		// Assets（仓库内私有）
		g.POST("/repos/:repo/assets/list", srv.listAssets)
		g.POST("/repos/:repo/assets/:sha/exists", srv.existsAsset)
		g.POST("/repos/:repo/assets/:sha/upload", srv.uploadAsset)
		g.POST("/repos/:repo/assets/:sha/delete", srv.deleteAsset)
		g.GET("/repos/:repo/assets/:sha", srv.getAsset) // 静态资源：下载/内联
	}
	return r
}

// bindJSON 解析 JSON body；空 body 视为「没带参数」而不是错误
func bindJSON(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
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
