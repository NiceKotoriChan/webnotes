// 笔记树：把扁平列表组装成树，按 mtime 降序
import type { Note } from '../api';

export type TreeNode = Note & { children?: TreeNode[] };

export function buildTree(notes: Note[]): TreeNode[] {
  const map = new Map<string, TreeNode>();
  for (const n of notes) map.set(n.id, { ...n, children: [] });
  const roots: TreeNode[] = [];
  for (const n of map.values()) {
    const p = n.parent_id && map.get(n.parent_id);
    (p ? p.children! : roots).push(n);
  }
  const byMtime = (a: TreeNode, b: TreeNode) => b.mtime - a.mtime;
  roots.sort(byMtime);
  map.forEach((n) => n.children!.sort(byMtime));
  return roots;
}
