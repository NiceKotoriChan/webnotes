import { ref } from 'vue';
import * as api from '../api';

// 标签不是实体：列表是后端派生的字符串数组，没有任何笔记在用的标签根本不存在。
const tags = ref<string[]>([]);
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

  // 改名：to 已存在就是合并，后端不报错；返回受影响的笔记数
  async function rename(repo: string, from: string, to: string) {
    const r = await api.renameTag(repo, from, to);
    await load(repo);
    return r.count;
  }

  async function remove(repo: string, name: string) {
    const r = await api.deleteTag(repo, name);
    await load(repo);
    return r.count;
  }

  return { tags, error, load, rename, remove };
}
