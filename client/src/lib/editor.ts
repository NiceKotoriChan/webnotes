// 正文编辑器的共享类型与小状态。
// 单独成文件是因为 <script setup> 里不能有 export 语句，类型必须放在外面。

export type EditorMode = 'edit' | 'split' | 'preview';

export type ToolbarAction =
  | 'bold'
  | 'italic'
  | 'strike'
  | 'h1'
  | 'h2'
  | 'h3'
  | 'ul'
  | 'ol'
  | 'task'
  | 'quote'
  | 'inlineCode'
  | 'codeBlock'
  | 'link'
  | 'image'
  | 'attach';

const MODE_KEY = 'webnotes:editorMode';

export function readEditorMode(): EditorMode {
  const v = localStorage.getItem(MODE_KEY);
  return v === 'edit' || v === 'split' || v === 'preview' ? v : 'split';
}

export function rememberEditorMode(m: EditorMode) {
  localStorage.setItem(MODE_KEY, m);
}
