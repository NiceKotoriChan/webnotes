<script setup lang="ts">
// 总览（主区的默认页）。没选中笔记时占住编辑器位置，见 docs/ui.md 的 overview。
// 具体内容「暂时不确定」，先给最实用的三块：统计、最近改动、标签。
import { computed } from 'vue';
import type { Note, Repo } from '../../api';

const props = defineProps<{
  repo: Repo | null;
  notes: Note[];
  tags: string[];
  onSelect: (n: Note) => void;
}>();

const recentlyUpdated = computed(() =>
  [...props.notes].sort((a, b) => b.updated_at - a.updated_at).slice(0, 8),
);
const recentlyCreated = computed(() =>
  [...props.notes].sort((a, b) => b.created_at - a.created_at).slice(0, 5),
);
const roots = computed(() => props.notes.filter((n) => !n.parent_id).length);
const tagged = computed(() => props.notes.filter((n) => n.tags?.length).length);

function fmt(ts: number) {
  return new Date(ts).toLocaleString('zh-CN', { hour12: false, month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
}
</script>

<template>
  <div class="overview">
    <template v-if="!repo">
      <div class="hero">
        <Icon icon="mdi:notebook-outline" width="48" height="48" />
        <p class="big">还没有打开仓库</p>
        <p class="hint">点击左上角的仓库图标，新建或打开一个仓库开始</p>
      </div>
    </template>

    <template v-else>
      <h1 class="repo-name">{{ repo.name }}</h1>

      <div class="stats">
        <div class="stat"><span class="num">{{ notes.length }}</span><span class="label">笔记</span></div>
        <div class="stat"><span class="num">{{ roots }}</span><span class="label">顶层</span></div>
        <div class="stat"><span class="num">{{ tags.length }}</span><span class="label">标签</span></div>
        <div class="stat"><span class="num">{{ tagged }}</span><span class="label">带标签</span></div>
      </div>

      <section class="col">
        <h2>最近改动</h2>
        <ul v-if="recentlyUpdated.length">
          <li v-for="n in recentlyUpdated" :key="n.id">
            <button class="note" @click="onSelect(n)">
              <span class="title">{{ n.title || '(无标题)' }}</span>
              <span class="time">{{ fmt(n.updated_at) }}</span>
            </button>
          </li>
        </ul>
        <p v-else class="empty">还没有笔记</p>
      </section>

      <section class="col">
        <h2>最近新建</h2>
        <ul v-if="recentlyCreated.length">
          <li v-for="n in recentlyCreated" :key="n.id">
            <button class="note" @click="onSelect(n)">
              <span class="title">{{ n.title || '(无标题)' }}</span>
              <span class="time">{{ fmt(n.created_at) }}</span>
            </button>
          </li>
        </ul>
        <p v-else class="empty">还没有笔记</p>
      </section>

      <section v-if="tags.length" class="col">
        <h2>标签</h2>
        <div class="chips">
          <span v-for="t in tags" :key="t" class="chip">#{{ t }}</span>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.overview {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 32px 40px 64px;
}
.hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 100%;
  color: var(--fg-muted);
}
.hero .big {
  margin: 0;
  font-size: 16px;
  color: var(--fg-default);
}
.hero .hint {
  margin: 0;
  font-size: 13px;
}
.repo-name {
  margin: 0 0 20px;
  font-size: 24px;
  font-weight: 600;
}
.stats {
  display: flex;
  gap: 28px;
  padding: 16px 20px;
  margin-bottom: 28px;
  background: var(--canvas-subtle);
  border: 1px solid var(--border-muted);
  border-radius: 8px;
}
.stat {
  display: flex;
  flex-direction: column;
}
.stat .num {
  font-size: 22px;
  font-weight: 600;
  line-height: 1.2;
}
.stat .label {
  font-size: 12px;
  color: var(--fg-muted);
}
.col {
  margin-bottom: 24px;
  max-width: 720px;
}
.col h2 {
  margin: 0 0 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--fg-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.col ul {
  list-style: none;
  margin: 0;
  padding: 0;
}
.note {
  display: flex;
  align-items: baseline;
  gap: 12px;
  width: 100%;
  padding: 7px 10px;
  color: var(--fg-default);
  background: none;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  text-align: left;
  cursor: pointer;
}
.note:hover {
  background: var(--btn-hover-bg);
}
.note .title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.note .time {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--fg-muted);
}
.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.chip {
  padding: 2px 8px;
  font-size: 12px;
  color: var(--accent-fg);
  background: var(--accent-subtle);
  border-radius: 999px;
}
.empty {
  margin: 0;
  padding: 8px 10px;
  font-size: 13px;
  color: var(--fg-muted);
}
</style>
