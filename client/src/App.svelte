<script lang="ts">
  // 无入口页面：打开即主界面。仓库通过活动栏的仓库图标悬浮菜单切换
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { reposStore, currentRepo, LAST_REPO_KEY } from './stores';
  import Workspace from './components/Workspace.svelte';

  // 启动：加载仓库列表，并恢复上次打开的仓库（已被删除则停在未选择状态）
  onMount(async () => {
    await reposStore.load();
    const rid = localStorage.getItem(LAST_REPO_KEY);
    if (rid) {
      const r = get(reposStore).find((x) => x.id === rid);
      if (r) currentRepo.set(r);
    }
  });

  // 仓库改名后同步当前仓库对象；当前仓库被删除时回到未选择状态
  $effect(() => {
    const r = $currentRepo;
    if (!r) return;
    const fresh = $reposStore.find((x) => x.id === r.id);
    if (!fresh) currentRepo.set(null);
    else if (fresh !== r) currentRepo.set(fresh);
  });
</script>

<main>
  {#key $currentRepo?.id ?? 'none'}
    <Workspace repo={$currentRepo} />
  {/key}
</main>

<style>
  main { height: 100%; }
</style>
