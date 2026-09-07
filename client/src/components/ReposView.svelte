<script lang="ts">
  import { reposStore } from '../stores';
  import { currentRepo } from '../stores';

  let newId = '';
  let creating = false;
  let errMsg = '';

  async function create() {
    if (!newId.trim()) return;
    creating = true;
    errMsg = '';
    try {
      await reposStore.create(newId.trim());
      newId = '';
    } catch (e: any) {
      errMsg = e.message;
    } finally {
      creating = false;
    }
  }

  async function remove(id: string) {
    if (!confirm(`删除仓库 "${id}"？所有笔记将一并丢失。`)) return;
    try {
      await reposStore.remove(id);
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  function open(id: string) {
    currentRepo.set(id);
  }
</script>

<section class="page">
  <h2>仓库</h2>

  <form on:submit|preventDefault={create}>
    <input bind:value={newId} placeholder="新仓库 ID（如 work、linux）" />
    <button type="submit" class="primary" disabled={creating}>新建</button>
  </form>

  {#if errMsg}<p class="err">{errMsg}</p>{/if}

  <ul class="repo-list">
    {#each $reposStore as r (r.id)}
      <li>
        <a href="#" on:click|preventDefault={() => open(r.id)}>{r.id}</a>
        <button class="danger" on:click={() => remove(r.id)}>删除</button>
      </li>
    {:else}
      <li class="empty">还没有仓库</li>
    {/each}
  </ul>
</section>

<style>
  .page { padding: 16px; }
  form { display: flex; gap: 8px; margin-bottom: 16px; }
  .repo-list { list-style: none; padding: 0; }
  li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    border: 1px solid var(--border);
    border-radius: 4px;
    margin-bottom: 4px;
  }
  li a { font-size: 15px; }
  .empty { color: var(--muted); justify-content: center; border-style: dashed; }
  .err { color: var(--danger); margin: 8px 0; }
</style>
