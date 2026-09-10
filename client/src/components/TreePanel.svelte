<script module lang="ts">
  // 模块级拖拽状态（所有递归实例共享）
  let draggedId: string | null = null;
</script>

<script lang="ts">
  // 递归笔记树：层级缩进，支持折叠/选中/新建子/删除/重命名/拖拽移动
  import type { Note } from '../api';
  import TreePanel from './TreePanel.svelte';

  export type TreeNode = Note & { children?: TreeNode[] };

  let {
    notes,
    depth = 0,
    selectedId,
    collapsed,
    onSelect,
    onToggle,
    onAddChild,
    onDelete,
    onRename,
    onMove,
  }: {
    notes: TreeNode[];
    depth?: number;
    selectedId: string | null;
    collapsed: Set<string>;
    onSelect: (n: Note) => void;
    onToggle: (n: Note) => void;
    onAddChild: (n: Note) => void;
    onDelete: (n: Note) => void;
    onRename: (n: Note, title: string) => void;
    onMove: (draggedId: string, targetId: string) => void;
  } = $props();

  let editingId: string | null = $state(null);
  let editValue = $state('');
  let renameEl: HTMLInputElement | undefined = $state();
  let dragOverId = $state<string | null>(null);

  function startRename(n: Note) {
    editingId = n.id;
    editValue = n.title;
  }

  function commitRename(n: Note) {
    if (editingId !== n.id) return;
    const v = editValue.trim();
    if (v && v !== n.title) onRename(n, v);
    editingId = null;
  }

  function cancelRename() {
    editingId = null;
  }

  $effect(() => {
    if (editingId && renameEl) renameEl.focus();
  });

  // 拖拽：dragstart 设置模块级 draggedId，drop 时调 onMove
  function onDragStart(e: DragEvent, n: Note) {
    draggedId = n.id;
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', n.id);
    }
  }

  function onDragEnd() {
    draggedId = null;
    dragOverId = null;
  }

  function onDragOver(e: DragEvent, n: Note) {
    if (!draggedId || draggedId === n.id) return;
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
    dragOverId = n.id;
  }

  function onDragLeave(n: Note) {
    if (dragOverId === n.id) dragOverId = null;
  }

  function onDrop(e: DragEvent, n: Note) {
    e.preventDefault();
    e.stopPropagation();
    const id = draggedId;
    dragOverId = null;
    if (id && id !== n.id) onMove(id, n.id);
  }
</script>

{#each notes as n (n.id)}
  <li class="row" class:active={selectedId === n.id} class:drag-over={dragOverId === n.id}
    style="padding-left: {8 + depth * 16}px"
    draggable={editingId !== n.id}
    ondragstart={(e) => onDragStart(e, n)}
    ondragend={onDragEnd}
    ondragover={(e) => onDragOver(e, n)}
    ondragleave={() => onDragLeave(n)}
    ondrop={(e) => onDrop(e, n)}
  >
    <button class="twisty" onclick={() => onToggle(n)} title={collapsed.has(n.id) ? '展开' : '折叠'}>
      {collapsed.has(n.id) ? '▶' : '▼'}
    </button>
    {#if editingId === n.id}
      <input class="rename-input" bind:this={renameEl} bind:value={editValue}
        onkeydown={(e) => {
          if (e.key === 'Enter') commitRename(n);
          else if (e.key === 'Escape') cancelRename();
        }}
        onblur={() => commitRename(n)} />
    {:else}
      <button class="link title" onclick={() => onSelect(n)}>
        {n.title || '(无标题)'}
      </button>
    {/if}
    <button class="icon" title="新建子笔记" onclick={() => onAddChild(n)}>＋</button>
    <button class="icon" title="重命名" onclick={() => startRename(n)}>✎</button>
    <button class="icon danger" title="删除" onclick={() => onDelete(n)}>×</button>
  </li>
  {#if !collapsed.has(n.id)}
    <TreePanel
      notes={n.children ?? []}
      depth={depth + 1}
      {selectedId}
      {collapsed}
      {onSelect}
      {onToggle}
      {onAddChild}
      {onDelete}
      {onRename}
      {onMove}
    />
  {/if}
{/each}

<style>
  li.row {
    display: flex;
    align-items: center;
    gap: 4px;
    padding-right: 4px;
    margin: 1px 0;
    border-radius: 4px;
    cursor: pointer;
  }
  li.row:hover { background: var(--hover); }
  li.row.active { background: var(--accent-soft); }
  li.row.active .title { color: var(--accent-text); font-weight: 600; }
  li.row.drag-over { background: var(--accent-soft); border: 1px dashed var(--accent); }
  .twisty {
    border: none;
    background: none;
    padding: 0;
    width: 20px;
    font-size: 10px;
    color: var(--muted);
    cursor: pointer;
  }
  .title {
    flex: 1;
    background: none;
    border: none;
    padding: 5px 0;
    text-align: left;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    cursor: pointer;
    color: inherit;
    font: inherit;
  }
  .rename-input {
    flex: 1;
    padding: 3px 6px;
    font: inherit;
    font-size: 13px;
    border: 1px solid var(--accent);
    border-radius: 4px;
    background: var(--input-bg);
  }
  .icon {
    border: none;
    background: none;
    padding: 0 4px;
    color: var(--muted);
    visibility: hidden;
    cursor: pointer;
    font-size: 13px;
  }
  li.row:hover .icon { visibility: visible; }
  .icon:hover { color: var(--fg); }
  .icon.danger { color: var(--danger); }
  .icon.danger:hover { color: var(--danger); }
</style>
