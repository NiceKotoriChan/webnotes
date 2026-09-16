// Package icons 后端托管图标集：本地文件是唯一真相源，远端更新遵循「拿到更好的才替换」。
// 启动时同步保证本地有一份（仅在缺失时才下载），之后按固定间隔在后台检查更新。
package icons

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// maxDownload 单次下载上限，防止坏源把磁盘写满
	maxDownload = 64 << 20
	// requestTimeout 单次下载/检查的超时
	requestTimeout = 60 * time.Second
)

// Meta 本地副本的状态，随文件一起落盘为 <file>.meta.json，便于事后排查「现在用的是哪一版」。
type Meta struct {
	Source    string `json:"source"`
	ETag      string `json:"etag"`
	LastMod   string `json:"last_modified"`
	SHA256    string `json:"sha256"`
	Size      int64  `json:"size"`
	Icons     int    `json:"icons"`
	UpdatedAt int64  `json:"updated_at"` // Unix ms
}

// Set 管理一个图标集文件的本地副本与远端更新。
type Set struct {
	dir      string
	file     string
	sources  []string
	minCount int
	http     *http.Client

	mu   sync.RWMutex
	data []byte // 内存里的副本，直接给 HTTP 请求读
	meta Meta
}

// New 准备好目录并读入本地副本（读不到就留空，交给 Ensure 下载）。
func New(dir, file string, sources []string, minCount int) (*Set, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Set{
		dir:      dir,
		file:     file,
		sources:  sources,
		minCount: minCount,
		http:     &http.Client{Timeout: requestTimeout},
	}
	s.load()
	return s, nil
}

func (s *Set) localPath() string { return filepath.Join(s.dir, s.file) }
func (s *Set) metaPath() string  { return filepath.Join(s.dir, s.file+".meta.json") }

// Describe 一行状态描述，供启动日志使用。
func (s *Set) Describe() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.data) == 0 {
		return "无（本地副本缺失）"
	}
	when := "时间未知"
	if s.meta.UpdatedAt > 0 {
		when = time.UnixMilli(s.meta.UpdatedAt).Local().Format("2006-01-02 15:04")
	}
	return fmt.Sprintf("%d 个图标，%s，更新于 %s", s.meta.Icons, humanSize(s.meta.Size), when)
}

// humanSize 人类可读的字节数
func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// load 读本地副本与 meta。文件缺失或损坏只清空内存副本，不删盘上文件（留给人看）。
func (s *Set) load() {
	if data, err := os.ReadFile(s.metaPath()); err == nil {
		var meta Meta
		if json.Unmarshal(data, &meta) == nil {
			s.meta = meta
		}
	}

	body, err := os.ReadFile(s.localPath())
	if err != nil {
		log.Printf("图标集: 本地副本缺失（%s），将尝试下载", s.localPath())
		return
	}
	count, err := validate(body)
	if err != nil {
		log.Printf("图标集: 本地副本不可用（%v），将尝试重新下载", err)
		return
	}

	sum := sha256.Sum256(body)
	s.data = body
	s.meta.SHA256 = hex.EncodeToString(sum[:])
	s.meta.Size = int64(len(body))
	s.meta.Icons = count
	log.Printf("图标集: 已载入 %s", s.Describe())
}

// Ensure 保证本地有一份可用图标集：读得到就直接返回，否则同步下载一次。
func (s *Set) Ensure(ctx context.Context) error {
	s.mu.RLock()
	ok := len(s.data) > 0
	s.mu.RUnlock()
	if ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	updated, err := s.Check(ctx)
	if err != nil {
		return err
	}
	if !updated {
		return errors.New("本地副本缺失，且所有下载源都没拿到新数据")
	}
	return nil
}

// Start 后台维护：启动即检查一次，之后每 interval 检查一次。不阻塞调用方；
// 失败只在 Check 内部记日志，不影响已有副本与外层流程。
func (s *Set) Start(ctx context.Context, interval time.Duration) {
	go func() {
		s.Check(ctx)
	}()
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.Check(ctx)
			}
		}
	}()
}

