<script setup lang="ts">
// 图标选择器：搜索全部 mdi 图标，或从常用图标里挑；支持恢复自动匹配
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import { PRESET_ICONS, allMdiNames } from '../../lib/icons';

const props = defineProps<{ current: string | null }>();
const emit = defineEmits<{
  (e: 'select', icon: string | null): void;
  (e: 'close'): void;
}>();

const query = ref('');

onMounted(() => window.addEventListener('keydown', onKeydown));
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown));

const shown = computed(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return PRESET_ICONS;
  return allMdiNames().filter((n) => n.includes(q)).slice(0, 80);
});

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close');
}

function onBackdrop(e: MouseEvent) {
  if (e.target === e.currentTarget) emit('close');
}
</script>

<template>
  <div class="backdrop" @click="onBackdrop">
    <div class="panel" role="dialog" aria-modal="true" aria-label="选择图标">
      <div class="bar">
        <input v-model="query" placeholder="搜索图标..." />
        <button class="reset" @click="emit('select', null)">恢复自动匹配</button>
      </div>
      <div class="grid">
        <button
          v-for="name in shown"
          :key="name"
          class="cell"
          :class="{ active: name === current }"
          :title="name"
          @click="emit('select', name)"
        >
          <Icon :icon="'mdi:' + name" width="20" height="20" />
        </button>
        <div v-if="shown.length === 0" class="empty">没有匹配的图标</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.backdrop {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  z-index: 110;
}
.panel {
  width: 360px;
  max-width: 92vw;
  max-height: 70vh;
  display: flex;
  flex-direction: column;
  background: var(--canvas-overlay);
  border: 1px solid var(--border-default);
  border-radius: 12px;
  box-shadow: var(--shadow-floating);
  overflow: hidden;
}
.bar {
  display: flex;
  gap: 8px;
  padding: 12px;
  border-bottom: 1px solid var(--border-muted);
}
.bar input {
  flex: 1;
  min-width: 0;
}
.reset {
  flex-shrink: 0;
  padding: 0 10px;
  font-size: 12px;
  color: var(--accent-fg);
  background: transparent;
  border: 1px solid var(--border-default);
  border-radius: 6px;
  cursor: pointer;
}
.reset:hover {
  background: var(--btn-hover-bg);
}
.grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 4px;
  padding: 12px;
  overflow-y: auto;
}
.cell {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  aspect-ratio: 1;
  color: var(--fg-muted);
  background: transparent;
  border: 1px solid transparent;
  border-radius: 6px;
  cursor: pointer;
}
.cell:hover {
  background: var(--btn-hover-bg);
  color: var(--fg-default);
}
.cell.active {
  background: var(--accent-subtle);
  color: var(--accent-fg);
  border-color: var(--accent-emphasis);
}
.empty {
  grid-column: 1 / -1;
  padding: 24px;
  text-align: center;
  color: var(--fg-muted);
  font-size: 13px;
}
</style>
