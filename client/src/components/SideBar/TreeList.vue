<script setup lang="ts">
// 递归笔记树列表
import { ref, watch, nextTick } from 'vue';
import type { Note } from '../../api';
import type { TreeNode } from '../../lib/tree';
import { setDraggedId, getDraggedId } from '../../lib/drag';
import { autoIcon } from '../../lib/autoIcon';
import IconPicker from './IconPicker.vue';

defineOptions({ name: 'TreeList' });

const props = defineProps<{
  notes: TreeNode[];
  depth: number;
  selectedId: string | null;
  collapsed: Set<string>;
  onSelect: (n: Note) => void;
  onToggle: (n: Note) => void;
  onAddChild: (n: Note) => void;
  onDelete: (n: Note) => void;
  onRename: (n: Note, title: string) => void;
  onMove: (draggedId: string, targetId: string) => void;
  onSetIcon: (n: Note, icon: string | null) => void;
}>();

const editingId = ref<string | null>(null);
const editValue = ref('');
const renameEl = ref<HTMLInputElement | null>(null);
const dragOverId = ref<string | null>(null);
const pickerNote = ref<Note | null>(null);

function setRenameEl(el: unknown) {
  renameEl.value = el as HTMLInputElement | null;
}

watch(editingId, async (id) => {
  if (id) {
    await nextTick();
    renameEl.value?.focus();
  }
});

function iconFor(n: Note): string {
  return n.icon || autoIcon(n.title);
}

function onPickIcon(icon: string | null) {
  if (pickerNote.value) props.onSetIcon(pickerNote.value, icon);
  pickerNote.value = null;
}

function startRename(n: Note) {
  editingId.value = n.id;
  editValue.value = n.title;
}
function commitRename(n: Note) {
  if (editingId.value !== n.id) return;
  const v = editValue.value.trim();
  if (v && v !== n.title) props.onRename(n, v);
  editingId.value = null;
}
function cancelRename() {
  editingId.value = null;
}

// 拖拽
function onDragStart(e: DragEvent, n: Note) {
  setDraggedId(n.id);
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', n.id);
  }
}
function onDragEnd() {
  setDraggedId(null);
  dragOverId.value = null;
}
function onDragOver(e: DragEvent, n: Note) {
  const id = getDraggedId();
  if (!id || id === n.id) return;
  e.preventDefault();
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
  dragOverId.value = n.id;
}
function onDragLeave(n: Note) {
  if (dragOverId.value === n.id) dragOverId.value = null;
}
function onDrop(e: DragEvent, n: Note) {
  e.preventDefault();
  e.stopPropagation();
  const id = getDraggedId();
  dragOverId.value = null;
  if (id && id !== n.id) props.onMove(id, n.id);
}
</script>

<template>
  <ul class="tree" :style="{ paddingLeft: depth * 12 + 'px' }">
    <template v-for="n in notes" :key="n.id">
      <li
        class="row"
        :class="{ active: selectedId === n.id, 'drag-over': dragOverId === n.id }"
        :draggable="editingId !== n.id"
        @dragstart="onDragStart($event, n)"
        @dragend="onDragEnd"
        @dragover="onDragOver($event, n)"
        @dragleave="onDragLeave(n)"
        @drop="onDrop($event, n)"
      >
        <button class="twisty" :title="collapsed.has(n.id) ? '展开' : '折叠'" @click="onToggle(n)">
          <Icon
            v-if="n.children && n.children.length > 0"
            :icon="collapsed.has(n.id) ? 'mdi:chevron-right' : 'mdi:chevron-down'"
            width="12"
            height="12"
          />
          <span v-else class="twisty-placeholder"></span>
        </button>

        <button class="note-icon" title="设置图标" @click="pickerNote = n">
          <Icon :icon="'mdi:' + iconFor(n)" width="15" height="15" />
        </button>

        <input
          v-if="editingId === n.id"
          :ref="setRenameEl"
          v-model="editValue"
          class="rename-input"
          @keydown.enter="commitRename(n)"
          @keydown.esc="cancelRename"
          @blur="commitRename(n)"
        />
        <button v-else class="title" @click="onSelect(n)">{{ n.title || '(无标题)' }}</button>

        <div class="ops">
          <button class="op" title="新建子笔记" @click="onAddChild(n)">
            <Icon icon="mdi:plus" width="14" height="14" />
          </button>
          <button class="op" title="重命名" @click="startRename(n)">
            <Icon icon="mdi:pencil" width="14" height="14" />
          </button>
          <button class="op danger" title="删除" @click="onDelete(n)">
            <Icon icon="mdi:close" width="14" height="14" />
          </button>
        </div>
      </li>

      <TreeList
        v-if="!collapsed.has(n.id) && n.children && n.children.length > 0"
        :notes="n.children"
        :depth="depth + 1"
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
    </template>
    <li v-if="notes.length === 0 && depth === 0" class="empty">暂无笔记，点击上方 + 新建</li>
  </ul>

  <IconPicker v-if="pickerNote" :current="pickerNote.icon ?? null" @select="onPickIcon" @close="pickerNote = null" />
</template>

<style scoped>
.tree {
  list-style: none;
  margin: 0;
  padding: 4px 0;
}
.row {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 4px 8px;
  border-radius: 6px;
  cursor: pointer;
}
.row:hover {
  background: var(--btn-hover-bg);
}
.row.active {
  background: var(--accent-subtle);
}
.row.drag-over {
  background: var(--accent-subtle);
  outline: 1px dashed var(--accent-emphasis);
}
.twisty {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  background: none;
  border: none;
  color: var(--fg-muted);
  font-size: 10px;
  cursor: pointer;
  flex-shrink: 0;
}
.twisty-placeholder {
  width: 10px;
  height: 10px;
}
.note-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  background: none;
  border: none;
  color: var(--fg-muted);
  cursor: pointer;
  flex-shrink: 0;
}
.note-icon:hover {
  color: var(--accent-fg);
}
.title {
  flex: 1;
  padding: 2px 4px;
  background: none;
  border: none;
  text-align: left;
  color: var(--fg-default);
  font-size: 14px;
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.row.active .title {
  color: var(--accent-fg);
  font-weight: 600;
}
.rename-input {
  flex: 1;
  padding: 2px 6px;
  font-size: 14px;
  border: 1px solid var(--accent-emphasis);
  border-radius: 4px;
  background: var(--canvas-default);
  box-shadow: 0 0 0 3px var(--accent-subtle);
}
.ops {
  display: flex;
  gap: 2px;
  visibility: hidden;
}
.row:hover .ops {
  visibility: visible;
}
.op {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  background: none;
  border: none;
  color: var(--fg-muted);
  border-radius: 4px;
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
  font-size: 13px;
  padding: 12px;
  text-align: center;
}
</style>
