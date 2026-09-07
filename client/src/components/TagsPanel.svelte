<script lang="ts">
  import { onMount } from 'svelte';
  import * as api from '../api';
  import type { Tag } from '../api';

  export let repo: string;
  export let noteId: string;

  let allTags: Tag[] = [];
  let linkedTags: Tag[] = [];
  let newName = '';
  let errMsg = '';

  async function loadAll() {
    try {
      allTags = (await api.listTags(repo)) || [];
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  async function loadLinked() {
    try {
      linkedTags = (await api.listNoteTags(repo, noteId)) || [];
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  async function refresh() {
    await Promise.all([loadAll(), loadLinked()]);
  }

  onMount(refresh);

  function isLinked(id: string) {
    return linkedTags.some(t => t.id === id);
  }

  async function toggle(t: Tag) {
    try {
      if (isLinked(t.id)) {
        await api.removeNoteTag(repo, noteId, t.id);
      } else {
        await api.addNoteTag(repo, noteId, t.id);
      }
      await loadLinked();
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  async function create() {
    if (!newName.trim()) return;
    try {
      await api.createTag(repo, newName.trim());
      newName = '';
      await loadAll();
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  async function remove(id: string) {
    if (!confirm('删除该标签？')) return;
    try {
      await api.deleteTag(repo, id);
      await refresh();
    } catch (e: any) {
      errMsg = e.message;
    }
  }
</script>

<section>
  <h3>标签</h3>
  {#if errMsg}<p class="err">{errMsg}</p>{/if}

  <form on:submit|preventDefault={create}>
    <input bind:value={newName} placeholder="新标签…" />
    <button class="primary">添加</button>
  </form>

  <ul class="tags">
    {#each allTags as t (t.id)}
      <li class:linked={isLinked(t.id)}>
        <a href="#" on:click|preventDefault={() => toggle(t)}>{t.name}</a>
        <button class="danger" on:click={() => remove(t.id)}>×</button>
      </li>
    {:else}
      <li class="empty">尚无标签</li>
    {/each}
  </ul>
</section>

<style>
  section {
    border-top: 1px solid var(--border);
    padding: 8px;
  }
  h3 { margin: 0 0 8px 0; font-size: 14px; }
  form { display: flex; gap: 4px; margin-bottom: 8px; }
  form input { flex: 1; }
  .tags { list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 2px; }
  li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 4px 8px;
    border-radius: 4px;
  }
  li.linked { background: var(--hover); }
  li:hover { background: var(--hover); }
  .empty { color: var(--muted); justify-content: center; }
  .err { color: var(--danger); }
</style>
