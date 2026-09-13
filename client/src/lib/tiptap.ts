// TipTap 编辑器封装：Vue 3 手动集成。
// 导出 createEditor 供组件持有；命令封装成薄函数，组件模板可直接绑定。
import { Editor } from '@tiptap/core';
import StarterKit from '@tiptap/starter-kit';
import Image from '@tiptap/extension-image';
import Link from '@tiptap/extension-link';
import TaskList from '@tiptap/extension-task-list';
import TaskItem from '@tiptap/extension-task-item';
import { mdToHtml } from './md';

// 统一扩展集：StarterKit（段落/标题/粗斜体/列表/引用/代码）+ 链接 + 图片 + 任务列表
export function createEditor({
  initialHtml,
  onUpdate,
}: {
  initialHtml: string;
  onUpdate: (html: string) => void;
}): Editor {
  return new Editor({
    extensions: [
      StarterKit.configure({
        link: false, // 链接用独立的 Link 扩展（可交互，配置更细）
      }),
      Link.configure({
        openOnClick: false, // 编辑态点击不跳转，由工具栏/按钮打开
        autolink: true,
        HTMLAttributes: { rel: 'noopener noreferrer', target: '_blank' },
      }),
      Image.configure({
        allowBase64: false,
        // 仓库内附件是 /api/repos/... 同源相对 URL，必须放行
        inline: false,
      }),
      TaskList,
      TaskItem.configure({ nested: false }),
    ],
    content: initialHtml,
    autofocus: false,
    editorProps: {
      attributes: {
        class: 'tiptap',
      },
    },
    onUpdate: ({ editor }) => onUpdate(editor.getHTML()),
  });
}

// 命令：给工具栏用的薄封装（组件里 editor.chain().focus()... 也可直接用）
export const commands = {
  bold: (e: Editor) => e.chain().focus().toggleBold().run(),
  italic: (e: Editor) => e.chain().focus().toggleItalic().run(),
  strike: (e: Editor) => e.chain().focus().toggleStrike().run(),
  heading: (e: Editor, level: 1 | 2 | 3) => e.chain().focus().toggleHeading({ level }).run(),
  bulletList: (e: Editor) => e.chain().focus().toggleBulletList().run(),
  orderedList: (e: Editor) => e.chain().focus().toggleOrderedList().run(),
  taskList: (e: Editor) => e.chain().focus().toggleTaskList().run(),
  blockquote: (e: Editor) => e.chain().focus().toggleBlockquote().run(),
  codeBlock: (e: Editor) => e.chain().focus().toggleCodeBlock().run(),
  code: (e: Editor) => e.chain().focus().toggleCode().run(),
  undo: (e: Editor) => e.chain().focus().undo().run(),
  redo: (e: Editor) => e.chain().focus().redo().run(),
};

// 判断当前选区是否命中某个 mark/node（用于工具栏高亮）
export function isActive(e: Editor, type: string, attrs?: Record<string, unknown>): boolean {
  return e.isActive(type, attrs);
}

// 插入图片（附件已上传，把 /api/... 相对 URL 塞进正文）
export function insertImage(e: Editor, src: string, alt = '') {
  e.chain().focus().setImage({ src, alt }).run();
}

// 设置链接：选区文字变链接；有链接则移除
export function setLink(e: Editor, url: string) {
  if (url.trim()) {
    e.chain().focus().setLink({ href: url.trim() }).run();
  } else {
    e.chain().focus().unsetLink().run();
  }
}

// 打开笔记：把 Markdown content 渲染成 HTML 后注入（不触发 onUpdate，避免误存）
export function loadContent(e: Editor, md: string) {
  e.commands.setContent(mdToHtml(md), { emitUpdate: false });
}

// 销毁
export function destroyEditor(e: Editor | null) {
  e?.destroy();
}
