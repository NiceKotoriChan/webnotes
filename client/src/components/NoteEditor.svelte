<script lang="ts">
  import { untrack } from 'svelte';
  import * as api from '../api';
  import type { Note } from '../api';
  import MarkdownViewer from './MarkdownViewer.svelte';

  let {
    repo,
    note,
    onSaved,
  }: {
    repo: string;
    note: Note;
    onSaved?: () => void;
  } = $props();

  // {#key note.id} 保证切换笔记即重挂；untrack 只取初始值
  const noteId = untrack(() => note.id);
  const initialTitle = untrack(() => note.title);
  const initialContent = untrack(() => note.content);
  let title = $state(initialTitle);
  let content = $state(initialContent);
  let saving = $state(false);
  let savedAt = $state(0);
  let errMsg = $state('');
  let savedTitle = $state(initialTitle);
  let savedContent = $state(initialContent);

  const dirty = $derived(title !== savedTitle || content !== savedContent);
  const saveState = $derived(saving ? '保存中…' : dirty ? '未保存' : savedAt ? '已保存' : '');

  async function save() {
    if (!dirty || saving) return;
    saving = true;
    errMsg = '';
    try {
      await api.updateNote(repo, noteId, { title, content });
      savedTitle = title;
      savedContent = content;
      savedAt = Date.now();
      onSaved?.();
    } catch (e: any) {
      errMsg = e.message;
    } finally {
      saving = false;
    }
  }

  function onKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === 's') {
      e.preventDefault();
      save();
    }
  }

  // 附件：拖入或粘贴即上传，插入 Markdown 链接
  async function uploadFile(file: File) {
    try {
      const meta = await api.uploadAsset(repo, file);
      const isImage = file.type.startsWith('image/');
      const md = isImage
        ? `![${meta.name}](${api.assetURL(repo, meta.id, true)})`
        : `[${meta.name}](${api.assetURL(repo, meta.id)})`;
      content = content + (content.endsWith('\n') || content === '' ? '' : '\n\n') + md + '\n';
    } catch (e: any) {
      errMsg = e.message;
    }
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

  // 5 秒自动保存
  $effect(() => {
    const timer = setInterval(() => save(), 5000);
    return () => clearInterval(timer);
  });
</script>

<svelte:window onkeydown={onKeydown} />

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="editor card" role="region" aria-label="笔记编辑区（可拖入或粘贴附件）"
  ondrop={onDrop} ondragover={(e) => e.preventDefault()} onpaste={onPaste}>
  <div class="title-row">
    <input class="title-input" bind:value={title} placeholder="无标题" />
    {#if saveState}<span class="save-state">{saveState}</span>{/if}
  </div>

  {#if errMsg}<p class="err">{errMsg}</p>{/if}

  <div class="split">
    <textarea bind:value={content} placeholder="开始书写，支持 Markdown；可直接拖入或粘贴附件"></textarea>
    <div class="preview">
      <MarkdownViewer content={content} />
    </div>
  </div>
</div>

<style>
  .editor { display: flex; flex-direction: column; flex: 1; min-height: 0; overflow: hidden; }
  .title-row {
    display: flex;
    align-items: baseline;
    gap: 12px;
    padding: 18px 24px 10px;
  }
  .title-input {
    flex: 1;
    font-size: 22px;
    font-weight: 700;
    border: none;
    border-radius: 0;
    padding: 0;
    background: transparent;
  }
  .title-input:focus { border: none; }
  .save-state { font-size: 12px; color: var(--muted); white-space: nowrap; }
  .split { display: flex; flex: 1; min-height: 0; }
  textarea {
    flex: 1;
    border: none;
    border-right: 1px solid var(--border);
    border-radius: 0;
    resize: none;
    padding: 12px 24px 24px;
    font-family: ui-monospace, "SFMono-Regular", Menlo, monospace;
    line-height: 1.6;
    background: transparent;
  }
  textarea:focus { border: none; border-right: 1px solid var(--border); }
  .preview { flex: 1; padding: 12px 24px 24px; overflow-y: auto; line-height: 1.6; }
  .err { color: var(--danger); padding: 0 24px 8px; margin: 0; }
</style>
