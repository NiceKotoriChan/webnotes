<script setup lang="ts">
// 全局搜索面板：VSC 命令面板风格。笔记走服务端 FTS，标签/附件客户端匹配。
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue';
import * as api from '../../api';
import type { Note, Tag, AssetMeta } from '../../api';
import { useCurrentRepo } from '../../composables/useCurrentRepo';
import { useTags } from '../../composables/useTags';
import { useAssets } from '../../composables/useAssets';

const props = defineProps<{
  onSelectNote: (n: Note) => void;
  onClose: () => void;
}>();

const { current } = useCurrentRepo();
const { tags, load: loadTags } = useTags();
const { assets, load: loadAssets } = useAssets();

const repoId = computed(() => current.value?.id);

type Result = { type: 'note' | 'tag' | 'asset'; item: Note | Tag | AssetMeta; match: string };

const query = ref('');
const inputEl = ref<HTMLInputElement>();
const results = ref<Result[]>([]);
const selectedIndex = ref(0);
const loading = ref(false);
const tagFilter = ref<Tag | null>(null);

let seq = 0;
let timer: ReturnType<typeof setTimeout> | undefined;

onMounted(() => {
  inputEl.value?.focus();
  if (repoId.value) {
    loadTags(repoId.value);
    loadAssets(repoId.value);
  }
  window.addEventListener('keydown', onKeydown);
});
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown));

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') props.onClose();
  else if (e.key === 'ArrowDown') {
    e.preventDefault();
    selectedIndex.value = Math.min(selectedIndex.value + 1, results.value.length - 1);
  } else if (e.key === 'ArrowUp') {
    e.preventDefault();
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0);
  } else if (e.key === 'Enter' && results.value[selectedIndex.value]) {
    select(results.value[selectedIndex.value]);
  }
}

async function load() {
  const rid = repoId.value;
  const s = ++seq;
  if (!rid) {
    results.value = [];
    loading.value = false;
    return;
  }
  loading.value = true;
  const r: Result[] = [];

  if (tagFilter.value) {
    try {
      const notes = await api.listNotes(rid, { tag_id: tagFilter.value.id });
      if (s !== seq) return;
      for (const n of notes) r.push({ type: 'note', item: n, match: n.title || '(无标题)' });
    } catch {
      if (s !== seq) return;
    }
  } else {
    const q = query.value.trim();
    if (q.length < 3) {
      if (s === seq) {
        results.value = [];
        loading.value = false;
      }
      return;
    }
    try {
      const notes = await api.listNotes(rid, { q });
      if (s !== seq) return;
      for (const n of notes) r.push({ type: 'note', item: n, match: n.title || '(无标题)' });
    } catch {
      if (s !== seq) return;
    }
    const lower = q.toLowerCase();
    for (const t of tags.value) if (t.name.toLowerCase().includes(lower)) r.push({ type: 'tag', item: t, match: '#' + t.name });
    for (const a of assets.value) if (a.name.toLowerCase().includes(lower)) r.push({ type: 'asset', item: a, match: a.name });
  }

  results.value = r.slice(0, 30);
  selectedIndex.value = 0;
  loading.value = false;
}

watch(query, () => {
  if (tagFilter.value) return;
  clearTimeout(timer);
  const q = query.value.trim();
  if (q.length < 3) {
    seq++;
    results.value = [];
    loading.value = false;
    return;
  }
  loading.value = true;
  timer = setTimeout(load, 200);
});

watch(tagFilter, () => load());

function select(r: Result) {
  if (r.type === 'note') {
    props.onSelectNote(r.item as Note);
    props.onClose();
    return;
  }
  if (r.type === 'tag') {
    tagFilter.value = r.item as Tag;
    query.value = '';
    return;
  }
  if (r.type === 'asset') {
    const rid = repoId.value;
    if (rid) {
      navigator.clipboard?.writeText(api.assetURL(rid, (r.item as AssetMeta).id)).catch(() => {});
    }
    props.onClose();
  }
}

function clearTagFilter() {
  tagFilter.value = null;
  query.value = '';
  results.value = [];
}

function onBackdropClick(e: MouseEvent) {
  if (e.target === e.currentTarget) props.onClose();
}
</script>

<template>
  <div class="backdrop" role="presentation" @click="onBackdropClick">
    <div class="panel" role="dialog" aria-modal="true" aria-label="全局搜索">
      <div class="input-wrap">
        <Icon class="search-icon" icon="mdi:magnify" width="16" height="16" />
        <input ref="inputEl" v-model="query" placeholder="搜索笔记、标签、附件..." />
      </div>

      <div v-if="tagFilter" class="tag-filter">
        <span>标签 #{{ tagFilter.name }} 的笔记</span>
        <button class="link" @click="clearTagFilter">清除</button>
      </div>

      <div v-if="loading" class="status">搜索中...</div>
      <div v-else-if="!tagFilter && query.trim().length < 3" class="status">输入至少 3 个字符开始搜索</div>
      <div v-else-if="results.length === 0" class="status">没有找到结果</div>
      <ul v-else class="results">
        <li v-for="(r, i) in results" :key="r.type + (r.item as any).id" :class="{ selected: i === selectedIndex }">
          <button class="result-btn" @mouseenter="selectedIndex = i" @click="select(r)">
            <span class="type"><Icon :icon="r.type === 'note' ? 'mdi:file-document-outline' : r.type === 'tag' ? 'mdi:tag' : 'mdi:paperclip'" width="16" height="16" /></span>
            <span class="match">{{ r.match }}</span>
          </button>
        </li>
      </ul>

      <div class="footer">
        <span class="hint">↑↓ 导航 · Enter 选择 · Esc 关闭</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.backdrop {
  position: fixed;
  inset: 0;
  display: flex;
  justify-content: center;
  padding-top: 15vh;
  background: rgba(0, 0, 0, 0.5);
  z-index: 100;
}
.panel {
  width: 560px;
  max-width: 90vw;
  background: var(--canvas-overlay);
  border: 1px solid var(--border-default);
  border-radius: 12px;
  box-shadow: var(--shadow-floating);
  overflow: hidden;
}
.input-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-muted);
}
.search-icon {
  color: var(--fg-muted);
  flex-shrink: 0;
}
.input-wrap input {
  flex: 1;
  border: none;
  padding: 0;
  background: transparent;
  font-size: 16px;
}
.input-wrap input:focus {
  box-shadow: none;
}
.tag-filter {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  border-bottom: 1px solid var(--border-muted);
  background: var(--accent-subtle);
  font-size: 13px;
}
.status {
  padding: 24px;
  text-align: center;
  color: var(--fg-muted);
  font-size: 14px;
}
.results {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 320px;
  overflow-y: auto;
}
.results li {
  display: flex;
  align-items: center;
}
.result-btn {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
  padding: 10px 16px;
  background: none;
  border: none;
  text-align: left;
  cursor: pointer;
  color: var(--fg-default);
  font: inherit;
}
.results li:hover .result-btn,
.results li.selected .result-btn {
  background: var(--accent-subtle);
}
.results li.selected {
  outline: 2px solid var(--accent-emphasis);
  outline-offset: -2px;
}
.type {
  font-size: 16px;
}
.match {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
}
.footer {
  padding: 8px 16px;
  border-top: 1px solid var(--border-muted);
  background: var(--canvas-subtle);
}
.hint {
  font-size: 12px;
  color: var(--fg-muted);
}
</style>
