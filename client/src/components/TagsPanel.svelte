<script lang="ts">
  // 标签面板：点击名称筛选；选中笔记时 ＋/✓ 打标签
  import { tagsStore } from '../stores';
  import type { Note, Tag } from '../api';

  let {
    repoId,
    tagId,
    selected,
    linkedIds,
    onSelect,
    onLink,
    onCreate,
    onRename,
    onDelete,
  }: {
    repoId: string | null;
    tagId: string;
    selected: Note | null;
    linkedIds: Set<string>;
    onSelect: (id: string) => void;
    onLink: (t: Tag) => void;
    onCreate: (name: string) => Promise<void>;
    onRename: (t: Tag) => void;
    onDelete: (t: Tag) => void;
  } = $props();

  let adding = $state(false);
  let newName = $state('');

  async function add() {
    const name = newName.trim();
    if (!name) return;
    await onCreate(name);
    newName = '';
    adding = false;
  }
</script>

<section class="tags-panel">
  <div class="panel-head">
    <h2>标签</h2>
    <button class="small" onclick={() => (adding = !adding)} disabled={!repoId}>＋</button>
  </div>
  {#if adding && repoId}
    <form class="add-tag" onsubmit={(e) => { e.preventDefault(); add(); }}>
      <input bind:value={newName} placeholder="标签名" />
      <button class="primary small" type="submit">添加</button>
    </form>
  {/if}
  <ul class="tag-list">
    {#if repoId}
      <li>
        <button class="tag" class:active={tagId === ''} onclick={() => onSelect('')}>
          <span class="hash">·</span>全部笔记
        </button>
      </li>
      {#each $tagsStore as t (t.id)}
        <li>
          <button class="tag" class:active={tagId === t.id}
            onclick={() => onSelect(tagId === t.id ? '' : t.id)}>
            <span class="hash">#</span>{t.name}
          </button>
          {#if selected}
            <button class="op" class:on={linkedIds.has(t.id)}
              title={linkedIds.has(t.id) ? '从当前笔记移除' : '加到当前笔记'}
              onclick={() => onLink(t)}>
              {linkedIds.has(t.id) ? '✓' : '＋'}
            </button>
          {/if}
          <button class="op" title="重命名" onclick={() => onRename(t)}>✎</button>
          <button class="op danger" title="删除标签" onclick={() => onDelete(t)}>×</button>
        </li>
      {/each}
      {#if $tagsStore.length === 0}
        <li class="empty">尚无标签</li>
      {/if}
    {:else}
      <li class="empty">打开仓库后可用</li>
    {/if}
  </ul>
</section>

<style>
  .tags-panel {
    flex-shrink: 0;
    max-height: 40%;
    display: flex;
    flex-direction: column;
    min-height: 0;
    border-top: 1px solid var(--border);
  }
  .panel-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
  }
  .panel-head h2 { margin: 0; font-size: 11px; color: var(--muted); font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; }
  .add-tag { display: flex; gap: 4px; padding: 0 8px 8px; }
  .add-tag input { flex: 1; min-width: 0; }
  .tag-list { list-style: none; margin: 0; padding: 0 4px 4px; overflow-y: auto; }
  .tag-list li { display: flex; align-items: center; border-radius: 4px; }
  .tag-list li:hover { background: var(--hover); }
  .tag {
    flex: 1;
    text-align: left;
    background: none;
    border: none;
    padding: 5px 8px;
    color: var(--fg);
    cursor: pointer;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 13px;
  }
  .tag .hash { color: var(--muted); margin-right: 2px; }
  .tag.active { color: var(--accent-text); font-weight: 600; }
  .tag.active .hash { color: var(--accent-text); }
  .op {
    border: none;
    background: none;
    color: var(--muted);
    padding: 2px 6px;
    font-size: 13px;
    visibility: hidden;
    cursor: pointer;
  }
  .tag-list li:hover .op { visibility: visible; }
  .op.on { color: var(--accent-text); font-weight: 700; }
  .op.danger { color: var(--danger); }
  .op.danger:hover { color: var(--danger); }
  .empty { color: var(--muted); padding: 12px; text-align: center; }
</style>
