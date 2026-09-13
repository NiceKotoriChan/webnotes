<script setup lang="ts">
// 仓库菜单：列出所有仓库，支持新建、切换、改名、删除
import { ref, watch, nextTick } from 'vue';
import type { Repo } from '../../api';

defineProps<{
  repos: Repo[];
  currentId: string | undefined;
  errMsg: string;
}>();
const emit = defineEmits<{
  (e: 'create', name: string): void;
  (e: 'pick', r: Repo): void;
  (e: 'rename', r: Repo): void;
  (e: 'remove', r: Repo): void;
}>();

const newName = ref('');
const creating = ref(false);
const nameInput = ref<HTMLInputElement>();

watch(creating, async (v) => {
  if (v) {
    await nextTick();
    nameInput.value?.focus();
  }
});

function submitCreate() {
  const name = newName.value.trim();
  if (!name) return;
  emit('create', name);
  newName.value = '';
  creating.value = false;
}
</script>

<template>
  <div class="menu">
    <form v-if="creating" class="create" @submit.prevent="submitCreate">
      <input ref="nameInput" v-model="newName" placeholder="仓库名称" />
      <div class="actions">
        <button class="btn btn-primary btn-sm" type="submit">创建</button>
        <button class="btn btn-sm" type="button" @click="creating = false; newName = ''">取消</button>
      </div>
    </form>
    <button v-else class="new-btn" @click="creating = true">
      <Icon icon="mdi:plus" width="16" height="16" />
      新建仓库
    </button>

    <p v-if="errMsg" class="err">{{ errMsg }}</p>

    <ul class="list">
      <li v-for="r in repos" :key="r.id" :class="{ active: currentId === r.id }">
        <button class="name" @click="emit('pick', r)">
          <span class="icon"><Icon icon="mdi:folder" width="16" height="16" /></span>
          <span class="text">{{ r.name }}</span>
        </button>
        <div class="ops">
          <button class="op" title="改名" @click="emit('rename', r)">
            <Icon icon="mdi:pencil" width="14" height="14" />
          </button>
          <button class="op danger" title="删除" @click="emit('remove', r)">
            <Icon icon="mdi:delete" width="14" height="14" />
          </button>
        </div>
      </li>
      <li v-if="repos.length === 0" class="empty">还没有仓库</li>
    </ul>
  </div>
</template>

<style scoped>
.menu {
  padding: 8px;
}
.new-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 12px;
  color: var(--fg-default);
  background: transparent;
  border: 1px dashed var(--border-default);
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}
.new-btn:hover {
  background: var(--btn-hover-bg);
  border-color: var(--accent-emphasis);
}
.create {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.create input {
  width: 100%;
}
.actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
.err {
  color: var(--danger-fg);
  font-size: 12px;
  margin: 8px 0 0;
  padding: 0;
}
.list {
  list-style: none;
  margin: 8px 0 0;
  padding: 0;
  max-height: 300px;
  overflow-y: auto;
}
.list li {
  display: flex;
  align-items: center;
  margin: 2px 0;
  border-radius: 6px;
}
.list li:hover {
  background: var(--btn-hover-bg);
}
.list li.active {
  background: var(--accent-subtle);
}
.name {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  background: none;
  border: none;
  text-align: left;
  cursor: pointer;
  color: var(--fg-default);
  font-size: 14px;
  min-width: 0;
}
.icon {
  flex-shrink: 0;
}
.text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.list li.active .name {
  font-weight: 600;
}
.ops {
  display: flex;
  gap: 4px;
  padding-right: 6px;
  visibility: hidden;
}
.list li:hover .ops {
  visibility: visible;
}
.op {
  display: flex;
  padding: 4px;
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
  padding: 16px;
  text-align: center;
}
</style>
