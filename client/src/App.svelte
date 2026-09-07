<script lang="ts">
  import { reposStore, currentRepo, notesStore } from './stores';
  import ReposView from './components/ReposView.svelte';
  import NotesView from './components/NotesView.svelte';

  let view: 'repos' | 'notes' = 'repos';

  currentRepo.subscribe((r) => {
    view = r ? 'notes' : 'repos';
    if (r) notesStore.load(r);
  });

  function refresh() {
    reposStore.load();
  }

  refresh();
</script>

<main>
  <header>
    <h1>WebNotes</h1>
    {#if $currentRepo}
      <nav>
        <a href="#" on:click|preventDefault={() => currentRepo.set(null)}>
          ← 仓库列表
        </a>
        <span> / {$currentRepo}</span>
      </nav>
    {/if}
  </header>

  {#if view === 'repos'}
    <ReposView />
  {:else if $currentRepo}
    <NotesView repo={$currentRepo} />
  {/if}
</main>

<style>
  main {
    display: flex;
    flex-direction: column;
    height: 100%;
  }
  header {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 8px 16px;
    border-bottom: 1px solid var(--border);
    background: var(--bg);
  }
  header h1 { font-size: 16px; margin: 0; }
  nav { display: flex; gap: 4px; align-items: center; }
</style>
