<script setup lang="ts">
// 正文区：标题 + 标签 + Markdown 编辑/预览。
// 存库走 POST /notes/:id 的**部分更新** —— 只把真正变了的字段放进 body（见 spec/api.md）。
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue';
import * as api from '../../api';
import type { Note } from '../../api';
import { insertAt, prefixLines, wrapSelection, type TextSel } from '../../lib/md';
import { readEditorMode, rememberEditorMode, type EditorMode, type ToolbarAction } from '../../lib/editor';
import MarkdownEditor from './MarkdownEditor.vue';
import EditorToolbar from './EditorToolbar.vue';

const props = defineProps<{
  repo: string;
  note: Note;
  onSaved?: () => void;
}>();

// 初值只读一次（切笔记由父级 :key 重挂）
const noteId = props.note.id;
const saved = {
  title: props.note.title,
  content: props.note.content,
  tags: [...(props.note.tags ?? [])],
};

const title = ref(saved.title);
const content = ref(saved.content);
const tags = ref<string[]>([...saved.tags]);

const saving = ref(false);
const savedAt = ref(0);
const errMsg = ref('');
const tagInput = ref('');
const mode = ref<EditorMode>(readEditorMode());

const editorEl = ref<InstanceType<typeof MarkdownEditor>>();

const dirty = computed(
  () =>
    title.value !== saved.title ||
    content.value !== saved.content ||
    tags.value.join('\u0000') !== saved.tags.join('\u0000'),
);
const saveState = computed(() =>
  saving.value ? '保存中…' : dirty.value ? '未保存' : savedAt.value ? '已保存' : '',
);

let saveTimer: ReturnType<typeof setTimeout> | undefined;
let interval: ReturnType<typeof setInterval> | undefined;

onMounted(() => {
  interval = setInterval(save, 5000);
  window.addEventListener('keydown', onKeydown);
});
onBeforeUnmount(() => {
  if (interval) clearInterval(interval);
  if (saveTimer) clearTimeout(saveTimer);
  window.removeEventListener('keydown', onKeydown);
});

function scheduleSave() {
  clearTimeout(saveTimer);
  saveTimer = setTimeout(save, 600);
}

async function save() {
  if (!dirty.value || saving.value) return;
  // 只发变化的字段：没出现的字段后端不会动
  const patch: api.NotePatch = {};
  if (title.value !== saved.title) patch.title = title.value;
  if (content.value !== saved.content) patch.content = content.value;
  if (tags.value.join('\u0000') !== saved.tags.join('\u0000')) patch.tags = tags.value;
  if (Object.keys(patch).length === 0) return;

  saving.value = true;
  errMsg.value = '';
  try {
    await api.updateNote(props.repo, noteId, patch);
    if (patch.title !== undefined) saved.title = title.value;
    if (patch.content !== undefined) saved.content = content.value;
    if (patch.tags !== undefined) saved.tags = [...tags.value];
    savedAt.value = Date.now();
    props.onSaved?.();
  } catch (e: any) {
    errMsg.value = e.message;
  } finally {
    saving.value = false;
  }
}

function onKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key === 's') {
    e.preventDefault();
    clearTimeout(saveTimer);
    save();
  }
}

function setMode(m: EditorMode) {
  mode.value = m;
  rememberEditorMode(m);
}

// —— 工具栏：每个动作只是往源码里插一段 Markdown ——
function runAction(a: ToolbarAction) {
  const ed = editorEl.value;
  if (!ed) return;
  switch (a) {
    case 'bold': ed.transform((s) => wrapMd(s, '**')); break;
    case 'italic': ed.transform((s) => wrapMd(s, '*')); break;
    case 'strike': ed.transform((s) => wrapMd(s, '~~')); break;
    case 'inlineCode': ed.transform((s) => wrapMd(s, '`')); break;
    case 'h1': ed.transform((s) => prefixMd(s, '# ')); break;
    case 'h2': ed.transform((s) => prefixMd(s, '## ')); break;
    case 'h3': ed.transform((s) => prefixMd(s, '### ')); break;
    case 'ul': ed.transform((s) => prefixMd(s, '- ')); break;
    case 'ol': ed.transform((s) => prefixMd(s, '1. ')); break;
    case 'task': ed.transform((s) => prefixMd(s, '- [ ] ')); break;
    case 'quote': ed.transform((s) => prefixMd(s, '> ')); break;
    case 'codeBlock': ed.insert('```\n\n```\n', 4); break;
    case 'link': {
      const url = prompt('链接地址', 'https://');
      if (url === null || !url.trim()) return;
      ed.transform((s) => linkMd(s, url.trim()));
      break;
    }
    case 'image': pickFile('image/*'); break;
    case 'attach': pickFile(); break;
  }
}

// 三个薄封装：把 lib/md.ts 的纯文本变换套到当前选区上
const wrapMd = (s: TextSel, mark: string) => wrapSelection(s, mark);
const prefixMd = (s: TextSel, prefix: string) => prefixLines(s, prefix);
function linkMd(s: TextSel, url: string) {
  const picked = s.text.slice(s.start, s.end);
  return insertAt(s, `[${picked || '文字'}](${url})`);
}

