<script setup lang="ts">
// 回收站面板：列出被删除的顶层笔记，支持还原与彻底删除
import { ref, watch } from 'vue';
import * as api from '../../api';
import type { Note } from '../../api';

const props = defineProps<{ repoId: string | null }>();
const emit = defineEmits<{ (e: 'changed'): void }>();

const trash = ref<Note[]>([]);
const errMsg = ref('');

watch(
  () => props.repoId,
  (rid) => {
    if (rid) load(rid);
  },
  { immediate: true },
);

async function load(rid: string) {
  try {
    trash.value = (await api.listTrash(rid)) || [];
    errMsg.value = '';
  } catch (e: any) {
    errMsg.value = e.message;
  }
}

async function restore(n: Note) {
  const rid = props.repoId;
  if (!rid) return;
  errMsg.value = '';
  try {
    await api.restoreNote(rid, n.id);
    emit('changed');
    await load(rid);
  } catch (e: any) {
    errMsg.value = e.message;
  }
}

async function purge(n: Note) {
  const rid = props.repoId;
  if (!rid) return;
  if (!confirm(`彻底删除「${n.title || '(无标题)'}」？此操作不可恢复。`)) return;
  errMsg.value = '';
  try {
    await api.deleteNote(rid, n.id, true);
    await load(rid);
  } catch (e: any) {
    errMsg.value = e.message;
  }
}

function fmtTime(ts?: number | null) {
  if (!ts) return '';
  return new Date(ts).toLocaleString('zh-CN', { hour12: false });
}
</script>

<template>
  <section class="panel">
    <div class="header">
      <h2>回收站</h2>
      <span class="count">{{ repoId ? trash.length : '' }}</span>
    </div>

    <p v-if="errMsg" class="err">{{ errMsg }}</p>

    <ul v-if="repoId" class="list">
      <li v-for="n in trash" :key="n.id">
        <div class="meta">
          <span class="name">{{ n.title || '(无标题)' }}</span>
          <span class="sub">{{ fmtTime(n.deleted_at) }}</span>
        </div>
        <div class="ops">
          <button class="op" title="还原" @click="restore(n)"><Icon icon="mdi:restore" width="14" height="14" /></button>
          <button class="op danger" title="彻底删除" @click="purge(n)"><Icon icon="mdi:delete" width="14" height="14" /></button>
        </div>
      </li>
      <li v-if="trash.length === 0" class="empty">回收站是空的</li>
    </ul>
    <div v-else class="empty">打开仓库后可用</div>
  </section>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-muted);
}
.header h2 {
  margin: 0;
  font-size: 12px;
  font-weight: 600;
  color: var(--fg-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.count {
  font-size: 12px;
  color: var(--fg-muted);
}
.list {
  list-style: none;
  margin: 0;
  padding: 4px 8px;
  overflow-y: auto;
  flex: 1;
}
.list li {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 6px;
  margin: 2px 0;
}
.list li:hover {
  background: var(--btn-hover-bg);
}
.meta {
  flex: 1;
  min-width: 0;
}
.name {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  color: var(--fg-default);
}
.sub {
  font-size: 12px;
  color: var(--fg-muted);
}
.ops {
  display: flex;
  gap: 4px;
  visibility: hidden;
}
.list li:hover .ops {
  visibility: visible;
}
.op {
  border: none;
  background: none;
  padding: 4px 6px;
  display: inline-flex;
  color: var(--fg-muted);
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}
.op:hover {
  background: var(--btn-active-bg);
  color: var(--fg-default);
}
.op.danger {
  color: var(--danger-fg);
}
.op.danger:hover {
  background: var(--danger-subtle);
}
.empty {
  color: var(--fg-muted);
  padding: 24px;
  text-align: center;
  font-size: 13px;
}
.err {
  color: var(--danger-fg);
  padding: 0 12px 8px;
  margin: 0;
  font-size: 12px;
}
</style>