// Check 按来源顺序检查一次更新，返回是否替换了本地副本。
// 任何一步失败都保留现有副本：绝不因为一次坏响应就把可用的图标集写坏。
func (s *Set) Check(ctx context.Context) (bool, error) {
	s.mu.RLock()
	cur := s.meta
	hasData := len(s.data) > 0
	s.mu.RUnlock()

	var lastErr error
	for _, url := range s.sources {
		etag := ""
		if hasData && url == cur.Source {
			etag = cur.ETag // 只对同一个源做条件请求
		}

		body, newETag, lastMod, notModified, err := s.fetch(ctx, url, etag)
		if err != nil {
			log.Printf("图标集: 源 %s 不可用: %v", url, err)
			lastErr = err
			continue
		}
		if notModified {
			log.Printf("图标集: 已是最新（%s 返回 304）", url)
			return false, nil
		}

		count, err := validate(body)
		if err != nil {
			log.Printf("图标集: 源 %s 返回的数据不可用: %v", url, err)
			lastErr = err
			continue
		}
		if count < s.minCount {
			err := fmt.Errorf("只有 %d 个图标，少于下限 %d", count, s.minCount)
			log.Printf("图标集: 源 %s 返回的数据不完整（%v），忽略", url, err)
			lastErr = err
			continue
		}

		sum := sha256.Sum256(body)
		sha := hex.EncodeToString(sum[:])
		meta := Meta{
			Source:    url,
			ETag:      newETag,
			LastMod:   lastMod,
			SHA256:    sha,
			Size:      int64(len(body)),
			Icons:     count,
			UpdatedAt: time.Now().UnixMilli(),
		}
		if hasData && sha == cur.SHA256 {
			// 内容没变，只把来源/ETag 记下来，方便下次直接用条件请求
			s.saveMeta(meta)
			log.Printf("图标集: 已是最新（%s 内容未变，%d 个图标）", url, count)
			return false, nil
		}

		if err := s.store(body, meta); err != nil {
			return false, err
		}
		log.Printf("图标集: 已更新 %s → %s，%d 个图标（源 %s）",
			humanSize(cur.Size), humanSize(int64(len(body))), count, url)
		return true, nil
	}

	if lastErr == nil {
		lastErr = errors.New("没有配置任何下载源")
	}
	log.Printf("图标集: 检查更新失败，继续使用现有副本: %v", lastErr)
	return false, lastErr
}

// fetch 发一次 GET；带 ETag 时命中 304 直接返回 notModified。
func (s *Set) fetch(ctx context.Context, url, etag string) (body []byte, newETag, lastMod string, notModified bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", "", false, err
	}
	req.Header.Set("User-Agent", "webnotes-iconset/1.0")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, "", "", false, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotModified:
		return nil, etag, resp.Header.Get("Last-Modified"), true, nil
	case resp.StatusCode != http.StatusOK:
		return nil, "", "", false, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err = io.ReadAll(io.LimitReader(resp.Body, maxDownload+1))
	if err != nil {
		return nil, "", "", false, err
	}
	if len(body) > maxDownload {
		return nil, "", "", false, fmt.Errorf("响应超过 %d MiB 上限", maxDownload>>20)
	}
	return body, resp.Header.Get("ETag"), resp.Header.Get("Last-Modified"), false, nil
}

// validate 解析并确认是一份可用的图标集，返回图标数。
func validate(body []byte) (int, error) {
	var set struct {
		Prefix string `json:"prefix"`
		Icons  map[string]struct {
			Body string `json:"body"`
		} `json:"icons"`
	}
	if err := json.Unmarshal(body, &set); err != nil {
		return 0, fmt.Errorf("不是合法 JSON: %w", err)
	}
	if set.Prefix == "" || len(set.Icons) == 0 {
		return 0, errors.New("缺少 prefix 或 icons，不像是图标集")
	}
	for _, icon := range set.Icons {
		if icon.Body != "" {
			return len(set.Icons), nil
		}
	}
	return 0, errors.New("icons 里没有 body，疑似只拿到了元数据")
}

// store 原子替换本地副本：tmp → fsync → rename，再更新内存与 meta。
func (s *Set) store(body []byte, meta Meta) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	tmp := s.localPath() + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	if f, err := os.Open(tmp); err == nil {
		f.Sync()
		f.Close()
	}
	if err := os.Rename(tmp, s.localPath()); err != nil {
		return err
	}

	s.mu.Lock()
	s.data, s.meta = body, meta
	s.mu.Unlock()

	s.saveMeta(meta)
	return nil
}

// saveMeta 落盘 meta；失败只记日志（不影响服务，下次会重算）。
func (s *Set) saveMeta(meta Meta) {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		log.Printf("图标集: meta 序列化失败: %v", err)
		return
	}
	tmp := s.metaPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		log.Printf("图标集: meta 写入失败: %v", err)
		return
	}
	if err := os.Rename(tmp, s.metaPath()); err != nil {
		log.Printf("图标集: meta 落盘失败: %v", err)
	}
}

// ServeHTTP 提供图标集本体：ETag 校验通过就回 304，避免每次重传几 MiB。
func (s *Set) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	data, meta := s.data, s.meta
	s.mu.RUnlock()

	if len(data) == 0 {
		http.Error(w, "icon set unavailable", http.StatusServiceUnavailable)
		return
	}

	header := w.Header()
	header.Set("Content-Type", "application/json; charset=utf-8")
	header.Set("ETag", `"`+meta.SHA256+`"`)
	// 可缓存但每次校验：命中 ETag 只花一个 304，换来后端一更新前端立刻拿到
	header.Set("Cache-Control", "no-cache")
	header.Set("Content-Length", strconv.Itoa(len(data)))

	if ifNoneMatch(r, meta.SHA256) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Write(data) // HEAD 请求的 body 由 net/http 自动丢弃
}

// ifNoneMatch 判断请求的 If-None-Match 是否命中当前版本。
func ifNoneMatch(r *http.Request, sha string) bool {
	value := r.Header.Get("If-None-Match")
	if value == "" || sha == "" {
		return false
	}
	want := `"` + sha + `"`
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(part), "W/"))
		if part == want || part == "*" {
			return true
		}
	}
	return false
}
