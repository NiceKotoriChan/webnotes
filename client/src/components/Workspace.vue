<script setup lang="ts">
// 主工作区：活动栏 | 侧栏（五个区块之一）| 主区（总览或编辑器）。
// 区块清单在 lib/panels.ts，加区块只改那里 + 这里挂一个组件。
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue';
import { useNotes } from '../composables/useNotes';
import { useTags } from '../composables/useTags';
import { lastNoteKey } from '../composables/useCurrentRepo';
import * as api from '../api';
import type { Note, NotePatch, Repo } from '../api';
import { buildTree } from '../lib/tree';
import { initialPanel, rememberPanel, type Panel } from '../lib/panels';
import ActivityBar from './ActivityBar/ActivityBar.vue';
import TreePanel from './SideBar/TreePanel.vue';
import CalendarPanel from './SideBar/CalendarPanel.vue';
import TagsPanel from './SideBar/TagsPanel.vue';
import AssetPanel from './SideBar/AssetPanel.vue';
import TrashPanel from './SideBar/TrashPanel.vue';
import OverviewPanel from './Overview/OverviewPanel.vue';
import NoteEditor from './Editor/NoteEditor.vue';
import SearchPanel from './ActivityBar/SearchPanel.vue';

const props = defineProps<{ repo: Repo | null }>();
const repoId = computed(() => props.repo?.id ?? null);

const notes = useNotes();
const tags = useTags();

const selected = ref<Note | null>(null);
const errMsg = ref('');
const loading = ref(false);
const collapsed = ref<Set<string>>(new Set());
const view = ref<Panel>(initialPanel());
const searchOpen = ref(false);

const tree = computed(() => buildTree(notes.notes.value));

function setView(p: Panel) {
  view.value = p;
  rememberPanel(p);
}

// 仓库切换时加载笔记 + 标签
watch(
  repoId,
  (rid) => {
    if (!rid) return;
    loading.value = true;
    notes.load(rid).finally(() => (loading.value = false));
    tags.load(rid);
  },
  { immediate: true },
);

// notes 刷新后，把 selected 指向新对象
watch(
  () => notes.notes.value,
  (list) => {
    if (!selected.value) return;
    const fresh = list.find((x) => x.id === selected.value!.id);
    if (fresh && fresh !== selected.value) selected.value = fresh;
  },
);

// 首次加载完笔记后，恢复上次打开的笔记
let didRestore = false;
watch(
  () => notes.notes.value,
  (list) => {
    const rid = repoId.value;
    if (didRestore || list.length === 0 || !rid) return;
    didRestore = true;
    const nid = localStorage.getItem(lastNoteKey(rid));
    const found = nid ? list.find((n) => n.id === nid) : undefined;
    if (found) selected.value = found;
  },
);

async function guarded(fn: () => Promise<unknown>) {
  errMsg.value = '';
  try {
    await fn();
    return true;
  } catch (e: any) {
    errMsg.value = e.message;
    return false;
  }
}

async function createRoot() {
  return create(null);
}
async function createChild(parent: Note) {
  const next = new Set(collapsed.value);
  next.delete(parent.id);
  collapsed.value = next;
  return create(parent.id);
}
async function create(parent_id: string | null) {
  if (!repoId.value) return;
  await guarded(async () => {
    const n = await notes.create(parent_id);
    if (n) selectNote(n);
  });
}

function selectNote(n: Note) {
  selected.value = n;
  const rid = repoId.value;
  if (rid) localStorage.setItem(lastNoteKey(rid), n.id);
}

async function remove(n: Note) {
  await guarded(async () => {
    await notes.remove(n.id);
    if (selected.value?.id === n.id) {
      selected.value = null;
      const rid = repoId.value;
      if (rid) localStorage.removeItem(lastNoteKey(rid));
    }
  });
}

async function patchNote(n: Note, patch: NotePatch) {
  await guarded(() => notes.update(n.id, patch));
}

const renameNote = (n: Note, title: string) => patchNote(n, { title });
const moveNote = (draggedId: string, targetId: string) =>
  guarded(async () => {
    await api.updateNote(repoId.value!, draggedId, { parent_id: targetId });
    const next = new Set(collapsed.value);
    next.delete(targetId);
    collapsed.value = next;
    await notes.refresh();
  });
const setNoteIcon = (n: Note, icon: string | null) => patchNote(n, { icon });

