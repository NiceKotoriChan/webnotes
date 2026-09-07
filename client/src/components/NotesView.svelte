<script lang="ts">
  import { notesStore } from '../stores';
  import * as api from '../api';
  import NoteEditor from './NoteEditor.svelte';

  export let repo: string;

  let query = '';
  let selected: api.Note | null = null;
  let errMsg = '';

  $: notesStore.refresh({ q: query });

  function open(n: api.Note) {
    selected = n;
  }

  async function createNew() {
    try {
      const n = await notesStore.create('', '');
      if (n) selected = n;
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  async function remove(id: string) {
    if (!confirm('删除该笔记？')) return;
    try {
      await notesStore.remove(id);
      if (selected?.id === id) selected = null;
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  function fmtTime(ts: number) {
    return new Date(ts).toLocaleString('zh-CN', { hour12: false });
  }
</script>

<div class="layout">
  <aside>
    <div class="search-row">
      <input bind:value={query} placeholder="搜索…" />
      <button class="primary" on:click={createNew}>新建笔记</button>
    </div>

    {#if errMsg}<p class="err">{errMsg}</p>{/if}

    <ul class="note-list">
      {#each $notesStore as n (n.id)}
        <li class:active={selected?.id === n.id}>
          <a href="#" on:click|preventDefault={() => open(n)} class="title">
            {n.title || '(无标题)'}
          </a>
          <span class="mtime">{fmtTime(n.mtime)}</span>
          <button class="danger" on:click={() => remove(n.id)}>×</button>
        </li>
      {:else}
        <li class="empty">暂无笔记</li>
      {/each}
    </ul>
  </aside>

  <section class="editor-wrap">
    {#if selected}
      <NoteEditor {repo} note={selected} on:saved={() => notesStore.refresh({ q: query })} />
    {:else}
      <div class="placeholder">选择左侧笔记，或点击「新建笔记」</div>
    {/if}
  </section>
</div>

<style>
  .layout { display: flex; flex: 1; min-height: 0; }
  aside {
    width: 280px;
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
  .search-row {
    display: flex;
    gap: 4px;
    padding: 8px;
    border-bottom: 1px solid var(--border);
  }
  .search-row input { flex: 1; }
  .note-list {
    list-style: none;
    padding: 0;
    margin: 0;
    overflow-y: auto;
    flex: 1;
  }
  li {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    border-bottom: 1px solid var(--border);
  }
  li:hover { background: var(--hover); }
  li.active { background: var(--hover); }
  .title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .mtime { font-size: 12px; color: var(--muted); }
  .empty { color: var(--muted); justify-content: center; padding: 16px; }
  .editor-wrap { flex: 1; display: flex; min-height: 0; }
  .placeholder { color: var(--muted); margin: auto; }
  .err { color: var(--danger); padding: 8px; }
</style>
