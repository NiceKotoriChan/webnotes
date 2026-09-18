<script setup lang="ts">
// 标签面板（第 4 区块）。标签不是实体、没有 id：列表是后端派生的字符串，
// 「新建」就是把这个名字写到某篇笔记上，「删除」是全表改写。见 spec/model.md。
import { ref } from 'vue';
import { useTags } from '../../composables/useTags';
import type { Note } from '../../api';

const props = defineProps<{
  repoId: string | null;
  selected: Note | null;
  onToggle: (name: string) => void;
  onCreate: (name: string) => void;
  onRename: (name: string) => void;
  onDelete: (name: string) => void;
}>();

const { tags } = useTags();

const adding = ref(false);
const newName = ref('');

function submit() {
  const name = newName.value.trim();
  if (!name) return;
  props.onCreate(name);
  newName.value = '';
  adding.value = false;
}

const linked = (name: string) => !!props.selected?.tags?.includes(name);
</script>

<template>
  <div class="panel">
    <div class="header">
      <span class="title">标签</span>
      <button v-if="repoId && selected && !adding" class="add-btn" title="给当前笔记加标签" @click="adding = true">
        <Icon icon="mdi:plus" width="14" height="14" />
      </button>
    </div>

    <form v-if="adding" class="add-form" @submit.prevent="submit">
      <input v-model="newName" placeholder="标签名" />
      <button class="btn btn-primary btn-sm" type="submit">添加</button>
      <button class="btn btn-sm" type="button" @click="adding = false; newName = ''">取消</button>
    </form>

    <p v-if="repoId && !selected" class="hint">选中一篇笔记后可勾选标签</p>

    <ul class="list">
      <template v-if="repoId">
        <li v-for="name in tags" :key="name">
          <button class="tag" :disabled="!selected" @click="onToggle(name)">
            <span class="hash"><Icon icon="mdi:pound" width="12" height="12" /></span>
            <span class="name">{{ name }}</span>
            <span v-if="selected" class="check">
              <Icon v-if="linked(name)" icon="mdi:check" width="12" height="12" />
            </span>
          </button>
          <div class="ops">
            <button class="op" title="重命名（同名会自动合并）" @click="onRename(name)">
              <Icon icon="mdi:pencil" width="14" height="14" />
            </button>
            <button class="op danger" title="从所有笔记移除" @click="onDelete(name)">
              <Icon icon="mdi:close" width="14" height="14" />
            </button>
          </div>
        </li>
        <li v-if="tags.length === 0" class="empty">暂无标签</li>
      </template>
      <li v-else class="empty">打开仓库后可用</li>
    </ul>
  </div>
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
.title {
  font-size: 12px;
  font-weight: 600;
  color: var(--fg-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.add-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  background: none;
  border: none;
  color: var(--fg-muted);
  border-radius: 4px;
  cursor: pointer;
}
.add-btn:hover {
  background: var(--btn-hover-bg);
  color: var(--fg-default);
}
.hint {
  margin: 0;
  padding: 8px 12px;
  font-size: 12px;
  color: var(--fg-muted);
}
.add-form {
  display: flex;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-muted);
}
.add-form input {
  flex: 1;
  min-width: 0;
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
  border-radius: 6px;
  margin: 2px 0;
}
.list li:hover {
  background: var(--btn-hover-bg);
}
.tag {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
  background: none;
  border: none;
  text-align: left;
  cursor: pointer;
  color: var(--fg-default);
  font-size: 14px;
  min-width: 0;
}
.tag:disabled {
  cursor: default;
  color: var(--fg-muted);
}
.hash {
  color: var(--fg-muted);
  flex-shrink: 0;
}
.name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.check {
  color: var(--success-fg);
  margin-left: auto;
  flex-shrink: 0;
}
.ops {
  display: flex;
  gap: 2px;
  padding-right: 4px;
  visibility: hidden;
}
.list li:hover .ops {
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