// —— 标签：都作用在「当前笔记」或「全体笔记」上，标签自身不是实体 ——
async function toggleTag(name: string) {
  const sel = selected.value;
  if (!sel || !repoId.value) return;
  const cur = sel.tags ?? [];
  const next = cur.includes(name) ? cur.filter((t) => t !== name) : [...cur, name];
  await guarded(async () => {
    await notes.update(sel.id, { tags: next });
    // 标签是派生的：勾选可能让这个标签第一次出现，也可能让它彻底消失，所以每次都重拉
    tags.load(repoId.value!);
  });
}

async function createTag(name: string) {
  const sel = selected.value;
  if (!sel) return;
  const cur = sel.tags ?? [];
  if (cur.includes(name)) return;
  await toggleTag(name);
}

async function renameTag(from: string) {
  const rid = repoId.value;
  if (!rid) return;
  const to = prompt(`把标签「${from}」改名为`, from);
  if (to === null) return;
  const name = to.trim().replace(/^#/, '');
  if (!name || name === from) return;
  await guarded(async () => {
    await tags.rename(rid, from, name);
    await notes.refresh();
  });
}

async function deleteTag(name: string) {
  const rid = repoId.value;
  if (!rid) return;
  if (!confirm(`把标签「${name}」从所有笔记上移除？`)) return;
  await guarded(async () => {
    await tags.remove(rid, name);
    await notes.refresh();
  });
}

function toggle(n: Note) {
  const next = new Set(collapsed.value);
  if (next.has(n.id)) next.delete(n.id);
  else next.add(n.id);
  collapsed.value = next;
}

// Ctrl+P 打开搜索
function onKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key === 'p') {
    e.preventDefault();
    searchOpen.value = true;
  }
}
onMounted(() => window.addEventListener('keydown', onKeydown));
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown));
</script>

<template>
  <div class="workspace">
    <ActivityBar :view="view" @set-view="setView" @open-search="searchOpen = true" />

    <aside class="sidebar">
      <p v-if="errMsg" class="err">{{ errMsg }}</p>

      <TreePanel
        v-if="view === 'tree'"
        :repo-name="repo?.name ?? null"
        :notes="tree"
        :selected-id="selected?.id ?? null"
        :collapsed="collapsed"
        :on-select="selectNote"
        :on-toggle="toggle"
        :on-add-child="createChild"
        :on-delete="remove"
        :on-rename="renameNote"
        :on-move="moveNote"
        :on-set-icon="setNoteIcon"
        :on-create-root="createRoot"
        :loading="loading"
        :repo-id="repoId"
      />

      <CalendarPanel
        v-else-if="view === 'calendar'"
        :notes="notes.notes.value"
        :selected-id="selected?.id ?? null"
        :on-select="selectNote"
      />

      <TagsPanel
        v-else-if="view === 'tags'"
        :repo-id="repoId"
        :selected="selected"
        :on-toggle="toggleTag"
        :on-create="createTag"
        :on-rename="renameTag"
        :on-delete="deleteTag"
      />

      <AssetPanel v-else-if="view === 'assets'" :repo-id="repoId" />

      <TrashPanel v-else :repo-id="repoId" @changed="notes.refresh()" />
    </aside>

    <section class="main">
      <NoteEditor
        v-if="selected && repoId"
        :key="selected.id"
        :repo="repoId"
        :note="selected"
        :on-saved="notes.refresh"
      />
      <OverviewPanel
        v-else
        :repo="repo"
        :notes="notes.notes.value"
        :tags="tags.tags.value"
        :on-select="selectNote"
      />
    </section>
  </div>

  <SearchPanel
    v-if="searchOpen"
    :on-select-note="(n) => { selectNote(n); searchOpen = false; }"
    :on-close="() => (searchOpen = false)"
  />
</template>

<style scoped>
.workspace {
  display: flex;
  height: 100%;
  overflow: hidden;
}
.sidebar {
  display: flex;
  flex-direction: column;
  width: 280px;
  flex-shrink: 0;
  height: 100%;
  background: var(--canvas-subtle);
  border-right: 1px solid var(--border-muted);
}
.sidebar .err {
  margin: 0;
  padding: 8px 12px;
  font-size: 12px;
  color: var(--danger-fg);
  border-bottom: 1px solid var(--border-muted);
}
.main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  height: 100%;
  background: var(--canvas-default);
}
</style>
