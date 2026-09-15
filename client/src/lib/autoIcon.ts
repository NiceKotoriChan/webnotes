// 文档树图标自动匹配：按标题关键词/扩展名映射到 mdi 图标名（不含前缀）。
// 自定义图标存 notes.icon（null = 用自动匹配）。
// 常用图标（图标选择器默认网格）
export const PRESET_ICONS = [
  'file-document-outline', 'file-outline', 'folder', 'folder-open', 'notebook', 'book', 'book-open-variant',
  'calendar', 'calendar-range', 'clock-outline', 'checkbox-marked', 'format-list-bulleted', 'format-list-checks',
  'code-tags', 'console', 'code-braces', 'bug', 'database', 'server', 'cloud-outline',
  'lock-outline', 'key', 'account-outline', 'account-group', 'heart', 'star', 'flag', 'target',
  'lightbulb-on-outline', 'rocket-launch', 'wallet', 'cart', 'email-outline', 'message-text-outline',
  'image', 'video', 'music', 'map-outline', 'web', 'link', 'paperclip', 'tag',
  'cog', 'home', 'bell-outline', 'archive',
];

const EXT_ICON: Record<string, string> = {
  md: 'file-document-outline', txt: 'file-document-outline', text: 'file-document-outline',
  js: 'file-code-outline', ts: 'file-code-outline', jsx: 'file-code-outline', tsx: 'file-code-outline',
  go: 'file-code-outline', py: 'file-code-outline', rb: 'file-code-outline', rs: 'file-code-outline',
  java: 'file-code-outline', c: 'file-code-outline', h: 'file-code-outline', cpp: 'file-code-outline',
  sh: 'console', bash: 'console', zsh: 'console',
  json: 'code-braces', yaml: 'code-braces', yml: 'code-braces', toml: 'code-braces', xml: 'code-braces',
  html: 'file-code-outline', css: 'file-code-outline', scss: 'file-code-outline', vue: 'file-code-outline', svelte: 'file-code-outline',
  png: 'file-image-outline', jpg: 'file-image-outline', jpeg: 'file-image-outline', gif: 'file-image-outline', svg: 'file-image-outline', webp: 'file-image-outline',
  mp3: 'file-music-outline', wav: 'file-music-outline', flac: 'file-music-outline', ogg: 'file-music-outline',
  mp4: 'video', mov: 'video', mkv: 'video', webm: 'video',
  zip: 'folder-zip-outline', tar: 'folder-zip-outline', gz: 'folder-zip-outline', rar: 'folder-zip-outline',
  pdf: 'file-document-outline', doc: 'file-document-outline', docx: 'file-document-outline', xls: 'file-document-outline', ppt: 'file-document-outline',
};

const RULES: [RegExp, string][] = [
  // —— 操作系统 / 发行版 ——
  [/linux-mint/i, 'linux-mint'],
  [/linux|内核|kernel/i, 'linux'],
  [/ubuntu/i, 'ubuntu'],
  [/debian/i, 'debian'],
  [/fedora/i, 'fedora'],
  [/archlinux|arch\s*linux|\barch\b/i, 'arch'],
  [/macos|mac\s*os|苹果|apple|iphone|ipad|\bios\b/i, 'apple'],
  [/windows|微软|microsoft/i, 'microsoft-windows'],
  [/android/i, 'android'],
  // —— 开发 / 运维 ——
  [/docker/i, 'docker'],
  [/kubernetes|\bk8s\b/i, 'kubernetes'],
  [/github/i, 'github'],
  [/gitlab/i, 'gitlab'],
  [/\bgit\b/i, 'git'],
  [/terminal|shell|bash|zsh|终端|命令行/i, 'console'],
  // —— 编程语言 / 框架 ——
  [/typescript|\bts\b/i, 'language-typescript'],
  [/javascript|\bjs\b/i, 'language-javascript'],
  [/python|\bpy\b/i, 'language-python'],
  [/golang|\bgo\b/i, 'language-go'],
  [/rust|\brs\b/i, 'language-rust'],
  [/java/i, 'language-java'],
  [/c\+\+|cpp/i, 'language-cpp'],
  [/\bc\b|c语言/i, 'language-c'],
  [/c#|csharp/i, 'language-csharp'],
  [/php/i, 'language-php'],
  [/ruby/i, 'language-ruby'],
  [/swift/i, 'language-swift'],
  [/kotlin/i, 'language-kotlin'],
  [/lua/i, 'language-lua'],
  [/haskell/i, 'language-haskell'],
  [/react/i, 'react'],
  [/vue/i, 'vuejs'],
  [/node/i, 'nodejs'],
  // —— 通用 ——
  [/readme/i, 'book-open-variant'],
  [/todo|待办|任务清单|task/i, 'format-list-checks'],
  [/会议|meeting|纪要/i, 'calendar-range'],
  [/日记|diary|journal|周记/i, 'book'],
  [/笔记|note|study|学习/i, 'notebook'],
  [/密码|password|secret|密钥/i, 'lock-outline'],
  [/代码|code|snippet|脚本|script/i, 'code-tags'],
  [/收藏|书签|bookmark/i, 'bookmark'],
  [/图片|相册|照片|photo|image|gallery/i, 'image'],
  [/音乐|歌单|music|song/i, 'music'],
  [/视频|video|movie|电影/i, 'video'],
  [/数据库|database|db/i, 'database'],
  [/部署|deploy|服务器|server|运维/i, 'server'],
  [/购物|shopping|清单/i, 'cart'],
  [/邮件|email|mail|收件/i, 'email-outline'],
  [/设置|配置|config|settings|偏好/i, 'cog'],
  [/用户|个人|profile|简历/i, 'account-outline'],
  [/计划|plan|日程|schedule/i, 'calendar'],
  [/重要|star|精选|收藏/i, 'star'],
  [/bug|问题|issue|缺陷/i, 'bug'],
  [/灵感|idea|想法|创意/i, 'lightbulb-on-outline'],
  [/财务|money|账单|finance|理财/i, 'wallet'],
  [/地图|旅行|travel|map|攻略/i, 'map-outline'],
  [/链接|网址|url/i, 'link'],
  [/归档|archive/i, 'archive'],
];

// 去掉扩展名后再做关键词匹配
export function autoIcon(title: string): string {
  const t = title.trim();
  if (!t) return 'file-document-outline';

  const ext = t.match(/\.([a-z0-9]+)$/i)?.[1]?.toLowerCase();
  if (ext && EXT_ICON[ext]) return EXT_ICON[ext];

  for (const [re, icon] of RULES) {
    if (re.test(t)) return icon;
  }
  return 'file-document-outline';
}

