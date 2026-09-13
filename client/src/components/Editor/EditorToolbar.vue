<script setup lang="ts">
// 所见即所得工具栏：加粗/斜体/删除线/标题/列表/任务/引用/代码/链接/图片/附件/撤销重做
import { reactive, onMounted, onBeforeUnmount } from 'vue';
import type { Editor } from '@tiptap/core';
import { commands } from '../../lib/tiptap';

const props = defineProps<{
  editor: Editor;
  onPickImage: () => void;
  onPickAttach: () => void;
}>();

const btn =
  'inline-flex h-7 w-7 items-center justify-center rounded-md text-[13px] text-fg-muted hover:bg-btn-hover-bg hover:text-fg-default disabled:opacity-40';
const active = 'bg-accent-subtle text-accent-fg';
const levels = [1, 2, 3] as const;

const state = reactive({
  bold: false,
  italic: false,
  strike: false,
  h1: false,
  h2: false,
  h3: false,
  bulletList: false,
  orderedList: false,
  taskList: false,
  blockquote: false,
  code: false,
  codeBlock: false,
  link: false,
  canUndo: false,
  canRedo: false,
});

function refresh() {
  const e = props.editor;
  state.bold = e.isActive('bold');
  state.italic = e.isActive('italic');
  state.strike = e.isActive('strike');
  state.h1 = e.isActive('heading', { level: 1 });
  state.h2 = e.isActive('heading', { level: 2 });
  state.h3 = e.isActive('heading', { level: 3 });
  state.bulletList = e.isActive('bulletList');
  state.orderedList = e.isActive('orderedList');
  state.taskList = e.isActive('taskList');
  state.blockquote = e.isActive('blockquote');
  state.code = e.isActive('code');
  state.codeBlock = e.isActive('codeBlock');
  state.link = e.isActive('link');
  state.canUndo = e.can().undo();
  state.canRedo = e.can().redo();
}

onMounted(() => {
  refresh();
  props.editor.on('transaction', refresh);
  props.editor.on('selectionUpdate', refresh);
});
onBeforeUnmount(() => {
  props.editor.off('transaction', refresh);
  props.editor.off('selectionUpdate', refresh);
});

function heading(level: 1 | 2 | 3) {
  commands.heading(props.editor, level);
}
function activeForLevel(level: number) {
  return level === 1 ? state.h1 : level === 2 ? state.h2 : state.h3;
}
function setLink() {
  const e = props.editor;
  const prev = e.getAttributes('link').href as string | undefined;
  const url = prompt('链接地址', prev ?? '');
  if (url === null) return;
  if (url.trim() === '') e.chain().focus().unsetLink().run();
  else e.chain().focus().setLink({ href: url.trim() }).run();
}
</script>

<template>
  <div class="flex flex-wrap items-center gap-0.5 border-b border-muted bg-subtle px-2 py-1 select-none">
    <button :class="[btn, state.bold ? active : '']" title="加粗 (Ctrl+B)" @click="commands.bold(editor)">
      <Icon icon="mdi:format-bold" width="16" height="16" />
    </button>
    <button :class="[btn, state.italic ? active : '']" title="斜体 (Ctrl+I)" @click="commands.italic(editor)">
      <Icon icon="mdi:format-italic" width="16" height="16" />
    </button>
    <button :class="[btn, state.strike ? active : '']" title="删除线" @click="commands.strike(editor)">
      <Icon icon="mdi:format-strikethrough" width="16" height="16" />
    </button>

    <span class="mx-1 h-4 w-px bg-border-muted"></span>

    <button v-for="level in levels" :key="level" :class="[btn, activeForLevel(level) ? active : '']"
      :title="'标题 ' + level" @click="heading(level)">
      <Icon :icon="'mdi:format-header-' + level" width="16" height="16" />
    </button>

    <span class="mx-1 h-4 w-px bg-border-muted"></span>

    <button :class="[btn, state.bulletList ? active : '']" title="无序列表" @click="commands.bulletList(editor)">
      <Icon icon="mdi:format-list-bulleted" width="16" height="16" />
    </button>
    <button :class="[btn, state.orderedList ? active : '']" title="有序列表" @click="commands.orderedList(editor)">
      <Icon icon="mdi:format-list-numbered" width="16" height="16" />
    </button>
    <button :class="[btn, state.taskList ? active : '']" title="任务列表" @click="commands.taskList(editor)">
      <Icon icon="mdi:format-list-checks" width="16" height="16" />
    </button>
    <button :class="[btn, state.blockquote ? active : '']" title="引用" @click="commands.blockquote(editor)">
      <Icon icon="mdi:format-quote-close" width="16" height="16" />
    </button>

    <span class="mx-1 h-4 w-px bg-border-muted"></span>

    <button :class="[btn, state.code ? active : '']" title="行内代码" @click="commands.code(editor)">
      <Icon icon="mdi:code-tags" width="16" height="16" />
    </button>
    <button :class="[btn, state.codeBlock ? active : '']" title="代码块" @click="commands.codeBlock(editor)">
      <Icon icon="mdi:code-braces" width="16" height="16" />
    </button>

    <span class="mx-1 h-4 w-px bg-border-muted"></span>

    <button :class="[btn, state.link ? active : '']" title="插入/编辑链接" @click="setLink">
      <Icon icon="mdi:link" width="16" height="16" />
    </button>
    <button :class="btn" title="插入图片" @click="onPickImage">
      <Icon icon="mdi:image" width="16" height="16" />
    </button>
    <button :class="btn" title="插入附件" @click="onPickAttach">
      <Icon icon="mdi:paperclip" width="16" height="16" />
    </button>

    <span class="mx-1 h-4 w-px bg-border-muted"></span>

    <button :class="btn" title="撤销 (Ctrl+Z)" :disabled="!state.canUndo" @click="commands.undo(editor)">
      <Icon icon="mdi:undo" width="16" height="16" />
    </button>
    <button :class="btn" title="重做 (Ctrl+Y)" :disabled="!state.canRedo" @click="commands.redo(editor)">
      <Icon icon="mdi:redo" width="16" height="16" />
    </button>
  </div>
</template>