// —— 标签 ——
function addTag() {
  const name = tagInput.value.trim().replace(/^#/, '');
  tagInput.value = '';
  if (!name || tags.value.includes(name)) return;
  tags.value = [...tags.value, name].sort((a, b) => a.localeCompare(b, 'zh-CN'));
  scheduleSave();
}
function removeTag(name: string) {
  tags.value = tags.value.filter((t) => t !== name);
  scheduleSave();
}

// —— 上传：插入 Markdown 引用，不再生成 HTML ——
interface UploadEntry {
  name: string;
  loaded: number;
  total: number;
  status: 'uploading' | 'done' | 'error';
}
const uploads = ref<UploadEntry[]>([]);

async function uploadFile(file: File) {
  const entry = reactive<UploadEntry>({
    name: file.name,
    loaded: 0,
    total: file.size || 1,
    status: 'uploading',
  });
  uploads.value.push(entry);
  try {
    const meta = await api.uploadAsset(props.repo, file, (loaded, total) => {
      entry.loaded = loaded;
      entry.total = total;
    });
    entry.status = 'done';
    const url = api.assetURL(props.repo, meta.id, file.type.startsWith('image/'));
    const alt = meta.name.replace(/[[\]]/g, '');
    editorEl.value?.insert(file.type.startsWith('image/') ? `![${alt}](${url})` : `[${alt}](${url})`);
  } catch (e: any) {
    entry.status = 'error';
    errMsg.value = e.message;
  }
}

function onDrop(e: DragEvent) {
  const files = e.dataTransfer?.files;
  if (!files?.length) return;
  e.preventDefault();
  for (const f of files) uploadFile(f);
}

function onPaste(e: ClipboardEvent) {
  for (const it of e.clipboardData?.items ?? []) {
    if (it.kind === 'file') {
      const f = it.getAsFile();
      if (f) {
        e.preventDefault();
        uploadFile(f);
      }
    }
  }
}

function pickFile(accept?: string) {
  const input = document.createElement('input');
  input.type = 'file';
  if (accept) input.accept = accept;
  input.onchange = () => {
    const f = input.files?.[0];
    if (f) uploadFile(f);
  };
  input.click();
}

function pct(u: UploadEntry) {
  if (!u.total) return 0;
  return Math.min(100, Math.round((u.loaded / u.total) * 100));
}
</script>

<template>
  <div class="editor-root" @drop="onDrop" @dragover.prevent @paste="onPaste">
    <div class="head">
      <input v-model="title" class="title-input" placeholder="无标题" @input="scheduleSave" />
      <div class="meta">
        <span v-if="saveState" class="save-state">{{ saveState }}</span>
        <span class="time" :title="'新建于 ' + new Date(note.created_at).toLocaleString('zh-CN')">
          改动 {{ new Date(note.updated_at).toLocaleString('zh-CN', { hour12: false }) }}
        </span>
      </div>
    </div>

    <div class="tag-row">
      <span v-for="t in tags" :key="t" class="chip">
        #{{ t }}
        <button class="chip-x" title="移除" @click="removeTag(t)"><Icon icon="mdi:close" width="11" height="11" /></button>
      </span>
      <form class="tag-add" @submit.prevent="addTag">
        <input v-model="tagInput" placeholder="+ 标签" @keydown.enter.prevent="addTag" />
      </form>
    </div>

    <p v-if="errMsg" class="err">{{ errMsg }}</p>

    <EditorToolbar :mode="mode" @action="runAction" @set-mode="setMode" />

    <MarkdownEditor ref="editorEl" v-model="content" :mode="mode" @update:model-value="scheduleSave" />

    <div v-if="uploads.length" class="uploads">
      <div v-for="(u, i) in uploads" :key="i" class="upload">
        <span class="upload-name">{{ u.name }}</span>
        <span v-if="u.status === 'uploading'" class="upload-bar"><i :style="{ width: pct(u) + '%' }"></i></span>
        <span v-else-if="u.status === 'done'" class="upload-done"><Icon icon="mdi:check" width="12" height="12" /></span>
        <span v-else class="upload-error"><Icon icon="mdi:close" width="12" height="12" /></span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.editor-root {
  position: relative;
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px 6px;
  flex-shrink: 0;
}
.title-input {
  flex: 1;
  min-width: 0;
  padding: 2px 0;
  font-size: 19px;
  font-weight: 600;
  background: transparent;
  border: none;
  border-radius: 0;
}
.title-input:focus {
  border: none;
  box-shadow: none;
}
.meta {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  font-size: 12px;
  color: var(--fg-muted);
}
.tag-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  padding: 0 16px 10px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--border-muted);
}
.chip {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 1px 4px 1px 8px;
  font-size: 12px;
  color: var(--accent-fg);
  background: var(--accent-subtle);
  border-radius: 999px;
}
.chip-x {
  display: inline-flex;
  padding: 1px;
  color: inherit;
  background: none;
  border: none;
  border-radius: 50%;
  cursor: pointer;
  opacity: 0.7;
}
.chip-x:hover {
  opacity: 1;
}
.tag-add input {
  width: 90px;
  padding: 2px 8px;
  font-size: 12px;
  background: transparent;
  border: 1px dashed var(--border-default);
  border-radius: 999px;
}
.err {
  color: var(--danger-fg);
  padding: 8px 16px 0;
  margin: 0;
  font-size: 13px;
}
.uploads {
  position: absolute;
  bottom: 8px;
  left: 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-width: 280px;
}
.upload {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  background: var(--canvas-overlay);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  box-shadow: var(--shadow-default);
  font-size: 12px;
}
.upload-name {
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--fg-default);
}
.upload-bar {
  flex: 1;
  min-width: 60px;
  height: 4px;
  background: var(--btn-active-bg);
  border-radius: 2px;
  overflow: hidden;
}
.upload-bar i {
  display: block;
  height: 100%;
  background: var(--accent-emphasis);
}
.upload-done {
  color: var(--success-fg);
}
.upload-error {
  color: var(--danger-fg);
}
</style>
