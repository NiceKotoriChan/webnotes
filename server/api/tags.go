package api

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// 标签不是实体：没有标签表、没有 id，「新建」就是给笔记写上这个字符串。
// 下面三条都在 notes.tags 这个 JSON 列上做，见 spec/model.md。

// GET /api/repos/:repo/tags → string[]（扫未删笔记的 tags 去重，按名升序）
func (s *Server) listTags(c *gin.Context) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT DISTINCT json_each.value AS name
		FROM notes, json_each(notes.tags)
		WHERE notes.deleted_at IS NULL AND json_each.value <> ''
		ORDER BY name`)
	if err != nil {
		internal(c, err)
		return
	}
	defer rows.Close()

	tags := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			internal(c, err)
			return
		}
		tags = append(tags, name)
	}
	if err := rows.Err(); err != nil {
		internal(c, err)
		return
	}
	c.JSON(http.StatusOK, tags)
}

// POST /api/repos/:repo/tags/rename  body: { from, to } → { count }
// 全表改写；to 已存在就是两个标签合并
func (s *Server) renameTag(c *gin.Context) {
	var body struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "invalid json")
		return
	}
	from, to := strings.TrimSpace(body.From), strings.TrimSpace(body.To)
	if from == "" || to == "" {
		badRequest(c, "from and to are required")
		return
	}
	count, ok := s.rewriteTags(c, func(tags []string) []string {
		if !containsTag(tags, from) {
			return nil
		}
		out := make([]string, 0, len(tags))
		for _, t := range tags {
			if t == from {
				out = append(out, to)
				continue
			}
			out = append(out, t)
		}
		return normalizeTags(out) // to 可能已在别的笔记里出现 → 去重
	})
	if !ok {
		return
	}
	if count == 0 {
		notFound(c, "tag not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

// POST /api/repos/:repo/tags/delete  body: { name } → { count }
func (s *Server) deleteTag(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "invalid json")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		badRequest(c, "name is required")
		return
	}
	count, ok := s.rewriteTags(c, func(tags []string) []string {
		if !containsTag(tags, name) {
			return nil
		}
		out := make([]string, 0, len(tags))
		for _, t := range tags {
			if t != name {
				out = append(out, t)
			}
		}
		return normalizeTags(out)
	})
	if !ok {
		return
	}
	if count == 0 {
		notFound(c, "tag not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

// rewriteTags 遍历全部未删笔记，把 rewrite 的返回值写回去（返回 nil = 这篇没这个标签，不动）。
// 命中的笔记连 updated_at 一起刷新。返回受影响的笔记数；失败已写响应。
func (s *Server) rewriteTags(c *gin.Context, rewrite func(tags []string) []string) (int, bool) {
	db, _, ok := s.openRepo(c)
	if !ok {
		return 0, false
	}
	defer db.Close()

	type row struct {
		id   string
		tags []string
	}
	rows, err := db.Query(`SELECT id, tags FROM notes WHERE deleted_at IS NULL`)
	if err != nil {
		internal(c, err)
		return 0, false
	}
	var all []row
	for rows.Next() {
		var id string
		var raw sql.NullString
		if err := rows.Scan(&id, &raw); err != nil {
			rows.Close()
			internal(c, err)
			return 0, false
		}
		all = append(all, row{id: id, tags: parseTags(raw)})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		internal(c, err)
		return 0, false
	}
	rows.Close()

	tx, err := db.Begin()
	if err != nil {
		internal(c, err)
		return 0, false
	}
	defer tx.Rollback()

	now := nowMillis()
	count := 0
	for _, r := range all {
		next := rewrite(r.tags)
		if next == nil {
			continue
		}
		if _, err := tx.Exec(
			`UPDATE notes SET tags = ?, updated_at = ? WHERE id = ?`, encodeTags(next), now, r.id,
		); err != nil {
			internal(c, err)
			return 0, false
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		internal(c, err)
		return 0, false
	}
	return count, true
}

func containsTag(tags []string, name string) bool {
	for _, t := range tags {
		if t == name {
			return true
		}
	}
	return false
}
