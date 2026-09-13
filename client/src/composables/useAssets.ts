import { ref } from 'vue';
import * as api from '../api';
import type { AssetMeta } from '../api';

const assets = ref<AssetMeta[]>([]);
const error = ref<string | null>(null);
let repo: string | null = null;

export function useAssets() {
  function load(r: string) {
    repo = r;
    return refresh();
  }
  async function refresh() {
    const r = repo;
    if (!r) return;
    try {
      const list = (await api.listAssets(r)) || [];
      if (r !== repo) return; // repo 已切换，丢弃过期结果
      assets.value = list;
      error.value = null;
    } catch (e: any) {
      if (r !== repo) return;
      error.value = e.message;
    }
  }
  async function remove(sha: string) {
    if (!repo) return;
    await api.deleteAsset(repo, sha);
    await refresh();
  }
  return { assets, error, load, refresh, remove };
}
