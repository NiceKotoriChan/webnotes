<script setup lang="ts">
// 活动栏（第 1 区块）：最左一列图标按钮 —— 仓库、搜索、五个侧栏区块、主题。
import { ref, onMounted, onBeforeUnmount } from 'vue';
import { useRepos } from '../../composables/useRepos';
import { useCurrentRepo, LAST_REPO_KEY } from '../../composables/useCurrentRepo';
import { useTheme } from '../../composables/useTheme';
import { PANELS, type Panel } from '../../lib/panels';
import type { Repo } from '../../api';
import RepoMenu from './RepoMenu.vue';

defineProps<{ view: Panel }>();
const emit = defineEmits<{
  (e: 'set-view', p: Panel): void;
  (e: 'open-search'): void;
}>();

const { repos, create: createRepoApi, rename: renameRepoApi, remove: removeRepoApi } = useRepos();
const { current } = useCurrentRepo();
const { theme, toggle: toggleTheme } = useTheme();

const repoOpen = ref(false);
const errMsg = ref('');
const menuEl = ref<HTMLElement>();
const btnEl = ref<HTMLElement>();

// 点击外部关闭仓库菜单（始终注册，内部按状态判断）
function onPointerDown(e: PointerEvent) {
  if (!repoOpen.value) return;
  const t = e.target as Node;
  if (!menuEl.value?.contains(t) && !btnEl.value?.contains(t)) repoOpen.value = false;
}
onMounted(() => window.addEventListener('pointerdown', onPointerDown));
onBeforeUnmount(() => window.removeEventListener('pointerdown', onPointerDown));

async function createRepo(name: string) {
  errMsg.value = '';
  try {
    const r = await createRepoApi(name);
    current.value = r;
    repoOpen.value = false;
  } catch (e: any) {
    errMsg.value = e.message;
  }
}

function pick(r: Repo) {
  current.value = r;
  repoOpen.value = false;
}

async function rename(r: Repo) {
  const name = prompt('仓库名称', r.name);
  if (name === null || !name.trim() || name.trim() === r.name) return;
  errMsg.value = '';
  try {
    await renameRepoApi(r.id, name.trim());
  } catch (e: any) {
    errMsg.value = e.message;
  }
}

async function remove(r: Repo) {
  if (!confirm(`删除仓库「${r.name}」？所有笔记与附件将一并丢失。`)) return;
  errMsg.value = '';
  try {
    await removeRepoApi(r.id);
    if (current.value?.id === r.id) {
      current.value = null;
      localStorage.removeItem(LAST_REPO_KEY);
    }
  } catch (e: any) {
    errMsg.value = e.message;
  }
}
</script>

<template>
  <nav class="bar">
    <!-- 仓库切换 -->
    <div class="relative">
      <button ref="btnEl" class="icon-btn" :class="{ active: repoOpen }" title="切换仓库" @click="repoOpen = !repoOpen">
        <Icon icon="mdi:folder-multiple-outline" width="20" height="20" />
      </button>

      <div v-if="repoOpen" ref="menuEl" class="popover">
        <RepoMenu
          :repos="repos"
          :current-id="current?.id"
          :err-msg="errMsg"
          @create="createRepo"
          @pick="pick"
          @rename="rename"
          @remove="remove"
        />
      </div>
    </div>

    <!-- 搜索 -->
    <button class="icon-btn" title="搜索 (Ctrl+P)" @click="emit('open-search')">
      <Icon icon="mdi:magnify" width="20" height="20" />
    </button>

    <span class="divider"></span>

    <!-- 五个侧栏区块 -->
    <button
      v-for="p in PANELS"
      :key="p.id"
      class="icon-btn"
      :class="{ active: view === p.id }"
      :title="p.title"
      @click="emit('set-view', p.id)"
    >
      <Icon :icon="'mdi:' + p.icon" width="20" height="20" />
    </button>

    <span class="spacer"></span>

    <!-- 主题切换 -->
    <button class="icon-btn" :title="theme === 'dark' ? '切换为亮色模式' : '切换为暗色模式'" @click="toggleTheme">
      <Icon :icon="theme === 'dark' ? 'mdi:white-balance-sunny' : 'mdi:weather-night'" width="20" height="20" />
    </button>
  </nav>
</template>

<style scoped>
.bar {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  width: 48px;
  flex-shrink: 0;
  padding: 8px 0;
  background: var(--canvas-subtle);
  border-right: 1px solid var(--border-muted);
}
.relative {
  position: relative;
}
.spacer {
  flex: 1;
}
.divider {
  width: 24px;
  height: 1px;
  margin: 4px 0;
  background: var(--border-muted);
}
.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  color: var(--fg-muted);
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}
.icon-btn:hover {
  color: var(--fg-default);
  background: var(--btn-hover-bg);
}
.icon-btn.active {
  color: var(--fg-default);
  background: var(--btn-active-bg);
}
.popover {
  position: absolute;
  top: 0;
  left: calc(100% + 4px);
  width: 280px;
  z-index: 50;
  background: var(--canvas-overlay);
  border: 1px solid var(--border-default);
  border-radius: 12px;
  box-shadow: var(--shadow-floating);
  overflow: hidden;
}
</style>
