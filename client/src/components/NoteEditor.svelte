<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import * as api from '../api';
  import type { Note } from '../api';
  import TagsPanel from './TagsPanel.svelte';

  export let repo: string;
  export let note: Note;

  const dispatch = createEventDispatcher<{ saved: void }>();

  let title: string = note.title;
  let content: string = note.content;
  let dirty = false;
  let saving = false;
  let errMsg = '';
  let lastSaved: Note = note;

  // 切换笔记时重置
  $: if (note.id !== lastSaved.id) {
    title = note.title;
    content = note.content;
    dirty = false;
    lastSaved = note;
  }

  $: dirty = title !== note.title || content !== note.content;

  async function save() {
    if (!dirty || saving) return;
    saving = true;
    errMsg = '';
    try {
      const updated = await api.updateNote(repo, note.id, title, content);
      lastSaved = { ...note, ...updated };
      dirty = false;
      dispatch('saved');
    } catch (e: any) {
      errMsg = e.message;
    } finally {
      saving = false;
    }
  }

  // Ctrl/Cmd+S 保存
  function onKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === 's') {
      e.preventDefault();
      save();
    }
  }

  async function uploadFile(file: File) {
    try {
      const meta = await api.uploadAsset(file);
      const isImage = file.type.startsWith('image/');
      const md = isImage
        ? `![${meta.name}](${api.assetURL(meta.id, true)})`
        : `[${meta.name}](${api.assetURL(meta.id)})`;
      content = content + (content.endsWith('\n') || content === '' ? '' : '\n\n') + md + '\n';
      dirty = true;
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
    const items = e.clipboardData?.items;
    if (!items) return;
    for (const it of items) {
      if (it.kind === 'file') {
        const f = it.getAsFile();
        if (f) {
          e.preventDefault();
          uploadFile(f);
        }
      }
    }
  }

  let fileInput: HTMLInputElement;
  function onFilePick(e: Event) {
    const target = e.target as HTMLInputElement;
    if (!target.files) return;
    for (const f of target.files) uploadFile(f);
    target.value = '';
  }

  onMount(() => {
    const interval = setInterval(() => { if (dirty) save(); }, 5000);
    return () => clearInterval(interval);
  });
</script>

<div class="editor" on:keydown={onKeydown} on:drop={onDrop} on:dragover|preventDefault on:paste={onPaste}>
  <header>
    <input class="title" bind:value={title} placeholder="标题" />
    <button class="primary" on:click={save} disabled={!dirty || saving}>
      {saving ? '保存中…' : '保存'}
    </button>
    <button on:click={() => fileInput.click()}>插入附件</button>
    <input bind:this={fileInput} type="file" multiple on:change={onFilePick} style="display:none" />
  </header>

  {#if errMsg}<p class="err">{errMsg}</p>{/if}

  <div class="split">
    <textarea bind:value={content} placeholder="Markdown 正文…"></textarea>
    <div class="preview markdown">{@html markdown(content)}</div>
  </div>

  <TagsPanel {repo} noteId={note.id} />
</div>

<script context="module" lang="ts">
  // 轻量 Markdown 渲染（避免引入完整库）
  export function markdown(s: string): string {
    const esc = (t: string) => t
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');
    const lines = s.split('\n');
    const out: string[] = [];
    let inCode = false;
    for (const line of lines) {
      if (line.startsWith('```')) {
        if (inCode) { out.push('</code></pre>'); inCode = false; }
        else { out.push('<pre><code>'); inCode = true; }
        continue;
      }
      if (inCode) { out.push(esc(line)); continue; }
      let l = esc(line);
      l = l.replace(/^###\s+(.*)$/, '<h3>$1</h3>');
      l = l.replace(/^##\s+(.*)$/, '<h2>$1</h2>');
      l = l.replace(/^#\s+(.*)$/, '<h1>$1</h1>');
      l = l.replace(/^-\s+(.*)$/, '<li>$1</li>');
      l = l.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '<img alt="$1" src="$2">');
      l = l.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>');
      out.push(l);
    }
    return out.join('\n');
  }
</script>

<style>
  .editor { display: flex; flex-direction: column; flex: 1; min-height: 0; }
  header {
    display: flex;
    gap: 8px;
    padding: 8px;
    border-bottom: 1px solid var(--border);
  }
  header .title { flex: 1; font-size: 16px; padding: 6px 8px; }
  .split { display: flex; flex: 1; min-height: 0; }
  textarea {
    flex: 1;
    border: none;
    border-right: 1px solid var(--border);
    border-radius: 0;
    resize: none;
    padding: 12px;
    font-family: ui-monospace, "SFMono-Regular", Menlo, monospace;
    line-height: 1.5;
  }
  .preview {
    flex: 1;
    padding: 12px;
    overflow-y: auto;
    line-height: 1.5;
  }
  .markdown img { max-width: 100%; }
  .markdown pre { background: var(--hover); padding: 8px; overflow-x: auto; }
  .markdown li { margin-left: 1em; }
  .err { color: var(--danger); padding: 8px; }
</style>
