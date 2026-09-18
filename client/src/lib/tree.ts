// 笔记树：把扁平列表组装成树。
// 接口不出排序参数（spec/api.md），排序在前端做 —— 文档树按名称排（docs/ui.md）。
import type { Note } from '../api';

export type TreeNode = Note & { children?: TreeNode[] };

// 中文按拼音排，数字按数值排（`第2章` 在 `第10章` 前）；同名用 id 兜底，保证顺序稳定
const byName = (a: TreeNode, b: TreeNode) =>
  a.title.localeCompare(b.title, 'zh-CN', { numeric: true, sensitivity: 'base' }) ||
  (a.id < b.id ? -1 : a.id > b.id ? 1 : 0);

export function buildTree(notes: Note[]): TreeNode[] {
  const map = new Map<string, TreeNode>();
  for (const n of notes) map.set(n.id, { ...n, children: [] });
  const roots: TreeNode[] = [];
  for (const n of map.values()) {
    // 父节点不在本次结果里（被删或未加载）时当根处理，避免整棵子树凭空消失
    const p = n.parent_id ? map.get(n.parent_id) : undefined;
    (p ? p.children! : roots).push(n);
  }
  roots.sort(byName);
  map.forEach((n) => n.children!.sort(byName));
  return roots;
}

// 深度优先展平，顺序与树一致（日历、统计这类要「按树序」的地方用）
export function flattenTree(nodes: TreeNode[]): Note[] {
  const out: Note[] = [];
  const walk = (list: TreeNode[]) => {
    for (const n of list) {
      const { children, ...note } = n;
      out.push(note as Note);
      if (children?.length) walk(children);
    }
  };
  walk(nodes);
  return out;
}
