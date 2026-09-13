<script setup lang="ts">
// 文档树面板：顶部仓库名 + 递归笔记树
import type { Note } from '../../api';
import type { TreeNode } from '../../lib/tree';
import TreeList from './TreeList.vue';

defineProps<{
  repoName: string | null;
  notes: TreeNode[];
  selectedId: string | null;
  collapsed: Set<string>;
  onSelect: (n: Note) => void;
  onToggle: (n: Note) => void;
  onAddChild: (n: Note) => void;
  onDelete: (n: Note) => void;
  onRename: (n: Note, title: string) => void;
  onMove: (draggedId: string, targetId: string) => void;
  onSetIcon: (n: Note, icon: string | null) => void;
  onCreateRoot: () => void;
  loading: boolean;
  repoId: string | null;
}>();
</script>

<template>
  <div class="tree-panel">
    <div class="header">
      <span class="repo-name">{{ repoName ?? '未打开仓库' }}</span>
      <button class="new-btn" :disabled="!repoId" title="新建笔记" @click="onCreateRoot">
        <Icon icon="mdi:plus" width="14" height="14" />
      </button>
    </div>

    <div class="tree-scroll">
      <div v-if="!repoId" class="empty">点击左侧仓库图标打开仓库</div>
      <div v-else-if="loading" class="empty">加载中...</div>
      <TreeList
        v-else
        :notes="notes"
        :depth="0"
        :selected-id="selectedId"
        :collapsed="collapsed"
        :on-select="onSelect"
        :on-toggle="onToggle"
        :on-add-child="onAddChild"
        :on-delete="onDelete"
        :on-rename="onRename"
        :on-move="onMove"
        :on-set-icon="onSetIcon"
      />
    </div>
  </div>
</template>

<style scoped>
.tree-panel {
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
  flex-shrink: 0;
}
.repo-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--fg-default);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.new-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: var(--fg-muted);
  cursor: pointer;
}
.new-btn:hover {
  background: var(--btn-hover-bg);
  color: var(--fg-default);
}
.new-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.tree-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
.empty {
  color: var(--fg-muted);
  font-size: 13px;
  padding: 16px;
  text-align: center;
}
</style>
