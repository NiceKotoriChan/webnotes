// API 客户端：薄封装 fetch。契约见 spec/api.md —— 19 条路由，方法只有 GET / POST。
//
// 三条贯穿全篇的约定：
// - GET 只读，POST 是唯一写入口；POST 按路径形状分新建（集合）/ 更新（资源）/ 动作（资源 + 动词）。
// - 一律回 JSON body，没有 204；删除类动作回 { id } 作回执。
// - 列表返回裸数组，不套信封。
const BASE = "/api";

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, body?: unknown): Promise<T> {
  const init: RequestInit = { method: body === undefined ? "GET" : "POST" };
  if (body !== undefined) {
    init.headers = { "Content-Type": "application/json" };
    init.body = JSON.stringify(body);
  }
  const res = await fetch(BASE + path, init);
  const text = await res.text();
  let parsed: any;
  try {
    parsed = text ? JSON.parse(text) : undefined;
  } catch {
    parsed = undefined; // 服务端意外回了非 JSON，按状态码报错即可
  }
  if (!res.ok) throw new ApiError(res.status, parsed?.error ?? res.statusText);
  return parsed as T;
}

// 部分更新的 body：只放「要改的字段」。字段缺席 = 不动，显式 null = 各有语义（见 spec）。
export interface NotePatch {
  title?: string;
  content?: string;
  parent_id?: string | null;
  tags?: string[];
  icon?: string | null;
}

// 类型（见 spec/model.md）
export interface Repo {
  id: string;
  name: string;
  date: number;
}
export interface Note {
  id: string;
  parent_id: string | null;
  title: string;
  content: string;
  created_at: number;
  updated_at: number;
  deleted_at: number | null;
  tags: string[];
  icon: string | null;
}
export interface AssetMeta {
  id: string;
  name: string;
  mime: string;
  size: number;
  date: number;
}
export interface Ack {
  id: string;
}
export interface CountAck {
  count: number;
}

// —— Repos ——
export const listRepos = () => request<Repo[]>("/repos");
export const createRepo = (name: string) => request<Repo>("/repos", { name });
export const renameRepo = (id: string, name: string) => request<Repo>(`/repos/${id}`, { name });
export const deleteRepo = (id: string) => request<Ack>(`/repos/${id}/delete`, {});

// —— Notes ——
export interface ListNotesOpts {
  q?: string;
  tag?: string; // 标签名，精确匹配
  parent_id?: string; // 传空串 = 只根节点；不传 = 全部未删
  limit?: number;
  offset?: number;
}
export const listNotes = (repo: string, opts: ListNotesOpts = {}) => {
  const p = new URLSearchParams();
  if (opts.q) p.set("q", opts.q);
  if (opts.tag) p.set("tag", opts.tag);
  if (opts.parent_id !== undefined) p.set("parent_id", opts.parent_id);
  if (opts.limit !== undefined) p.set("limit", String(opts.limit));
  if (opts.offset !== undefined) p.set("offset", String(opts.offset));
  const qs = p.toString();
  return request<Note[]>(`/repos/${repo}/notes${qs ? `?${qs}` : ""}`);
};
export const createNote = (repo: string, body: NotePatch = {}) =>
  request<Note>(`/repos/${repo}/notes`, body);
export const getNote = (repo: string, id: string) => request<Note>(`/repos/${repo}/notes/${id}`);
export const updateNote = (repo: string, id: string, patch: NotePatch) =>
  request<Note>(`/repos/${repo}/notes/${id}`, patch);
export const deleteNote = (repo: string, id: string, permanent = false) =>
  request<Ack>(`/repos/${repo}/notes/${id}/delete`, permanent ? { permanent: true } : {});
export const restoreNote = (repo: string, id: string) =>
  request<Note>(`/repos/${repo}/notes/${id}/restore`, {});
export const listTrash = (repo: string) => request<Note[]>(`/repos/${repo}/trash`);

// —— Tags（标签不是实体，没有 id，也没有「新建标签」）——
export const listTags = (repo: string) => request<string[]>(`/repos/${repo}/tags`);
export const renameTag = (repo: string, from: string, to: string) =>
  request<CountAck>(`/repos/${repo}/tags/rename`, { from, to });
export const deleteTag = (repo: string, name: string) =>
  request<CountAck>(`/repos/${repo}/tags/delete`, { name });

// —— Assets ——
export const assetURL = (repo: string, sha: string, inline = false) =>
  `/api/repos/${repo}/assets/${sha}${inline ? "?inline=1" : ""}`;

export const listAssets = (repo: string) => request<AssetMeta[]>(`/repos/${repo}/assets`);
// 秒传探测：404 = 没有这份内容，200 = 有（且拿到服务端权威元数据）
export const assetMeta = (repo: string, sha: string) =>
  request<AssetMeta>(`/repos/${repo}/assets/${sha}/meta`);
export const deleteAsset = (repo: string, sha: string) =>
  request<Ack>(`/repos/${repo}/assets/${sha}/delete`, {});

// 上传：客户端算 sha256 → GET meta 探秒传 → 未命中才 POST 原始字节。
// 用 XHR 是为了拿上传进度（fetch 没有上传进度事件）。
export async function uploadAsset(
  repo: string,
  file: File,
  onProgress?: (loaded: number, total: number) => void,
): Promise<AssetMeta> {
  const buf = await file.arrayBuffer();
  const hash = await crypto.subtle.digest("SHA-256", buf);
  const sha = bytesToHex(new Uint8Array(hash));

  try {
    return await assetMeta(repo, sha); // 秒传：服务端已有这份内容，一个字节都不用传
  } catch (e) {
    if (!(e instanceof ApiError) || e.status !== 404) throw e;
  }

  return new Promise<AssetMeta>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", assetURL(repo, sha));
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
          /* 非 JSON 错误体，用状态文本 */
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
