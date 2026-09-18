package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"webnotes/server/api"
	"webnotes/server/asset"
	"webnotes/server/icons"
	"webnotes/server/store"
)

func main() {
	if len(os.Args) > 1 {
		log.Printf("提示: 配置已全部硬编码在 server/config.go，忽略命令行参数 %v", os.Args[1:])
	}

	dataDir, iconDir, err := resolvePaths()
	if err != nil {
		log.Fatalf("配置有误: %v", err)
	}

	s, err := store.New(dataDir)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	a := asset.New()

	// 图标集：本地缺失才同步下载一份，之后每 IconCheckInterval 后台检查一次更新
	ic, err := icons.New(iconDir, IconSetFile, IconSources, IconMinCount)
	if err != nil {
		log.Fatalf("init icons: %v", err)
	}
	ctx := context.Background()
	if err := ic.Ensure(ctx); err != nil {
		log.Printf("图标集不可用（前端图标将加载失败）: %v", err)
	}
	ic.Start(ctx, IconCheckInterval)

	// 图标规则：自定义规则缺失时先用默认规则灌一份，之后由前端通过 /api 改它
	rules := icons.NewRules(iconDir, IconRulesDefaultFile, IconRulesCustomFile)
	if err := rules.Ensure(); err != nil {
		log.Printf("图标规则初始化失败（回退到默认设置的接口会报错）: %v", err)
	}

	// 逐仓库对账附件状态（清孤儿 tmp、uploading→ready、deleting 收尾）
	for _, ri := range s.ListRepos() {
		db, err := s.OpenRepo(ri.ID)
		if err != nil {
			log.Printf("reconcile %s: open: %v", ri.ID, err)
			continue
		}
		if err := a.Reconcile(db, s.RepoDir(ri.ID)); err != nil {
			log.Printf("reconcile %s: %v", ri.ID, err)
		}
		db.Close()
	}

	log.Printf("webnotes 数据目录: %s", dataDir)
	log.Printf("图标集 %s: %s（每 %s 检查更新）", "/icons/"+IconSetFile, ic.Describe(), IconCheckInterval)
	log.Printf("图标规则: %s / %s（目录 %s）", IconRulesDefaultFile, IconRulesCustomFile, iconDir)
	log.Printf("listening on %s", Addr)

	// /icons/* 由这里直接托管：图标集在内存里（后台会自动更新），两个规则文件每次现读盘
	mux := http.NewServeMux()
	mux.Handle("/icons/"+IconSetFile, ic)
	mux.Handle("/icons/"+IconRulesDefaultFile, icons.Static(iconDir, IconRulesDefaultFile))
	mux.Handle("/icons/"+IconRulesCustomFile, icons.Static(iconDir, IconRulesCustomFile))
	mux.Handle("/", api.NewRouter(s, a, rules))
	log.Fatal(http.ListenAndServe(Addr, mux))
}
