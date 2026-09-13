import { ref } from 'vue';
import * as api from '../api';
import type { Repo } from '../api';

const repos = ref<Repo[]>([]);
const error = ref<string | null>(null);

export function useRepos() {
  async function load() {
    try {
      repos.value = (await api.listRepos()) || [];
      error.value = null;
    } catch (e: any) {
      error.value = e.message;
    }
  }
  async function create(name: string) {
    const r = await api.createRepo(name);
    await load();
    return r;
  }
  async function rename(id: string, name: string) {
    await api.renameRepo(id, name);
    await load();
  }
  async function remove(id: string) {
    await api.deleteRepo(id);
    await load();
  }
  return { repos, error, load, create, rename, remove };
}
