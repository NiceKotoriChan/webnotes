<script setup lang="ts">
// 工具栏：往 Markdown 源码里插语法。它不是富文本工具栏 —— 每个按钮只是
// 在选区上做一次纯文本变换，真正的变换逻辑在 lib/md.ts。
import type { EditorMode, ToolbarAction } from '../../lib/editor';

defineProps<{ mode: EditorMode }>();
const emit = defineEmits<{
  (e: 'action', a: ToolbarAction): void;
  (e: 'set-mode', m: EditorMode): void;
}>();

const btn = 'tool-btn';
const MODES: { id: EditorMode; icon: string; title: string }[] = [
  { id: 'edit', icon: 'pencil-outline', title: '只看源码' },
  { id: 'split', icon: 'view-split-vertical', title: '源码 + 预览' },
  { id: 'preview', icon: 'eye-outline', title: '只看预览' },
];
</script>

<template>
  <div class="bar">
    <button :class="btn" title="加粗" @click="emit('action', 'bold')"><Icon icon="mdi:format-bold" width="16" height="16" /></button>
    <button :class="btn" title="斜体" @click="emit('action', 'italic')"><Icon icon="mdi:format-italic" width="16" height="16" /></button>
    <button :class="btn" title="删除线" @click="emit('action', 'strike')"><Icon icon="mdi:format-strikethrough" width="16" height="16" /></button>

    <span class="sep"></span>

    <button :class="btn" title="一级标题" @click="emit('action', 'h1')"><Icon icon="mdi:format-header-1" width="16" height="16" /></button>
    <button :class="btn" title="二级标题" @click="emit('action', 'h2')"><Icon icon="mdi:format-header-2" width="16" height="16" /></button>
    <button :class="btn" title="三级标题" @click="emit('action', 'h3')"><Icon icon="mdi:format-header-3" width="16" height="16" /></button>

    <span class="sep"></span>

    <button :class="btn" title="无序列表" @click="emit('action', 'ul')"><Icon icon="mdi:format-list-bulleted" width="16" height="16" /></button>
    <button :class="btn" title="有序列表" @click="emit('action', 'ol')"><Icon icon="mdi:format-list-numbered" width="16" height="16" /></button>
    <button :class="btn" title="任务列表" @click="emit('action', 'task')"><Icon icon="mdi:format-list-checks" width="16" height="16" /></button>
    <button :class="btn" title="引用" @click="emit('action', 'quote')"><Icon icon="mdi:format-quote-close" width="16" height="16" /></button>

    <span class="sep"></span>

    <button :class="btn" title="行内代码" @click="emit('action', 'inlineCode')"><Icon icon="mdi:code-tags" width="16" height="16" /></button>
    <button :class="btn" title="代码块" @click="emit('action', 'codeBlock')"><Icon icon="mdi:code-braces" width="16" height="16" /></button>
    <button :class="btn" title="链接" @click="emit('action', 'link')"><Icon icon="mdi:link" width="16" height="16" /></button>

    <span class="sep"></span>

    <button :class="btn" title="上传并插入图片" @click="emit('action', 'image')"><Icon icon="mdi:image" width="16" height="16" /></button>
    <button :class="btn" title="上传并插入附件" @click="emit('action', 'attach')"><Icon icon="mdi:paperclip" width="16" height="16" /></button>

    <span class="spacer"></span>

    <button
      v-for="m in MODES"
      :key="m.id"
      :class="[btn, mode === m.id ? 'active' : '']"
      :title="m.title"
      @click="emit('set-mode', m.id)"
    >
      <Icon :icon="'mdi:' + m.icon" width="16" height="16" />
    </button>
  </div>
</template>

<style scoped>
.bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 2px;
  padding: 4px 8px;
  border-bottom: 1px solid var(--border-muted);
  background: var(--canvas-subtle);
  user-select: none;
}
.tool-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  color: var(--fg-muted);
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}
.tool-btn:hover {
  color: var(--fg-default);
  background: var(--btn-hover-bg);
}
.tool-btn.active {
  color: var(--accent-fg);
  background: var(--accent-subtle);
}
.sep {
  width: 1px;
  height: 16px;
  margin: 0 4px;
  background: var(--border-muted);
}
.spacer {
  flex: 1;
}
</style>
