package icons

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

// Static 托管图标目录里的一份静态文件（图标规则就是这种：纯静态、不下载、用户随时可改）。
// 每次请求都现读盘：文件只有几 KiB，换来「改完立刻生效」，不必重启，也省掉一套重载逻辑。
// 路径由调用方写死（不接受请求里的路径），所以没有目录穿越问题。
func Static(dir, name string) http.Handler {
	return staticFile{path: filepath.Join(dir, name)}
}

type staticFile struct{ path string }

func (f staticFile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := os.ReadFile(f.path)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	sum := sha256.Sum256(data)
	etag := hex.EncodeToString(sum[:])

	header := w.Header()
	header.Set("Content-Type", "application/json; charset=utf-8")
	header.Set("ETag", `"`+etag+`"`)
	// 可缓存但每次校验：文件改了就靠 ETag 立刻生效，没改只花一个 304
	header.Set("Cache-Control", "no-cache")
	header.Set("Content-Length", strconv.Itoa(len(data)))

	if ifNoneMatch(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Write(data) // HEAD 请求的 body 由 net/http 自动丢弃
}
