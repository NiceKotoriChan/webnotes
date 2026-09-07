// API 客户端：薄封装 fetch，统一错误处理
const BASE = "/api";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

async function request<T>(
  method: string,
  path: string,
  opts: RequestInit = {},
): Promise<T> {
  const res = await fetch(BASE + path, { ...opts, method });
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  const body = text ? JSON.parse(text) : undefined;
  if (!res.ok) {
    const msg = body?.error ?? res.statusText;
    throw new ApiError(res.status, msg);
  }
  return body as T;
}

// 类型
export interface Repo {
  id: string;
}
export interface Note {
  id: string;
  title: string;
  content: string;
  ctime: number;
  mtime: number;
}
export interface Tag {
  id: string;
  name: string;
}
export interface Ref {
  source_id: string;
  target_id: string;
  ctime: number;
}
export interface AssetMeta {
  id: string;
  name: string;
  mime: string;
  size: number;
  ctime: number;
}

// Repos
export const listRepos = () => request<Repo[]>("GET", "/repos");
export const createRepo = (id: string) =>
  request<void>("POST", "/repos", {
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id }),
  });
export const deleteRepo = (id: string) =>
  request<void>("DELETE", `/repos/${id}`);

// Notes
export interface ListNotesOpts {
  q?: string;
  tag_id?: string;
  limit?: number;
  offset?: number;
}
export const listNotes = (repo: string, opts: ListNotesOpts = {}) => {
  const p = new URLSearchParams();
  if (opts.q) p.set("q", opts.q);
  if (opts.tag_id) p.set("tag_id", opts.tag_id);
  if (opts.limit) p.set("limit", String(opts.limit));
  if (opts.offset) p.set("offset", String(opts.offset));
  const qs = p.toString();
  return request<Note[]>("GET", `/repos/${repo}/notes${qs ? "?" + qs : ""}`);
};
export const createNote = (repo: string, title: string, content: string) =>
  request<Note>("POST", `/repos/${repo}/notes`, {
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title, content }),
  });
export const getNote = (repo: string, id: string) =>
  request<Note>("GET", `/repos/${repo}/notes/${id}`);
export const updateNote = (
  repo: string,
  id: string,
  title: string,
  content: string,
) =>
  request<Note>("PUT", `/repos/${repo}/notes/${id}`, {
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title, content }),
  });
export const deleteNote = (repo: string, id: string) =>
  request<void>("DELETE", `/repos/${repo}/notes/${id}`);

// Tags
export const listTags = (repo: string) =>
  request<Tag[]>("GET", `/repos/${repo}/tags`);
export const createTag = (repo: string, name: string) =>
  request<Tag>("POST", `/repos/${repo}/tags`, {
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
export const renameTag = (repo: string, id: string, name: string) =>
  request<Tag>("PUT", `/repos/${repo}/tags/${id}`, {
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
export const deleteTag = (repo: string, id: string) =>
  request<void>("DELETE", `/repos/${repo}/tags/${id}`);

// Note-Tag
export const listNoteTags = (repo: string, noteId: string) =>
  request<Tag[]>("GET", `/repos/${repo}/notes/${noteId}/tags`);
export const addNoteTag = (repo: string, noteId: string, tagId: string) =>
  request<void>("POST", `/repos/${repo}/notes/${noteId}/tags`, {
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ tag_id: tagId }),
  });
export const removeNoteTag = (repo: string, noteId: string, tagId: string) =>
  request<void>("DELETE", `/repos/${repo}/notes/${noteId}/tags/${tagId}`);

// Refs
export const listRefs = (repo: string, noteId: string) =>
  request<Ref[]>("GET", `/repos/${repo}/notes/${noteId}/refs`);
export const listBackrefs = (repo: string, noteId: string) =>
  request<Ref[]>("GET", `/repos/${repo}/notes/${noteId}/backrefs`);
export const createRef = (repo: string, noteId: string, targetId: string) =>
  request<Ref>("POST", `/repos/${repo}/notes/${noteId}/refs`, {
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ target_id: targetId }),
  });
export const deleteRef = (repo: string, noteId: string, targetId: string) =>
  request<void>("DELETE", `/repos/${repo}/notes/${noteId}/refs/${targetId}`);

// Assets
export const assetURL = (sha: string, inline = false) =>
  `/api/assets/${sha}${inline ? "?inline=1" : ""}`;

// 上传：客户端先算 sha256，HEAD 检测秒传，POST 实际上传
export async function uploadAsset(file: File): Promise<AssetMeta> {
  const buf = await file.arrayBuffer();
  const hash = await crypto.subtle.digest("SHA-256", buf);
  const sha = bytesToHex(new Uint8Array(hash));

  // HEAD 检查是否已 ready → 秒传
  const head = await fetch(assetURL(sha), { method: "HEAD" });
  if (head.status === 200) {
    return {
      id: sha,
      name: file.name,
      mime: file.type,
      size: file.size,
      ctime: Date.now(),
    };
  }

  // POST 上传
  const res = await fetch(assetURL(sha), {
    method: "POST",
    headers: {
      "X-Name": file.name,
      "X-Mime": file.type || "application/octet-stream",
      "X-Size": String(file.size),
    },
    body: file,
  });
  if (!res.ok) {
    const body = await res.text();
    let msg = res.statusText;
    try {
      msg = JSON.parse(body).error ?? msg;
    } catch {
      /* noop */
    }
    throw new ApiError(res.status, msg);
  }
  return res.json();
}

function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}
