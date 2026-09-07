// Package api 提供 HTTP 路由
package api

import (
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

	api := r.Group("/api")
	{
		// Repos
		api.GET("/repos", srv.listRepos)
		api.POST("/repos", srv.createRepo)
		api.DELETE("/repos/:repo", srv.deleteRepo)

		// Notes
		api.GET("/repos/:repo/notes", srv.listNotes)
		api.POST("/repos/:repo/notes", srv.createNote)
		api.GET("/repos/:repo/notes/:id", srv.getNote)
		api.PUT("/repos/:repo/notes/:id", srv.updateNote)
		api.DELETE("/repos/:repo/notes/:id", srv.deleteNote)

		// Tags
		api.GET("/repos/:repo/tags", srv.listTags)
		api.POST("/repos/:repo/tags", srv.createTag)
		api.PUT("/repos/:repo/tags/:id", srv.renameTag)
		api.DELETE("/repos/:repo/tags/:id", srv.deleteTag)

		// Note-Tag
		api.GET("/repos/:repo/notes/:id/tags", srv.listNoteTags)
		api.POST("/repos/:repo/notes/:id/tags", srv.addNoteTag)
		api.DELETE("/repos/:repo/notes/:id/tags/:tag_id", srv.removeNoteTag)

		// Refs
		api.GET("/repos/:repo/notes/:id/refs", srv.listRefs)
		api.GET("/repos/:repo/notes/:id/backrefs", srv.listBackrefs)
		api.POST("/repos/:repo/notes/:id/refs", srv.createRef)
		api.DELETE("/repos/:repo/notes/:id/refs/:target_id", srv.deleteRef)

		// Assets
		api.HEAD("/assets/:sha", srv.headAsset)
		api.POST("/assets/:sha", srv.uploadAsset)
		api.GET("/assets/:sha", srv.getAsset)
		api.DELETE("/assets/:sha", srv.deleteAsset)
	}
	return r
}
