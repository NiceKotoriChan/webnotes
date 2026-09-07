package api

import (
	"encoding/hex"
	"errors"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"webnotes/server/asset"
)

// HEAD /api/assets/:sha — 检查是否 ready
func (s *Server) headAsset(c *gin.Context) {
	sha := c.Param("sha")
	st, err := s.assets.Status(sha)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if st != "ready" {
		c.Status(http.StatusNotFound)
		return
	}
	c.Status(http.StatusOK)
}

// POST /api/assets/:sha — 上传
// Headers: X-Name, X-Mime, X-Size；Body: 原始字节
func (s *Server) uploadAsset(c *gin.Context) {
	sha := c.Param("sha")
	if !validSha256(sha) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sha256"})
		return
	}

	name := c.GetHeader("X-Name")
	contentType := c.GetHeader("X-Mime")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(name))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	size, _ := strconv.ParseInt(c.GetHeader("X-Size"), 10, 64)
	if size < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid size"})
		return
	}

	// 已 ready → 秒传
	if st, _ := s.assets.Status(sha); st == "ready" {
		c.Status(http.StatusNoContent)
		return
	}

	// INSERT uploading；冲突说明已 ready 或正在上传
	ok, err := s.assets.InsertUploading(sha, name, contentType, size, time.Now().UnixMilli())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusConflict, gin.H{"error": "asset exists or upload in progress"})
		return
	}

	// 流式接收 body → tmp → 校验 → rename → ready
	if err := s.assets.Save(sha, name, contentType, size, c.Request.Body); err != nil {
		if errors.Is(err, asset.ErrShaMismatch) {
			s.assets.CleanupUploading(sha)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	m, err := s.assets.Metadata(sha)
	if err != nil || m == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fetch metadata"})
		return
	}
	c.JSON(http.StatusCreated, m)
}

// GET /api/assets/:sha — 下载
// ?inline=1 → Content-Disposition: inline
func (s *Server) getAsset(c *gin.Context) {
	sha := c.Param("sha")
	m, err := s.assets.Metadata(sha)
	if err != nil || m == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset not ready"})
		return
	}

	c.Header("Content-Type", m.Mime)
	c.Header("Content-Disposition", disposition(m.Name, c.Query("inline") != ""))
	c.File(s.assets.ReadyPath(sha))
}

// DELETE /api/assets/:sha
func (s *Server) deleteAsset(c *gin.Context) {
	sha := c.Param("sha")
	if err := s.assets.Remove(sha); err != nil {
		if errors.Is(err, asset.ErrNotReady) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// validSha256 检查是否 64 hex
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
	// 清洗换行/控制字符防 header injection
	name = strings.Map(func(r rune) rune {
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
