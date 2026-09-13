<script setup lang="ts">
// 附件面板：侧边栏展示当前仓库全部附件，支持预览、复制链接、下载、删除
import { watch } from 'vue';
import { useAssets } from '../../composables/useAssets';
import * as api from '../../api';

const props = defineProps<{ repoId: string | null }>();

const { assets, error: errMsg, load, remove: removeAsset } = useAssets();

watch(
  () => props.repoId,
  (rid) => {
    if (rid) load(rid);
  },
  { immediate: true },
);

async function remove(m: { id: string; name: string }) {
  const rid = props.repoId;
  if (!rid) return;
  if (!confirm(`删除附件「${m.name}」？正文中引用它的地方将无法显示。`)) return;
  try {
    await removeAsset(m.id);
  } catch (e: any) {
    errMsg.value = e.message;
  }
}

async function copyLink(m: { id: string }) {
  const rid = props.repoId;
  if (!rid) return;
  try {
    await navigator.clipboard.writeText(api.assetURL(rid, m.id));
  } catch {
    /* clipboard 不可用时忽略 */
  }
}

function fmtSize(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

function fmtTime(ts: number) {
  return new Date(ts).toLocaleString('zh-CN', { hour12: false });
}
</script>

<template>
  <section class="panel">
    <div class="header">
      <h2>附件</h2>
      <span class="count">{{ repoId ? assets.length : '' }}</span>
    </div>

    <p v-if="errMsg" class="err">{{ errMsg }}</p>

    <ul v-if="repoId" class="asset-list">
      <li v-for="m in assets" :key="m.id">
        <img v-if="m.mime.startsWith('image/')" class="thumb" :src="api.assetURL(repoId, m.id, true)" :alt="m.name" loading="lazy" />
        <span v-else class="file-icon"><Icon icon="mdi:file-outline" width="20" height="20" /></span>
        <div class="meta">
          <a class="name" :href="api.assetURL(repoId, m.id)" target="_blank" rel="noreferrer" :title="m.name">{{ m.name }}</a>
          <span class="sub">{{ fmtSize(m.size) }} · {{ fmtTime(m.ctime) }}</span>
        </div>
        <div class="ops">
          <button class="op" title="复制链接" @click="copyLink(m)">
            <Icon icon="mdi:link" width="14" height="14" />
          </button>
          <button class="op danger" title="删除" @click="remove(m)">
            <Icon icon="mdi:delete" width="14" height="14" />
          </button>
        </div>
      </li>
      <li v-if="assets.length === 0" class="empty">还没有附件。在编辑器中拖入或粘贴文件即可上传。</li>
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
.asset-list {
  list-style: none;
  margin: 0;
  padding: 4px 8px;
  overflow-y: auto;
  flex: 1;
}
.asset-list li {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 6px;
  margin: 2px 0;
}
.asset-list li:hover {
  background: var(--btn-hover-bg);
}
.thumb {
  width: 40px;
  height: 40px;
  object-fit: cover;
  border-radius: 6px;
  border: 1px solid var(--border-default);
  flex-shrink: 0;
}
.file-icon {
  width: 40px;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  flex-shrink: 0;
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
.asset-list li:hover .ops {
  visibility: visible;
}
.op {
  border: none;
  background: none;
  padding: 4px;
  display: inline-flex;
  color: var(--fg-muted);
  border-radius: 6px;
  cursor: pointer;
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
