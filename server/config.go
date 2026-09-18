// 项目配置：全部硬编码在本文件里 —— 不使用 .env、环境变量或命令行参数。
// 改配置就是改这里的常量，然后重启服务。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// —— 服务 ——

// Addr HTTP 监听地址（端口）
const Addr = ":8080"

// —— 目录 ——
//
// 相对路径以 server/ 为基准（go.mod 所在处，也是 go run . 的工作目录）。

const (
	// DataDir 数据根目录：repos.json 与各仓库目录（data.db + assets/）
	DataDir = "../.data"

	// IconDir 图标目录：图标集与图标规则都放这里，后端托管 /icons/* 时从这里读
	IconDir = "../.data/icons"
)

// —— 图标 ——

const (
	// IconSetFile 图标集文件名，对外地址是 /icons/<IconSetFile>。
	// 它会从远端自动更新，当前版本的元信息记在 <IconSetFile>.meta.json
	IconSetFile = "mdi.json"

	// IconRulesDefaultFile 默认图标规则文件名（内置的通用规则）
	IconRulesDefaultFile = "mdi_rules_default.json"

	// IconRulesCustomFile 自定义图标规则文件名（用户自己的规则，压过默认规则）
	IconRulesCustomFile = "mdi_rules_custom.json"

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

// resolvePaths 把配置里的目录解析成绝对路径（相对路径的基准是 server/），并校验启动位置。
// 从别处启动会把相对路径算到不相干的位置，静默地建出一个空数据目录 —— 所以这里直接报错。
func resolvePaths() (dataDir, iconDir string, err error) {
	if dataDir, err = absPath(DataDir); err != nil {
		return "", "", err
	}
	if iconDir, err = absPath(IconDir); err != nil {
		return "", "", err
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

// absPath 解析单个配置路径，出错时带上出问题的那个值。
func absPath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("解析配置路径 %q: %w", p, err)
	}
	return abs, nil
}
