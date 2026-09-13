<script setup lang="ts">
import { onMounted, watch } from 'vue';
import { useRepos } from './composables/useRepos';
import { useCurrentRepo, LAST_REPO_KEY } from './composables/useCurrentRepo';
import Workspace from './components/Workspace.vue';

const { repos, load } = useRepos();
const { current } = useCurrentRepo();

// 启动：加载仓库列表，并恢复上次打开的仓库（已被删除则停在未选择状态）
onMounted(async () => {
  await load();
  const rid = localStorage.getItem(LAST_REPO_KEY);
  if (rid) {
    const r = repos.value.find((x) => x.id === rid);
    if (r) current.value = r;
  }
});

// 仓库改名后同步当前仓库对象；当前仓库被删除时回到未选择状态
watch([current, repos], () => {
  const r = current.value;
  if (!r) return;
  const fresh = repos.value.find((x) => x.id === r.id);
  if (!fresh) current.value = null;
  else if (fresh !== r) current.value = fresh;
});
</script>

<template>
  <main class="h-full">
    <Workspace :key="current?.id ?? 'none'" :repo="current" />
  </main>
</template>
