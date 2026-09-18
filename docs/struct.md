## struct

关于总体结构，数据不对外暴露，打包目录为 /usr/bin/webnotes 和 /usr/share/webnotes/

前者放 go 编译的可执行文件，后者放数据以及前端编译文件

后者具体结构：

- web/ 放前端文件
- data/ 放数据：repos.json 以及各个仓库目录
- icon/ mdi.json,mdi_rules_default.json,mdi_rules_custom.json

## 仓库

也可以叫工作区，里面的资源是完全隔开的
