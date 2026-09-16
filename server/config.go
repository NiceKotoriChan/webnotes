// 全部配置硬编码在这个文件里：不使用 .env、环境变量或命令行参数。
// 要改配置就改下面的常量/变量，然后重启服务。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// Addr HTTP 监听地址（端口）
	Addr = ":8080"

	// DataDir webnotes 根目录：仓库目录（data.db + assets/）与 repos.json 都在这里。
	// 相对路径按 server/ 解析（go.mod 所在处，也就是 go run . 的工作目录）。
	DataDir = "../data"

	// IconDir 图标集存放目录：后端托管 /icons/* 的内容，允许自动更新
	IconDir = "../data/icons"

	// IconFile 图标集文件名，对外地址是 /icons/<IconFile>
	IconFile = "mdi.json"

	// IconCheckInterval 图标集更新检查间隔：启动时查一次，之后每 24h 一次
	IconCheckInterval = 24 * time.Hour

	// IconMinCount 有效图标集的最小图标数，低于此值视为坏数据（不覆盖本地副本）
	IconMinCount = 1000
)

// IconSources 图标集下载源，按顺序尝试，第一个成功的即采用（同一份 Iconify MDI 集合的不同镜像，
// 后两个是可达性备份）。
var IconSources = []string{
	"https://cdn.jsdelivr.net/gh/iconify/icon-sets@master/json/mdi.json",
	"https://fastly.jsdelivr.net/gh/iconify/icon-sets@master/json/mdi.json",
	"https://raw.githubusercontent.com/iconify/icon-sets/master/json/mdi.json",
}

// resolvePaths 把配置里的目录解析成绝对路径。
// 相对路径的基准是 server/ 目录，所以这里顺带校验工作目录：目录不对时直接报错，
// 免得静默地在别处建出一个空 data/ 来。
func resolvePaths() (dataDir, iconDir string, err error) {
	if dataDir, err = filepath.Abs(DataDir); err != nil {
		return "", "", fmt.Errorf("解析 DataDir %q: %w", DataDir, err)
	}
	if iconDir, err = filepath.Abs(IconDir); err != nil {
		return "", "", fmt.Errorf("解析 IconDir %q: %w", IconDir, err)
	}
	if !filepath.IsAbs(DataDir) {
		root := filepath.Dir(dataDir)
		if fi, statErr := os.Stat(filepath.Join(root, "server")); statErr != nil || !fi.IsDir() {
			return "", "", fmt.Errorf(
				"DataDir %q 是相对路径，按 server/ 目录解析得到 %q，但 %q 下没有 server/。"+
					"请在 server/ 目录里运行：go run .（或 VS Code 的 dev: backend 任务），编译好的二进制也从 server/ 里起",
				DataDir, dataDir, root)
		}
	}
	return dataDir, iconDir, nil
}
