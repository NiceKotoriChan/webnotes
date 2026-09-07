import { writable } from 'svelte/store';
import * as api from './api';
import type { Repo, Note, Tag } from './api';

export const reposStore = (() => {
  const store = writable<Repo[]>([]);
  const error = writable<string | null>(null);
  return {
    subscribe: store.subscribe,
    error,
    async load() {
      try {
        const r = await api.listRepos();
        store.set(r || []);
        error.set(null);
      } catch (e: any) {
        error.set(e.message);
      }
    },
    async create(id: string) {
      await api.createRepo(id);
      await this.load();
    },
    async remove(id: string) {
      await api.deleteRepo(id);
      await this.load();
    },
  };
})();

export const currentRepo = writable<string | null>(null);

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
    async refresh(opts: api.ListNotesOpts = {}) {
      if (!repo) return;
      try {
        const list = await api.listNotes(repo, opts);
        store.set(list || []);
        error.set(null);
      } catch (e: any) {
        error.set(e.message);
      }
    },
    async create(title: string, content: string) {
      if (!repo) return null;
      const n = await api.createNote(repo, title, content);
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
        const list = await api.listTags(repo);
        store.set(list || []);
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
