<script lang="ts">
  // 活动栏（VSC 风格）：最左侧窄竖条，放主题切换 + 仓库图标（悬浮菜单增删改）
  import { reposStore, currentRepo, themeStore, LAST_REPO_KEY } from '../stores';
  import type { Repo } from '../api';

  let repoOpen = $state(false);
  let newName = $state('');
  let errMsg = $state('');
  let menuEl = $state<HTMLElement>();
  let btnEl = $state<HTMLElement>();

  // 菜单展开时，点击外部关闭
  $effect(() => {
    if (!repoOpen) return;
    const onDown = (e: MouseEvent) => {
      const t = e.target as Node;
      if (!menuEl?.contains(t) && !btnEl?.contains(t)) repoOpen = false;
    };
    window.addEventListener('pointerdown', onDown);
    return () => window.removeEventListener('pointerdown', onDown);
  });

  function toggleTheme() {
    themeStore.update((t) => (t === 'dark' ? 'light' : 'dark'));
  }

  async function createRepo() {
    const name = newName.trim();
    if (!name) return;
    errMsg = '';
    try {
      const r = await reposStore.create(name);
      newName = '';
      currentRepo.set(r);
      repoOpen = false;
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  function pick(r: Repo) {
    currentRepo.set(r);
    repoOpen = false;
  }

  async function rename(r: Repo) {
    const name = prompt('仓库名称', r.name);
    if (name === null || !name.trim() || name.trim() === r.name) return;
    errMsg = '';
    try {
      await reposStore.rename(r.id, name.trim());
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  async function remove(r: Repo) {
    if (!confirm(`删除仓库「${r.name}」？所有笔记与附件将一并丢失。`)) return;
    errMsg = '';
    try {
      await reposStore.remove(r.id);
      if ($currentRepo?.id === r.id) {
        currentRepo.set(null);
        localStorage.removeItem(LAST_REPO_KEY);
      }
    } catch (e: any) {
      errMsg = e.message;
    }
  }
</script>

<nav class="activity-bar">
  <button class="icon-btn" title={$themeStore === 'dark' ? '切换为亮色模式' : '切换为暗色模式'} onclick={toggleTheme}>
    {#if $themeStore === 'dark'}
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
        stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4"/><path
        d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/></svg>
    {:else}
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
        stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/></svg>
    {/if}
  </button>

  <div class="repo-wrap">
    <button class="icon-btn" bind:this={btnEl} title="打开仓库"
      onclick={() => (repoOpen = !repoOpen)}>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
        stroke-linecap="round" stroke-linejoin="round"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></svg>
    </button>

    {#if repoOpen}
      <div class="popover" bind:this={menuEl}>
        <form onsubmit={(e) => { e.preventDefault(); createRepo(); }}>
          <input bind:value={newName} placeholder="新建仓库，输入名称" />
          <button class="primary small" type="submit">新建</button>
        </form>
        {#if errMsg}<p class="err">{errMsg}</p>{/if}

        <ul>
          {#each $reposStore as r (r.id)}
            <li class:active={$currentRepo?.id === r.id}>
              <button class="name" onclick={() => pick(r)} title={r.name}>
                <span class="dot"></span>{r.name}
              </button>
              <span class="ops">
                <button class="op" title="改名" onclick={() => rename(r)}>
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
                    stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path
                    d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z"/></svg>
                </button>
                <button class="op danger" title="删除" onclick={() => remove(r)}>
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
                    stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6"/></svg>
                </button>
              </span>
            </li>
          {:else}
            <li class="empty">还没有仓库</li>
          {/each}
        </ul>
      </div>
    {/if}
  </div>
</nav>

<style>
  .activity-bar {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 8px 0;
    width: 48px;
    flex-shrink: 0;
    background: var(--bg);
    border-right: 1px solid var(--border);
  }
  .icon-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    padding: 0;
    color: var(--muted);
    background: transparent;
    border: none;
    border-radius: 6px;
    cursor: pointer;
  }
  .icon-btn:hover { color: var(--fg); background: var(--hover); }
  .repo-wrap { position: relative; }
  .popover {
    position: absolute;
    top: 0;
    left: calc(100% + 4px);
    width: 260px;
    z-index: 50;
    padding: 8px;
    background: var(--overlay);
    border: 1px solid var(--border);
    border-radius: 8px;
    box-shadow: var(--shadow-pop);
  }
  form { display: flex; gap: 4px; margin-bottom: 6px; }
  form input { flex: 1; min-width: 0; }
  ul { list-style: none; margin: 0; padding: 0; max-height: 300px; overflow-y: auto; }
  li {
    display: flex;
    align-items: center;
    border-radius: 4px;
  }
  li:hover, li.active { background: var(--hover); }
  .name {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 6px;
    background: none;
    border: none;
    padding: 7px 8px;
    text-align: left;
    cursor: pointer;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--fg);
  }
  .dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: transparent;
    flex-shrink: 0;
  }
  li.active .dot { background: var(--accent); }
  li.active .name { font-weight: 600; }
  .ops { display: flex; gap: 2px; padding-right: 4px; visibility: hidden; }
  li:hover .ops { visibility: visible; }
  .op {
    border: none;
    background: none;
    padding: 4px;
    display: inline-flex;
    color: var(--muted);
    border-radius: 4px;
  }
  .op:hover { background: var(--hover-strong); color: var(--fg); }
  .op.danger { color: var(--danger); }
  .op.danger:hover { background: var(--danger-soft); color: var(--danger); }
  .empty { color: var(--muted); justify-content: center; padding: 12px; }
  .err { color: var(--danger); padding: 4px 8px; margin: 0; font-size: 12px; }
</style>
