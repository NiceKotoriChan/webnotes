import { writable } from "svelte/store";
import * as api from "./api";
import type { Note, Repo, Tag } from "./api";

// 记忆上次访问：仓库 id 全局一个；笔记 id 按仓库分别记
export const LAST_REPO_KEY = "webnotes:lastRepo";
export const lastNoteKey = (repo: string) => `webnotes:lastNote:${repo}`;

// 主题（明/暗）：记忆用户选择，默认跟随系统
export type Theme = "light" | "dark";
const THEME_KEY = "webnotes:theme";
function initialTheme(): Theme {
  const t = localStorage.getItem(THEME_KEY);
  if (t === "dark" || t === "light") return t;
  return window.matchMedia?.("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}
export const themeStore = writable<Theme>(initialTheme());
themeStore.subscribe((t) => {
  // 模块加载即生效（早于首次渲染，无闪烁）
  document.documentElement.classList.toggle("dark", t === "dark");
  localStorage.setItem(THEME_KEY, t);
});

// 当前仓库（null = 尚未选择仓库）
export const currentRepo = writable<Repo | null>(null);
currentRepo.subscribe((r) => {
  if (r) localStorage.setItem(LAST_REPO_KEY, r.id);
});

export const reposStore = (() => {
  const store = writable<Repo[]>([]);
  const error = writable<string | null>(null);
  return {
    subscribe: store.subscribe,
    error,
    async load() {
      try {
        store.set((await api.listRepos()) || []);
        error.set(null);
      } catch (e: any) {
        error.set(e.message);
      }
    },
    async create(name: string) {
      const r = await api.createRepo(name);
      await this.load();
      return r;
    },
    async rename(id: string, name: string) {
      await api.renameRepo(id, name);
      await this.load();
    },
    async remove(id: string) {
      await api.deleteRepo(id);
      await this.load();
    },
  };
})();

export const notesStore = (() => {
  const store = writable<Note[]>([]);
  const error = writable<string | null>(null);
  let repo: string | null = null;
  return {
    subscribe: store.subscribe,
    error,
    load(r: string) {
      repo = r;
      return this.refresh();
    },
    // 无参数 = 拉全部（树形用）；有过滤参数 = 搜索/标签结果
    async refresh(opts: api.ListNotesOpts = {}) {
      if (!repo) return;
      try {
        store.set((await api.listNotes(repo, opts)) || []);
        error.set(null);
      } catch (e: any) {
        error.set(e.message);
      }
    },
    async create(parent_id: string | null) {
      if (!repo) return null;
      const n = await api.createNote(repo, {
        title: "",
        content: "",
        parent_id,
      });
      await this.refresh();
      return n;
    },
    async remove(id: string) {
      if (!repo) return;
      await api.deleteNote(repo, id);
      await this.refresh();
    },
  };
})();

export const tagsStore = (() => {
  const store = writable<Tag[]>([]);
  const error = writable<string | null>(null);
  return {
    subscribe: store.subscribe,
    error,
    async load(repo: string) {
      try {
        store.set((await api.listTags(repo)) || []);
        error.set(null);
      } catch (e: any) {
        error.set(e.message);
      }
    },
    async create(repo: string, name: string) {
      await api.createTag(repo, name);
      await this.load(repo);
    },
    async remove(repo: string, id: string) {
      await api.deleteTag(repo, id);
      await this.load(repo);
    },
  };
})();
