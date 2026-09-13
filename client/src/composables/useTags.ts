import { ref } from 'vue';
import * as api from '../api';
import type { Tag } from '../api';

const tags = ref<Tag[]>([]);
const error = ref<string | null>(null);
let seq = 0;

export function useTags() {
  async function load(repo: string) {
    const s = ++seq;
    try {
      const list = (await api.listTags(repo)) || [];
      if (s !== seq) return; // 已被更新的 load 取代
      tags.value = list;
      error.value = null;
    } catch (e: any) {
      if (s !== seq) return;
      error.value = e.message;
    }
  }
  async function create(repo: string, name: string) {
    await api.createTag(repo, name);
    await load(repo);
  }
  async function rename(repo: string, id: string, name: string) {
    await api.renameTag(repo, id, name);
    await load(repo);
  }
  async function remove(repo: string, id: string) {
    await api.deleteTag(repo, id);
    await load(repo);
  }
  return { tags, error, load, create, rename, remove };
}
