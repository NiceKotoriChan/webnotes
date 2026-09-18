package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 错误响应统一从这里出：`{ "error": "<msg>" }`，码与文案见 spec/api.md。
// msg 是给前端看的，一律自己写死，不要把 err.Error() 直接回出去。

func badRequest(c *gin.Context, msg string) { c.JSON(http.StatusBadRequest, gin.H{"error": msg}) }
func notFound(c *gin.Context, msg string)   { c.JSON(http.StatusNotFound, gin.H{"error": msg}) }
func conflict(c *gin.Context, msg string)   { c.JSON(http.StatusConflict, gin.H{"error": msg}) }

// internal 把 err 记进日志，对外只回 internal error —— SQL 原文不出服务端。
func internal(c *gin.Context, err error) {
	log.Printf("%s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}
