// 侧栏区块的清单。活动栏按它渲染按钮，工作区按它选面板，
// 两边共用一份定义，加一个区块只需要改这里。
export type Panel = 'tree' | 'calendar' | 'tags' | 'assets' | 'trash';

export interface PanelDef {
  id: Panel;
  icon: string; // 不带 mdi: 前缀
  title: string;
}

export const PANELS: PanelDef[] = [
  { id: 'tree', icon: 'file-tree', title: '文档树' },
  { id: 'calendar', icon: 'calendar-month-outline', title: '日历' },
  { id: 'tags', icon: 'tag-multiple-outline', title: '标签' },
  { id: 'assets', icon: 'paperclip', title: '附件' },
  { id: 'trash', icon: 'delete-outline', title: '回收站' },
];

const KEY = 'webnotes:panel';

export function initialPanel(): Panel {
  const v = localStorage.getItem(KEY) as Panel | null;
  return v && PANELS.some((p) => p.id === v) ? v : 'tree';
}

export function rememberPanel(p: Panel) {
  localStorage.setItem(KEY, p);
}
