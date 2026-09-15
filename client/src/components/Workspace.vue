<script setup lang="ts">
// 主工作区：ActivityBar | SideBar（TreePanel + TagsPanel 或 AssetPanel）| Editor
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue';
import { useNotes } from '../composables/useNotes';
import { useTags } from '../composables/useTags';
import { lastNoteKey } from '../composables/useCurrentRepo';
import * as api from '../api';
import type { Note, Repo, Tag } from '../api';
import { buildTree } from '../lib/tree';
import ActivityBar from './ActivityBar/ActivityBar.vue';
import TreePanel from './SideBar/TreePanel.vue';
import TagsPanel from './SideBar/TagsPanel.vue';
import AssetPanel from './SideBar/AssetPanel.vue';
import TrashPanel from './SideBar/TrashPanel.vue';
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
const linkedIds = ref<Set<string>>(new Set());
const activeView = ref<'explorer' | 'assets' | 'trash'>('explorer');
const searchOpen = ref(false);

const tree = computed(() => buildTree(notes.notes.value));

// 仓库切换时加载笔记 + 标签
watch(repoId, (rid) => {
  if (!rid) return;
  loading.value = true;
  notes.load(rid).finally(() => (loading.value = false));
  tags.load(rid);
}, { immediate: true });

// 当前笔记的标签关联
watch([repoId, () => selected.value?.id], () => {
  const rid = repoId.value;
  const nid = selected.value?.id;
  if (!rid || !nid) {
    linkedIds.value = new Set();
    return;
  }
  api.listNoteTags(rid, nid)
    .then((ts) => (linkedIds.value = new Set(ts.map((t) => t.id))))
    .catch(() => {});
});

// notes 刷新后，把 selected 指向新对象
watch(() => notes.notes.value, (list) => {
  if (!selected.value) return;
  const id = selected.value.id;
  const fresh = list.find((x) => x.id === id);
  if (fresh && fresh !== selected.value) selected.value = fresh;
});

// 首次加载完笔记后，恢复上次打开的笔记
let didRestore = false;
watch(() => notes.notes.value, (list) => {
  const rid = repoId.value;
  if (didRestore || list.length === 0 || !rid) return;
  didRestore = true;
  const nid = localStorage.getItem(lastNoteKey(rid));
  const found = nid ? list.find((n) => n.id === nid) : undefined;
  if (found) selected.value = found;
});

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
  const rid = repoId.value;
  if (!rid) return;
  errMsg.value = '';
  try {
    const n = await notes.create(parent_id);
    if (n) selectNote(n);
  } catch (e: any) {
    errMsg.value = e.message;
  }
}

function selectNote(n: Note) {
  selected.value = n;
  const rid = repoId.value;
  if (rid) localStorage.setItem(lastNoteKey(rid), n.id);
}

async function remove(n: Note) {
  try {
    await notes.remove(n.id);
    if (selected.value?.id === n.id) {
      selected.value = null;
      const rid = repoId.value;
      if (rid) localStorage.removeItem(lastNoteKey(rid));
    }
  } catch (e: any) {
    errMsg.value = e.message;
  }
}

async function renameNote(n: Note, title: string) {
  const rid = repoId.value;
  if (!rid) return;
  errMsg.value = '';
  try {
    await api.updateNote(rid, n.id, { title, data: n.data });
    await notes.refresh();
  } catch (e: any) {
    errMsg.value = e.message;
  }
}

async function moveNote(draggedId: string, targetId: string) {
  const rid = repoId.value;
  if (!rid) return;
  errMsg.value = '';
  try {
    await api.moveNote(rid, draggedId, targetId);
    const next = new Set(collapsed.value);
    next.delete(targetId);
    collapsed.value = next;
    await notes.refresh();
  } catch (e: any) {
    errMsg.value = e.message;
    await notes.refresh();
  }
}

async function setNoteIcon(n: Note, icon: string | null) {
  const rid = repoId.value;
  if (!rid) return;
  errMsg.value = '';
  try {
    await api.setNoteIcon(rid, n.id, icon);
    await notes.refresh();
  } catch (e: any) {
    errMsg.value = e.message;
  }
}

function toggle(n: Note) {
  const next = new Set(collapsed.value);
  next.has(n.id) ? next.delete(n.id) : next.add(n.id);
  collapsed.value = next;
}

async function addTag(name: string) {
  const rid = repoId.value;
  if (!rid) return;
  errMsg.value = '';
  try {
    await tags.create(rid, name);
  } catch (e: any) {
    errMsg.value = e.message;
  }
}

async function deleteTag(t: Tag) {
  const rid = repoId.value;
  if (!rid) return;
  if (!confirm(`删除标签「${t.name}」？`)) return;
  try {
    await tags.remove(rid, t.id);
  } catch (e: any) {
    errMsg.value = e.message;
  }
}

async function renameTag(t: Tag) {
  const name = prompt('标签名', t.name);
  if (name === null || !name.trim() || name.trim() === t.name) return;
  const rid = repoId.value;
  if (!rid) return;
  errMsg.value = '';
  try {
    await tags.rename(rid, t.id, name.trim());
  } catch (e: any) {
    errMsg.value = e.message;
  }
}

async function toggleLink(t: Tag) {
  const rid = repoId.value;
  const sel = selected.value;
  if (!rid || !sel) return;
  errMsg.value = '';
  try {
    if (linkedIds.value.has(t.id)) {
      await api.removeNoteTag(rid, sel.id, t.id);
      const next = new Set(linkedIds.value);
      next.delete(t.id);
      linkedIds.value = next;
    } else {
      await api.addNoteTag(rid, sel.id, t.id);
      const next = new Set(linkedIds.value);
      next.add(t.id);
      linkedIds.value = next;
    }
  } catch (e: any) {
    errMsg.value = e.message;
  }
}

function toggleAssets() {
  activeView.value = activeView.value === 'assets' ? 'explorer' : 'assets';
}

function toggleTrash() {
  activeView.value = activeView.value === 'trash' ? 'explorer' : 'trash';
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
    <ActivityBar
      :active-view="activeView"
      @open-search="searchOpen = true"
      @toggle-assets="toggleAssets"
      @toggle-trash="toggleTrash"
    />

    <aside class="sidebar">
      <template v-if="activeView === 'explorer'">
        <TreePanel
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
        <TagsPanel
          :repo-id="repoId"
          :selected="selected"
          :linked-ids="linkedIds"
          :on-link="toggleLink"
          :on-create="addTag"
          :on-rename="renameTag"
          :on-delete="deleteTag"
        />
      </template>
      <AssetPanel v-else-if="activeView === 'assets'" :repo-id="repoId" />
      <TrashPanel v-else :repo-id="repoId" @changed="notes.refresh()" />
    </aside>

    <section class="editor">
      <NoteEditor
        v-if="selected && repoId"
        :key="selected.id"
        :repo="repoId || ''"
        :note="selected"
        @saved="notes.refresh()"
      />
      <div v-else-if="!repoId" class="placeholder">
        <p>还没有打开仓库</p>
        <p class="hint">点击左侧仓库图标，新建或打开一个仓库开始</p>
      </div>
      <div v-else class="placeholder">选择左侧笔记，或点击「新建」</div>
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
.editor {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  height: 100%;
  background: var(--canvas-default);
}
.placeholder {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--fg-muted);
  gap: 8px;
}
.placeholder p {
  margin: 0;
}
.placeholder .hint {
  font-size: 13px;
}
</style>
