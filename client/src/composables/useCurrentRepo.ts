import { ref, watch } from 'vue';
import type { Repo } from '../api';

// 记忆上次访问：仓库 id 全局一个；笔记 id 按仓库分别记
export const LAST_REPO_KEY = 'webnotes:lastRepo';
export const lastNoteKey = (repo: string) => `webnotes:lastNote:${repo}`;

const current = ref<Repo | null>(null);
watch(current, (r) => {
  if (r) localStorage.setItem(LAST_REPO_KEY, r.id);
});

export function useCurrentRepo() {
  return { current, set: (r: Repo | null) => (current.value = r) };
}
