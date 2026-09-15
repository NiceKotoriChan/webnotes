<script setup lang="ts">
// 右侧内容区：所见即所得 Markdown 编辑器（TipTap）。
// 内容变化 → 防抖转 Markdown 存库；拖入/粘贴/工具栏附件自动上传并插入正文。
import { ref, shallowRef, computed, reactive, onMounted, onBeforeUnmount } from 'vue';
import type { Editor } from '@tiptap/core';
import * as api from '../../api';
import type { Note } from '../../api';
import { createEditor, destroyEditor } from '../../lib/tiptap';
import { mdToHtml, htmlToMd } from '../../lib/md';
import EditorToolbar from './EditorToolbar.vue';

const props = defineProps<{
  repo: string;
  note: Note;
  onSaved?: () => void;
}>();

// 初值只读一次（切笔记由父级 :key 重挂）
const noteId = props.note.id;
const initialContent = props.note.data;
const initialHtml = mdToHtml(initialContent);

const content = ref(initialContent);
const saving = ref(false);
const savedAt = ref(0);
const errMsg = ref('');
const savedContent = ref(initialContent);

interface UploadEntry {
  name: string;
  loaded: number;
  total: number;
  status: 'uploading' | 'done' | 'error';
}
const uploads = ref<UploadEntry[]>([]);

const containerEl = ref<HTMLElement>();
const editor = shallowRef<Editor | null>(null);

const dirty = computed(() => content.value !== savedContent.value);
const saveState = computed(() =>
  saving.value ? '保存中…' : dirty.value ? '未保存' : savedAt.value ? '已保存' : '',
);

let saveTimer: ReturnType<typeof setTimeout> | undefined;
let interval: ReturnType<typeof setInterval> | undefined;

onMounted(() => {
  if (!containerEl.value) return;
  const e = createEditor({ initialHtml, onUpdate: onEditorUpdate });
  editor.value = e;
  containerEl.value.appendChild(e.options.element as HTMLElement);
  interval = setInterval(() => save(), 5000);
  window.addEventListener('keydown', onKeydown);
});

onBeforeUnmount(() => {
  destroyEditor(editor.value);
  editor.value = null;
  if (interval) clearInterval(interval);
  if (saveTimer) clearTimeout(saveTimer);
  window.removeEventListener('keydown', onKeydown);
});

async function save() {
  if (!dirty.value || saving.value) return;
  saving.value = true;
  errMsg.value = '';
  try {
    await api.updateNote(props.repo, noteId, { title: props.note.title, data: content.value });
    savedContent.value = content.value;
    savedAt.value = Date.now();
    props.onSaved?.();
  } catch (e: any) {
    errMsg.value = e.message;
  } finally {
    saving.value = false;
  }
}

function onEditorUpdate(html: string) {
  content.value = htmlToMd(html);
  clearTimeout(saveTimer);
  saveTimer = setTimeout(save, 600);
}

function onKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key === 's') {
    e.preventDefault();
    clearTimeout(saveTimer);
    save();
  }
}

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
    const html = file.type.startsWith('image/')
      ? `<img src="${url}" alt="${escapeAttr(meta.name)}">`
      : `<a href="${url}">${escapeAttr(meta.name)}</a>`;
    editor.value?.chain().focus().insertContent(html).run();
  } catch (e: any) {
    entry.status = 'error';
    errMsg.value = e.message;
  }
}

function escapeAttr(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function onDrop(e: DragEvent) {
  const files = e.dataTransfer?.files;
  if (!files) return;
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
    <p v-if="errMsg" class="err">{{ errMsg }}</p>

    <EditorToolbar
      v-if="editor"
      :editor="editor"
      :on-pick-image="() => pickFile('image/*')"
      :on-pick-attach="() => pickFile()"
    />

    <div ref="containerEl" class="tiptap-host"></div>

    <span v-if="saveState" class="save-state">{{ saveState }}</span>

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
.err {
  color: var(--danger-fg);
  padding: 8px 24px 0;
  margin: 0;
}
.tiptap-host {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
.save-state {
  position: absolute;
  bottom: 8px;
  right: 12px;
  font-size: 12px;
  color: var(--fg-muted);
  pointer-events: none;
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
