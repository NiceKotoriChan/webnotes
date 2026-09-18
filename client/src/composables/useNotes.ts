import { ref } from 'vue';
import * as api from '../api';
import type { Note, NotePatch } from '../api';

// 当前仓库的全部笔记（文档树用）。列表不带排序参数，排序由前端做，见 lib/tree.ts。
const notes = ref<Note[]>([]);
const error = ref<string | null>(null);
let repo: string | null = null;

export function useNotes() {
  function load(r: string) {
    repo = r;
    return refresh();
  }

  // limit 上限 1000（spec/api.md）；超过就得分页，暂时够用
  async function refresh(opts: api.ListNotesOpts = { limit: 1000 }) {
    const r = repo;
    if (!r) return;
    try {
      const list = (await api.listNotes(r, opts)) || [];
      if (r !== repo) return; // 仓库已切换，丢弃过期结果
      notes.value = list;
      error.value = null;
    } catch (e: any) {
      if (r !== repo) return;
      error.value = e.message;
    }
  }

  async function create(parent_id: string | null = null) {
    if (!repo) return null;
    const n = await api.createNote(repo, { parent_id });
    await refresh();
    return n;
  }

  async function update(id: string, patch: NotePatch) {
    if (!repo) return null;
    const n = await api.updateNote(repo, id, patch);
    await refresh();
    return n;
  }

  async function remove(id: string, permanent = false) {
    if (!repo) return;
    await api.deleteNote(repo, id, permanent);
    await refresh();
  }

  async function restore(id: string) {
    if (!repo) return;
    await api.restoreNote(repo, id);
    await refresh();
  }

  return { notes, error, load, refresh, create, update, remove, restore };
}
