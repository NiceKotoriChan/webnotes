package api

import (
	"encoding/hex"
	"errors"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"webnotes/server/asset"
)

// GET /api/repos/:repo/assets — 全量 ready 附件，按 date 倒序
func (s *Server) listAssets(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	metas, err := s.assets.List(db)
	if err != nil {
		internal(c, err)
		return
	}
	c.JSON(http.StatusOK, metas)
}

// GET /api/repos/:repo/assets/:sha/meta — 200 AssetMeta / 404
// 用 meta 而不是 HEAD：客户端要的是元数据本身，不是「有没有」
func (s *Server) getAssetMeta(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	m, err := s.assets.Metadata(db, c.Param("sha"))
	if err != nil {
		internal(c, err)
		return
	}
	if m == nil {
		notFound(c, "asset not found")
		return
	}
	c.JSON(http.StatusOK, m)
}

// GET /api/repos/:repo/assets/:sha — 下载；?inline=1 供正文内嵌
func (s *Server) getAsset(c *gin.Context) {
	db, dir, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	m, err := s.assets.Metadata(db, c.Param("sha"))
	if err != nil {
		internal(c, err)
		return
	}
	if m == nil {
		notFound(c, "asset not found")
		return
	}

	c.Header("Content-Type", m.Mime)
	c.Header("Content-Disposition", disposition(m.Name, c.Query("inline") == "1"))
	c.File(s.assets.ReadyPath(dir, m.ID))
}

// POST /api/repos/:repo/assets/:sha — 上传
// Headers: X-Name（必传）、X-Mime、X-Size；Body: 原始字节
func (s *Server) uploadAsset(c *gin.Context) {
	db, dir, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	sha := c.Param("sha")
	if !validSha256(sha) {
		badRequest(c, "invalid sha256")
		return
	}

	name := c.GetHeader("X-Name")
	if name == "" {
		badRequest(c, "X-Name is required")
		return
	}
	contentType := c.GetHeader("X-Mime")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(name))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	size, _ := strconv.ParseInt(c.GetHeader("X-Size"), 10, 64)
	if size < 0 {
		badRequest(c, "invalid size")
		return
	}

	// 已 ready → 秒传：不动文件，回已有元数据
	if m, err := s.assets.Metadata(db, sha); err != nil {
		internal(c, err)
		return
	} else if m != nil {
		c.JSON(http.StatusOK, m)
		return
	}

	// 插入 uploading；主键冲突说明另一个请求正在上传同一个 sha
	inserted, err := s.assets.InsertUploading(db, sha, name, contentType, size, nowMillis())
	if err != nil {
		internal(c, err)
		return
	}
	if !inserted {
		conflict(c, "asset is being uploaded")
		return
	}

	// 流式接收 body → tmp → 校验 sha256 → 原子 rename → ready
	if err := s.assets.Save(db, dir, sha, name, contentType, size, c.Request.Body); err != nil {
		if errors.Is(err, asset.ErrShaMismatch) {
			s.assets.CleanupUploading(db, sha)
			badRequest(c, "sha256 mismatch")
			return
		}
		internal(c, err)
		return
	}

	m, err := s.assets.Metadata(db, sha)
	if err != nil || m == nil {
		internal(c, err)
		return
	}
	c.JSON(http.StatusCreated, m)
}

// POST /api/repos/:repo/assets/:sha/delete
func (s *Server) deleteAsset(c *gin.Context) {
	db, dir, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	if err := s.assets.Remove(db, dir, c.Param("sha")); err != nil {
		if errors.Is(err, asset.ErrNotReady) {
			conflict(c, "asset is not ready")
			return
		}
		internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": c.Param("sha")})
}

func validSha256(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// disposition 生成 Content-Disposition，按 RFC 5987 处理 UTF-8 文件名
func disposition(name string, inline bool) string {
	if inline {
		return "inline"
	}
	name = strings.Map(func(r rune) rune { // 清洗换行/控制字符，防 header injection
		if r == '\n' || r == '\r' || r == 0 {
			return -1
		}
		return r
	}, name)
	return "attachment; filename*=UTF-8''" + percentEncodeUTF8(name)
}

// percentEncodeUTF8 RFC 5987 百分号编码（保留 A-Z a-z 0-9 - _ . ~）
func percentEncodeUTF8(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		switch {
		case (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'):
			b.WriteByte(c)
		case c == '-' || c == '_' || c == '.' || c == '~':
			b.WriteByte(c)
		default:
			b.WriteString("%")
			b.WriteString(strings.ToUpper(strconv.FormatInt(int64(c), 16)))
		}
	}
	return b.String()
}
