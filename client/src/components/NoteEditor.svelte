<script lang="ts">
  // 右侧 content 组件：单栏 Markdown 预览，点击进入编辑
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
  const initialContent = untrack(() => note.content);
  let content = $state(initialContent);
  let editing = $state(false);
  let textareaEl: HTMLTextAreaElement | undefined = $state();
  let saving = $state(false);
  let savedAt = $state(0);
  let errMsg = $state('');
  let savedContent = $state(initialContent);

  const dirty = $derived(content !== savedContent);
  const saveState = $derived(saving ? '保存中…' : dirty ? '未保存' : savedAt ? '已保存' : '');

  // 保存：title 取自 note prop（树内重命名后自动同步），content 取本地编辑态
  async function save() {
    if (!dirty || saving) return;
    saving = true;
    errMsg = '';
    try {
      await api.updateNote(repo, noteId, { title: note.title, content });
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
    } else if (e.key === 'Escape' && editing) {
      editing = false;
    }
  }

  // 点击预览进入编辑（链接除外，让 a 标签正常工作）
  function onPreviewClick(e: MouseEvent) {
    if ((e.target as HTMLElement).closest('a')) return;
    editing = true;
  }

  // 附件：拖入或粘贴即上传，插入 Markdown 链接；同时切到编辑模式
  async function uploadFile(file: File) {
    editing = true;
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

  // 切到编辑模式时聚焦 textarea
  $effect(() => {
    if (editing && textareaEl) textareaEl.focus();
  });

  // 5 秒自动保存
  $effect(() => {
    const timer = setInterval(() => save(), 5000);
    return () => clearInterval(timer);
  });
</script>

<svelte:window onkeydown={onKeydown} />

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="editor" role="region" aria-label="笔记编辑区（可拖入或粘贴附件）"
  ondrop={onDrop} ondragover={(e) => e.preventDefault()} onpaste={onPaste}>
  {#if errMsg}<p class="err">{errMsg}</p>{/if}

  {#if editing}
    <textarea bind:this={textareaEl} bind:value={content}
      placeholder="开始书写，支持 Markdown；可直接拖入或粘贴附件"></textarea>
  {:else if content.trim()}
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
    <div class="preview" onclick={onPreviewClick}>
      <MarkdownViewer content={content} />
    </div>
  {:else}
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
    <div class="empty" onclick={() => (editing = true)}>点击开始编辑</div>
  {/if}

  {#if saveState}<span class="save-state">{saveState}</span>{/if}
</div>

<style>
  .editor { display: flex; flex-direction: column; flex: 1; min-height: 0; overflow: hidden; position: relative; }
  .preview { flex: 1; padding: 24px; overflow-y: auto; line-height: 1.6; cursor: text; }
  .empty { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--muted); cursor: text; }
  textarea {
    flex: 1;
    border: none;
    border-radius: 0;
    resize: none;
    padding: 24px;
    font-family: ui-monospace, "SFMono-Regular", Menlo, monospace;
    line-height: 1.6;
    background: transparent;
    outline: none;
  }
  .save-state { position: absolute; bottom: 8px; right: 12px; font-size: 12px; color: var(--muted); pointer-events: none; }
  .err { color: var(--danger); padding: 8px 24px 0; margin: 0; }
</style>
