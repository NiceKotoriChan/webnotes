<script setup lang="ts">
// Markdown 编辑器（正文区）：左侧源码框 + 右侧实时预览，模式由外面控制。
// 正文的真相源就是 Markdown，没有中间态 —— 所以这里不做 md↔HTML 往返，只单向渲染。
import { computed, nextTick, ref } from 'vue';
import { insertAt, renderMarkdown, type TextSel } from '../../lib/md';
import type { EditorMode } from '../../lib/editor';

const props = defineProps<{
  modelValue: string;
  mode: EditorMode;
}>();
const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>();

const ta = ref<HTMLTextAreaElement>();
const html = computed(() => renderMarkdown(props.modelValue));

function currentSelection(): TextSel {
  const el = ta.value;
  const start = el?.selectionStart ?? props.modelValue.length;
  const end = el?.selectionEnd ?? start;
  return { text: props.modelValue, start, end };
}

async function apply(next: TextSel) {
  emit('update:modelValue', next.text);
  await nextTick();
  const el = ta.value;
  if (!el) return;
  el.focus();
  el.setSelectionRange(next.start, next.end);
}

// 工具栏用这两个：都是纯文本变换，不碰 DOM，见 lib/md.ts
const transform = (fn: (s: TextSel) => TextSel) => apply(fn(currentSelection()));
const insert = (snippet: string, cursorOffset?: number) =>
  transform((s) => insertAt(s, snippet, cursorOffset));

function onInput(e: Event) {
  emit('update:modelValue', (e.target as HTMLTextAreaElement).value);
}

defineExpose({ transform, insert, focus: () => ta.value?.focus() });
</script>

<template>
  <div class="md-editor" :class="mode">
    <textarea
      v-if="mode !== 'preview'"
      ref="ta"
      class="md-source"
      :value="modelValue"
      spellcheck="false"
      placeholder="用 Markdown 写点什么…"
      @input="onInput"
    ></textarea>
    <!-- markdown-body 提供 github-markdown-css 的版式；html 已在 lib/md.ts 里关掉裸 HTML -->
    <div v-if="mode !== 'edit'" class="markdown-body md-preview" v-html="html"></div>
  </div>
</template>

<style scoped>
.md-editor {
  display: flex;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.md-editor.edit,
.md-editor.preview {
  display: block;
  overflow-y: auto;
}
.md-source {
  flex: 1;
  min-width: 0;
  border-right: 1px solid var(--border-muted);
}
.md-editor.edit .md-source,
.md-editor.preview .md-preview {
  border-right: none;
}
.md-editor.edit .md-source {
  height: 100%;
  overflow-y: auto;
}
.md-preview {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
}
</style>
