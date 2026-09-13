// 拖拽状态：TreeList 递归实例间共享（跨组件实例的模块级状态）
let draggedId: string | null = null;
export function setDraggedId(id: string | null) {
  draggedId = id;
}
export function getDraggedId() {
  return draggedId;
}
