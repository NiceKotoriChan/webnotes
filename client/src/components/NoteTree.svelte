<script lang="ts">
  // 递归笔记树：按层级缩进渲染，支持折叠、选中、新建子笔记、删除
  import type { Note } from '../api';
  import NoteTree from './NoteTree.svelte';

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
  }: {
    notes: TreeNode[];
    depth?: number;
    selectedId: string | null;
    collapsed: Set<string>;
    onSelect: (n: Note) => void;
    onToggle: (n: Note) => void;
    onAddChild: (n: Note) => void;
    onDelete: (n: Note) => void;
  } = $props();
</script>

{#each notes as n (n.id)}
  <li class="row" class:active={selectedId === n.id} style="padding-left: {8 + depth * 16}px">
    <button
      class="twisty"
      onclick={() => onToggle(n)}
      title={collapsed.has(n.id) ? '展开' : '折叠'}
    >{collapsed.has(n.id) ? '▶' : '▼'}</button>
    <button class="link title" onclick={() => onSelect(n)}>
      {n.title || '(无标题)'}
    </button>
    <button class="icon" title="新建子笔记" onclick={() => onAddChild(n)}>＋</button>
    <button class="icon danger" title="删除" onclick={() => onDelete(n)}>×</button>
  </li>
  {#if !collapsed.has(n.id)}
    <NoteTree
      notes={n.children ?? []}
      depth={depth + 1}
      {selectedId}
      {collapsed}
      {onSelect}
      {onToggle}
      {onAddChild}
      {onDelete}
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
    border-radius: 6px;
    cursor: pointer;
  }
  li.row:hover { background: var(--hover); }
  li.row.active { background: var(--accent-soft); }
  li.row.active .title { color: var(--accent-text); font-weight: 600; }
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
    padding: 7px 0;
    text-align: left;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    cursor: pointer;
  }
  .icon {
    border: none;
    background: none;
    padding: 0 4px;
    color: var(--muted);
    visibility: hidden;
  }
  li.row:hover .icon { visibility: visible; }
  .icon.danger { color: var(--danger); }
</style>
