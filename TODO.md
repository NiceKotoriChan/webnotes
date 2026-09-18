不确定，实现自动推送到 hexo

不确定，修复仓库索引的 date 恒为 0（store.reconcile 补进索引的仓库不回填日期，repos.json 两条都是 0）

不确定，删掉 idx_notes_created：列表不再排序后它成了死索引，没有查询用得上

不确定，GET /trash（按 deleted_at DESC）、GET /tags（按名升序）、GET /assets（按 date 倒序）三处仍是后端排序，是否也改成数据库默认顺序
