import { ref } from 'vue';
import * as api from '../api';
import type { Note } from '../api';

const notes = ref<Note[]>([]);
const error = ref<string | null>(null);
let repo: string | null = null;

export function useNotes() {
  function load(r: string) {
    repo = r;
    return refresh();
  }
  // 无参数 = 拉全部（树形用）；有过滤参数 = 搜索/标签结果
  async function refresh(opts: api.ListNotesOpts = {}) {
    const r = repo;
    if (!r) return;
    try {
      const list = (await api.listNotes(r, opts)) || [];
      if (r !== repo) return; // repo 已切换，丢弃过期结果
      notes.value = list;
      error.value = null;
    } catch (e: any) {
      if (r !== repo) return;
      error.value = e.message;
    }
  }
  async function create(parent_id: string | null) {
    if (!repo) return null;
    const n = await api.createNote(repo, { title: '', data: '', parent_id });
    await refresh();
    return n;
  }
  async function remove(id: string) {
    if (!repo) return;
    await api.deleteNote(repo, id);
    await refresh();
  }
  return { notes, error, load, refresh, create, remove };
}
