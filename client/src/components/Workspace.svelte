<script lang="ts">
  // 主工作区：VSC 布局 = 活动栏 | 侧边栏（文档树+标签）| 编辑器
  import { notesStore, tagsStore, lastNoteKey } from '../stores';
  import * as api from '../api';
  import type { Note, Repo, Tag } from '../api';
  import ActivityBar from './ActivityBar.svelte';
  import TreePanel, { type TreeNode } from './TreePanel.svelte';
  import TagsPanel from './TagsPanel.svelte';
  import NoteEditor from './NoteEditor.svelte';

  let { repo }: { repo: Repo | null } = $props();
  const repoId = $derived(repo?.id ?? null);

  let query = $state('');
  let tagId = $state('');
  let selected = $state<Note | null>(null);
  let errMsg = $state('');
  let loading = $state(false);
  let collapsed = $state<Set<string>>(new Set());
  let linkedIds = $state<Set<string>>(new Set());

  // 搜索/标签筛选时平铺展示结果，否则按树形展示
  const filtering = $derived(query.trim() !== '' || tagId !== '');
  const tree = $derived(buildTree($notesStore));
  const refreshOpts = () =>
    filtering ? { q: query.trim() || undefined, tag_id: tagId || undefined } : {};

  $effect(() => {
    if (repoId) tagsStore.load(repoId);
  });

  // 仓库切换或搜索/标签筛选变化时加载（load 同时设内部 repo + 拉取）
  $effect(() => {
    if (!repoId) return;
    const q = query.trim();
    const t = tagId;
    const opts = q || t ? { q: q || undefined, tag_id: t || undefined } : {};
    loading = true;
    notesStore.load(repoId, opts).finally(() => (loading = false));
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

  // notesStore 刷新后，把 selected 指向新对象（树内重命名后 NoteEditor 的 note prop 才能拿到新 title）
  $effect(() => {
    if (!selected) return;
    const id = selected.id;
    const fresh = $notesStore.find((x) => x.id === id);
    if (fresh && fresh !== selected) selected = fresh;
  });

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

  // 树内重命名：title 取新值，content 取 note 对象（上次保存态）
  async function renameNote(n: Note, title: string) {
    if (!repoId) return;
    errMsg = '';
    try {
      await api.updateNote(repoId, n.id, { title, content: n.content });
      await notesStore.refresh(refreshOpts());
    } catch (e: any) {
      errMsg = e.message;
    }
  }

  // 拖拽移动：把 dragged 移到 target 下（展开 target 让移动后的节点可见）
  async function moveNote(draggedId: string, targetId: string) {
    if (!repoId) return;
    errMsg = '';
    try {
      await api.moveNote(repoId, draggedId, targetId);
      const next = new Set(collapsed);
      next.delete(targetId);
      collapsed = next;
      await notesStore.refresh(refreshOpts());
    } catch (e: any) {
      errMsg = e.message;
      await notesStore.refresh(refreshOpts());
    }
  }

  // 拖到根节点：parent_id = null
  async function moveNoteToRoot(id: string) {
    if (!repoId) return;
    errMsg = '';
    try {
      await api.moveNote(repoId, id, null);
      await notesStore.refresh(refreshOpts());
    } catch (e: any) {
      errMsg = e.message;
      await notesStore.refresh(refreshOpts());
    }
  }

  function toggle(n: Note) {
    const next = new Set(collapsed);
    next.has(n.id) ? next.delete(n.id) : next.add(n.id);
    collapsed = next;
  }

  async function addTag(name: string) {
    if (!repoId) return;
    errMsg = '';
    try {
      await tagsStore.create(repoId, name);
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

  async function renameTag(t: Tag) {
    const name = prompt('标签名', t.name);
    if (name === null || !name.trim() || name.trim() === t.name) return;
    errMsg = '';
    try {
      await tagsStore.rename(repoId!, t.id, name.trim());
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
</script>

<div class="workspace">
  <ActivityBar />

  <aside class="sidebar">
    <!-- 文档树 -->
    <section class="tree-panel">
      <div class="panel-head">
        <h2>文档</h2>
        <button class="primary small" onclick={createRoot} disabled={!repoId}>＋ 新建</button>
      </div>
      {#if repoId}
        <div class="search-box">
          <input bind:value={query} placeholder="搜索笔记…" />
        </div>
      {/if}
      {#if errMsg}<p class="err">{errMsg}</p>{/if}

      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div class="list-wrap"
        ondragover={(e) => { if (e.dataTransfer?.types.includes('text/plain')) e.preventDefault(); }}
        ondrop={(e) => {
          e.preventDefault();
          const id = e.dataTransfer?.getData('text/plain');
          if (id) moveNoteToRoot(id);
        }}
      >
        {#if !repoId}
          <div class="empty">点击左侧仓库图标，打开或新建仓库</div>
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
            <TreePanel
              notes={tree}
              selectedId={selected?.id ?? null}
              {collapsed}
              onSelect={selectNote}
              onToggle={toggle}
              onAddChild={createChild}
              onDelete={remove}
              onRename={renameNote}
              onMove={moveNote}
            />
          </ul>
          {#if tree.length === 0}
            <div class="empty">暂无笔记，点击「新建」</div>
          {/if}
        {/if}
      </div>
    </section>

    <!-- 标签 -->
    <TagsPanel
      {repoId}
      {tagId}
      {selected}
      {linkedIds}
      onSelect={(id) => (tagId = id)}
      onLink={toggleLink}
      onCreate={addTag}
      onRename={renameTag}
      onDelete={deleteTag}
    />
  </aside>

  <!-- 右侧内容区：笔记 content -->
  <section class="content">
    {#if selected && repoId}
      {#key selected.id}
        <NoteEditor repo={repoId} note={selected} onSaved={() => notesStore.refresh(refreshOpts())} />
      {/key}
    {:else if !repoId}
      <div class="placeholder">
        <p>还没有打开仓库</p>
        <p class="hint">点击左侧仓库图标，新建或打开一个仓库开始</p>
      </div>
    {:else}
      <div class="placeholder">选择左侧笔记，或点击「新建」</div>
    {/if}
  </section>
</div>

<style>
  .workspace { display: flex; height: 100%; }

  .sidebar {
    width: 280px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    min-height: 0;
    background: var(--card);
    border-right: 1px solid var(--border);
  }
  .panel-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
  }
  .panel-head h2 { margin: 0; font-size: 11px; color: var(--muted); font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; }

  /* 文档树 */
  .tree-panel { display: flex; flex-direction: column; flex: 1; min-height: 0; }
  .search-box { padding: 0 8px 8px; }
  .search-box input { width: 100%; }
  .list-wrap { flex: 1; overflow-y: auto; padding: 0 4px 4px; }
  ul.tree, ul.flat { list-style: none; margin: 0; padding: 0; }
  ul.flat li {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 1px 0;
    padding: 6px 8px;
    border-radius: 4px;
    cursor: pointer;
  }
  ul.flat li:hover { background: var(--hover); }
  ul.flat li.active { background: var(--accent-soft); }
  ul.flat li.active .title { color: var(--accent-text); font-weight: 600; }
  .title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; text-align: left;
    background: none; border: none; padding: 0; cursor: pointer; color: inherit; font: inherit; }
  .mtime { font-size: 11px; color: var(--muted); white-space: nowrap; }

  .empty { color: var(--muted); padding: 16px; text-align: center; font-size: 13px; }
  .err { color: var(--danger); padding: 0 12px 8px; margin: 0; }

  /* 右侧内容区 */
  .content { flex: 1; display: flex; min-height: 0; background: var(--card); }
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
