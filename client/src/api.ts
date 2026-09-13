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

async function request<T>(method: string, path: string, opts: RequestInit = {}): Promise<T> {
  const res = await fetch(BASE + path, { ...opts, method });
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  const body = text ? JSON.parse(text) : undefined;
  if (!res.ok) throw new ApiError(res.status, body?.error ?? res.statusText);
  return body as T;
}

const json = (data: unknown): RequestInit => ({
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(data),
});

// 类型
export interface Repo {
  id: string;
  name: string;
  ctime: number;
}
export interface Note {
  id: string;
  parent_id: string | null;
  title: string;
  content: string;
  ctime: number;
  mtime: number;
  deleted_at?: number | null;
  icon?: string | null;
}
export interface Tag {
  id: string;
  name: string;
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
export const createRepo = (name: string) => request<Repo>("POST", "/repos", json({ name }));
export const renameRepo = (id: string, name: string) =>
  request<Repo>("PATCH", `/repos/${id}`, json({ name }));
export const deleteRepo = (id: string) => request<void>("DELETE", `/repos/${id}`);

// Notes
export interface ListNotesOpts {
  q?: string;
  tag_id?: string;
  parent_id?: string; // 传空串 = 只列根节点；不传 = 全部
}
export const listNotes = (repo: string, opts: ListNotesOpts = {}) => {
  const p = new URLSearchParams();
  if (opts.q) p.set("q", opts.q);
  if (opts.tag_id) p.set("tag_id", opts.tag_id);
  if (opts.parent_id !== undefined) p.set("parent_id", opts.parent_id);
  p.set("limit", "1000");
  return request<Note[]>("GET", `/repos/${repo}/notes?${p}`);
};
export const createNote = (repo: string, body: { title: string; content: string; parent_id?: string | null }) =>
  request<Note>("POST", `/repos/${repo}/notes`, json(body));
export const updateNote = (repo: string, id: string, body: { title: string; content: string }) =>
  request<Note>("PUT", `/repos/${repo}/notes/${id}`, json(body));
// 移动笔记：parent_id 传 null 移到根
export const moveNote = (repo: string, id: string, parent_id: string | null) =>
  request<Note>("PATCH", `/repos/${repo}/notes/${id}`, json({ parent_id }));
export const deleteNote = (repo: string, id: string, permanent = false) =>
  request<void>("DELETE", `/repos/${repo}/notes/${id}${permanent ? "?permanent=1" : ""}`);
// 还原回收站里的笔记（连同子树）
export const restoreNote = (repo: string, id: string) =>
  request<void>("POST", `/repos/${repo}/notes/${id}/restore`);
// 回收站：被删除的顶层笔记
export const listTrash = (repo: string) =>
  request<Note[]>("GET", `/repos/${repo}/trash`);
// 设置/清除笔记自定义图标（icon=null 恢复自动匹配）
export const setNoteIcon = (repo: string, id: string, icon: string | null) =>
  request<void>("PATCH", `/repos/${repo}/notes/${id}/icon`, json({ icon }));

// Tags
export const listTags = (repo: string) => request<Tag[]>("GET", `/repos/${repo}/tags`);
export const createTag = (repo: string, name: string) =>
  request<Tag>("POST", `/repos/${repo}/tags`, json({ name }));
export const renameTag = (repo: string, id: string, name: string) =>
  request<Tag>("PUT", `/repos/${repo}/tags/${id}`, json({ name }));
export const deleteTag = (repo: string, id: string) =>
  request<void>("DELETE", `/repos/${repo}/tags/${id}`);

// Note-Tag
export const listNoteTags = (repo: string, noteId: string) =>
  request<Tag[]>("GET", `/repos/${repo}/notes/${noteId}/tags`);
export const addNoteTag = (repo: string, noteId: string, tagId: string) =>
  request<void>("POST", `/repos/${repo}/notes/${noteId}/tags`, json({ tag_id: tagId }));
export const removeNoteTag = (repo: string, noteId: string, tagId: string) =>
  request<void>("DELETE", `/repos/${repo}/notes/${noteId}/tags/${tagId}`);

// Assets（仓库内私有）
export const assetURL = (repo: string, sha: string, inline = false) =>
  `/api/repos/${repo}/assets/${sha}${inline ? "?inline=1" : ""}`;

export const listAssets = (repo: string) =>
  request<AssetMeta[]>("GET", `/repos/${repo}/assets`);
export const deleteAsset = (repo: string, sha: string) =>
  request<void>("DELETE", `/repos/${repo}/assets/${sha}`);

// 上传：客户端算 sha256，HEAD 检测秒传，POST 用 XHR 以便回报进度
export async function uploadAsset(
  repo: string,
  file: File,
  onProgress?: (loaded: number, total: number) => void,
): Promise<AssetMeta> {
  const buf = await file.arrayBuffer();
  const hash = await crypto.subtle.digest("SHA-256", buf);
  const sha = bytesToHex(new Uint8Array(hash));
  const url = assetURL(repo, sha);

  if ((await fetch(url, { method: "HEAD" })).status === 200) {
    return { id: sha, name: file.name, mime: file.type, size: file.size, ctime: Date.now() };
  }

  return new Promise<AssetMeta>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", url);
    xhr.setRequestHeader("X-Name", file.name);
    xhr.setRequestHeader("X-Mime", file.type || "application/octet-stream");
    xhr.setRequestHeader("X-Size", String(file.size));
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) onProgress(e.loaded, e.total);
    };
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(JSON.parse(xhr.responseText) as AssetMeta);
      } else {
        let msg = xhr.statusText;
        try {
          msg = JSON.parse(xhr.responseText).error ?? msg;
        } catch {
          /* noop */
        }
        reject(new ApiError(xhr.status, msg));
      }
    };
    xhr.onerror = () => reject(new ApiError(0, "上传失败"));
    xhr.send(file);
  });
}

function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}
