package api

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"webnotes/server/icons"
)

// saveRules 覆盖自定义图标规则：body 就是一份完整的规则文件（前端改完整体回传）。
// 只有这里能写，改的是 mdi_rules_custom.json；默认规则永远只读。
func (s *Server) saveRules(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, icons.MaxRulesBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}
	doc, err := s.rules.SaveCustom(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, doc)
}

// resetRules 回退到默认设置：用默认规则覆盖自定义规则。
func (s *Server) resetRules(c *gin.Context) {
	doc, err := s.rules.ResetCustom()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, doc)
}
