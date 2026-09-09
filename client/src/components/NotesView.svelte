<script lang="ts">
  // 主界面：左侧三卡片（设置 / 文档树 / 标签），右侧笔记内容
  import { notesStore, tagsStore, lastNoteKey } from '../stores';
  import * as api from '../api';
  import type { Note, Repo, Tag } from '../api';
  import SettingsCard from './SettingsCard.svelte';
  import NoteTree, { type TreeNode } from './NoteTree.svelte';
  import NoteEditor from './NoteEditor.svelte';

  let { repo }: { repo: Repo | null } = $props();
  const repoId = $derived(repo?.id ?? null);

  let query = $state('');
  let tagId = $state('');
  let selected = $state<Note | null>(null);
  let errMsg = $state('');
  let loading = $state(false);
  let collapsed = $state<Set<string>>(new Set());
  let addingTag = $state(false);
  let newTag = $state('');
  let linkedIds = $state<Set<string>>(new Set());

  // 搜索/标签筛选时平铺展示结果，否则按树形展示
  const filtering = $derived(query.trim() !== '' || tagId !== '');
  const tree = $derived(buildTree($notesStore));
  const refreshOpts = () =>
    filtering ? { q: query.trim() || undefined, tag_id: tagId || undefined } : {};

  $effect(() => {
    if (repoId) tagsStore.load(repoId);
  });

  $effect(() => {
    if (!repoId) return;
    const q = query.trim();
    const t = tagId;
    loading = true;
    notesStore
      .refresh(q || t ? { q: q || undefined, tag_id: t || undefined } : {})
      .finally(() => (loading = false));
  });

  // 当前笔记的标签关联（选中笔记变化时加载）
  $effect(() => {
    const rid = repoId;
    const nid = selected?.id;
    if (!rid || !nid) {
      linkedIds = new Set();
      return;
    }
    api
      .listNoteTags(rid, nid)
      .then((ts) => (linkedIds = new Set(ts.map((t) => t.id))))
      .catch(() => {});
  });

  async function createRoot() {
    return create(null);
  }
  async function createChild(parent: Note) {
    collapsed = new Set(collapsed);
    collapsed.delete(parent.id); // 展开父节点，让新子笔记可见
    return create(parent.id);
  }
  async function create(parent_id: string | null) {
    if (!repoId) return;
    errMsg = '';
    try {
      const n = await notesStore.create(parent_id);
      if (n) selectNote(n);
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  // 选中笔记并记住；下次进入该仓库自动恢复
  function selectNote(n: Note) {
    selected = n;
    if (repoId) localStorage.setItem(lastNoteKey(repoId), n.id);
  }

  async function remove(n: Note) {
    if (!confirm('删除该笔记？其下级笔记将一并删除。')) return;
    try {
      await notesStore.remove(n.id);
      if (selected?.id === n.id) {
        selected = null;
        if (repoId) localStorage.removeItem(lastNoteKey(repoId));
      }
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  function toggle(n: Note) {
    const next = new Set(collapsed);
    next.has(n.id) ? next.delete(n.id) : next.add(n.id);
    collapsed = next;
  }

  async function addTag() {
    const name = newTag.trim();
    if (!name || !repoId) return;
    errMsg = '';
    try {
      await tagsStore.create(repoId, name);
      newTag = '';
      addingTag = false;
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  async function deleteTag(t: Tag) {
    if (!repoId) return;
    if (!confirm(`删除标签「${t.name}」？`)) return;
    try {
      await tagsStore.remove(repoId, t.id);
      if (tagId === t.id) tagId = '';
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  // 给当前笔记打/去标签
  async function toggleLink(t: Tag) {
    if (!repoId || !selected) return;
    errMsg = '';
    try {
      if (linkedIds.has(t.id)) {
        await api.removeNoteTag(repoId, selected.id, t.id);
        const next = new Set(linkedIds);
        next.delete(t.id);
        linkedIds = next;
      } else {
        await api.addNoteTag(repoId, selected.id, t.id);
        const next = new Set(linkedIds);
        next.add(t.id);
        linkedIds = next;
      }
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  function fmtTime(ts: number) {
    return new Date(ts).toLocaleString('zh-CN', { hour12: false });
  }

  // 扁平列表 → 树：按 parent_id 挂接，孤儿（父已删）归到根
  function buildTree(notes: Note[]): TreeNode[] {
    const map = new Map<string, TreeNode>();
    for (const n of notes) map.set(n.id, { ...n, children: [] });
    const roots: TreeNode[] = [];
    for (const n of map.values()) {
      const p = n.parent_id && map.get(n.parent_id);
      (p ? p.children! : roots).push(n);
    }
    const byMtime = (a: TreeNode, b: TreeNode) => b.mtime - a.mtime;
    roots.sort(byMtime);
    map.forEach((n) => n.children!.sort(byMtime));
    return roots;
  }

  // 首次加载完笔记后，恢复上次打开的笔记（组件按仓库 {#key} 重挂，每仓库只恢复一次）
  let didRestore = false;
  $effect(() => {
    const list = $notesStore;
    if (didRestore || filtering || list.length === 0 || !repoId) return;
    didRestore = true;
    const nid = localStorage.getItem(lastNoteKey(repoId));
    const found = nid ? list.find((n) => n.id === nid) : undefined;
    if (found) selected = found;
  });
</script>

<div class="layout">
  <aside>
    <SettingsCard />

    <!-- 文档树卡片 -->
    <div class="card tree-card">
      <div class="card-head">
        <h2>文档</h2>
        <button class="primary small" onclick={createRoot} disabled={!repoId}>＋ 新建</button>
      </div>
      {#if repoId}
        <div class="search-box">
          <input bind:value={query} placeholder="搜索笔记…" />
        </div>
      {/if}
      {#if errMsg}<p class="err">{errMsg}</p>{/if}

      <div class="list-wrap">
        {#if !repoId}
          <div class="empty">点击上方仓库图标，打开或新建仓库</div>
        {:else if loading}
          <div class="empty">加载中…</div>
        {:else if filtering}
          <ul class="flat">
            {#each $notesStore as n (n.id)}
              <li class:active={selected?.id === n.id}>
                <button class="link title" onclick={() => selectNote(n)}>
                  {n.title || '(无标题)'}
                </button>
                <span class="mtime">{fmtTime(n.mtime)}</span>
              </li>
            {:else}
              <li class="empty">无匹配结果</li>
            {/each}
          </ul>
        {:else}
          <ul class="tree">
            <NoteTree
              notes={tree}
              selectedId={selected?.id ?? null}
              {collapsed}
              onSelect={selectNote}
              onToggle={toggle}
              onAddChild={createChild}
              onDelete={remove}
            />
          </ul>
          {#if tree.length === 0}
            <div class="empty">暂无笔记，点击「新建」</div>
          {/if}
        {/if}
      </div>
    </div>

    <!-- 标签卡片：点击名称筛选；选中笔记时 ＋/✓ 打标签 -->
    <div class="card tags-card">
      <div class="card-head">
        <h2>标签</h2>
        <button class="small" onclick={() => (addingTag = !addingTag)} disabled={!repoId}>＋ 新标签</button>
      </div>
      {#if addingTag && repoId}
        <form class="add-tag" onsubmit={(e) => { e.preventDefault(); addTag(); }}>
          <input bind:value={newTag} placeholder="标签名" />
          <button class="primary small" type="submit">添加</button>
        </form>
      {/if}
      <ul class="tag-list">
        {#if repoId}
          <li>
            <button class="tag" class:active={tagId === ''} onclick={() => (tagId = '')}>
              <span class="hash">·</span>全部笔记
            </button>
          </li>
          {#each $tagsStore as t (t.id)}
            <li>
              <button class="tag" class:active={tagId === t.id}
                onclick={() => (tagId = tagId === t.id ? '' : t.id)}>
                <span class="hash">#</span>{t.name}
              </button>
              {#if selected}
                <button class="op" class:on={linkedIds.has(t.id)}
                  title={linkedIds.has(t.id) ? '从当前笔记移除' : '加到当前笔记'}
                  onclick={() => toggleLink(t)}>
                  {linkedIds.has(t.id) ? '✓' : '＋'}
                </button>
              {/if}
              <button class="op danger" title="删除标签" onclick={() => deleteTag(t)}>×</button>
            </li>
          {/each}
          {#if $tagsStore.length === 0}
            <li class="empty">尚无标签</li>
          {/if}
        {:else}
          <li class="empty">打开仓库后可用</li>
        {/if}
      </ul>
    </div>
  </aside>

  <!-- 右侧内容区：笔记 content -->
  <section class="content">
    {#if selected && repoId}
      {#key selected.id}
        <NoteEditor repo={repoId} note={selected} onSaved={() => notesStore.refresh(refreshOpts())} />
      {/key}
    {:else if !repoId}
      <div class="card placeholder">
        <p>还没有打开仓库</p>
        <p class="hint">点击左上角仓库图标，新建或打开一个仓库开始</p>
      </div>
    {:else}
      <div class="card placeholder">选择左侧笔记，或点击「新建」</div>
    {/if}
  </section>
</div>

<style>
  .layout { display: flex; gap: 12px; padding: 12px; height: 100%; }
  aside {
    width: 280px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-height: 0;
  }
  .card-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
  }
  .card-head h2 { margin: 0; font-size: 13px; color: var(--muted); font-weight: 600; }

  /* 文档树卡片 */
  .tree-card { display: flex; flex-direction: column; flex: 1; min-height: 0; }
  .search-box { padding: 0 12px 8px; }
  .search-box input { width: 100%; }
  .list-wrap { flex: 1; overflow-y: auto; padding: 0 6px 6px; }
  ul.tree, ul.flat { list-style: none; margin: 0; padding: 0; }
  ul.flat li {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 1px 0;
    padding: 7px 8px;
    border-radius: 6px;
    cursor: pointer;
  }
  ul.flat li:hover { background: var(--hover); }
  ul.flat li.active { background: var(--accent-soft); }
  ul.flat li.active .title { color: var(--accent-text); font-weight: 600; }
  .title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; text-align: left;
    background: none; border: none; padding: 0; cursor: pointer; }
  .mtime { font-size: 12px; color: var(--muted); white-space: nowrap; }

  /* 标签卡片 */
  .tags-card { flex-shrink: 0; max-height: 40%; display: flex; flex-direction: column; min-height: 0; }
  .add-tag { display: flex; gap: 4px; padding: 0 12px 8px; }
  .add-tag input { flex: 1; min-width: 0; }
  .tag-list { list-style: none; margin: 0; padding: 0 6px 6px; overflow-y: auto; }
  .tag-list li { display: flex; align-items: center; margin: 1px 0; border-radius: 6px; }
  .tag-list li:hover { background: var(--hover); }
  .tag {
    flex: 1;
    text-align: left;
    background: none;
    border: none;
    padding: 6px 8px;
    color: var(--fg);
    cursor: pointer;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
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
  }
  .tag-list li:hover .op { visibility: visible; }
  .op.on { color: var(--accent-text); font-weight: 700; }
  .op.danger:hover, .op.danger { color: var(--danger); }

  .empty { color: var(--muted); padding: 16px; text-align: center; }
  .err { color: var(--danger); padding: 0 12px 8px; margin: 0; }

  /* 右侧内容区 */
  .content { flex: 1; display: flex; min-height: 0; }
  .placeholder {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: var(--muted);
    gap: 4px;
  }
  .placeholder p { margin: 0; }
  .placeholder .hint { font-size: 13px; opacity: 0.8; }
</style>
