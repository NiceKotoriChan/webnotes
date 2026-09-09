<script lang="ts">
  import { marked } from 'marked';
  import { themeStore } from '../stores';
  // github-markdown-css 提供明暗两套皮肤，按主题切换 <link>
  import lightCss from 'github-markdown-css/github-markdown.css?url';
  import darkCss from 'github-markdown-css/github-markdown-dark.css?url';

  // 渲染用 marked（GFM），样式用 github-markdown-css，不自己造轮子
  marked.setOptions({ gfm: true, breaks: true });

  let { content = '' }: { content?: string } = $props();

  const html = $derived(marked.parse(content ?? '', { async: false }) as string);
  const themeHref = $derived($themeStore === 'dark' ? darkCss : lightCss);
</script>

<svelte:head>
  <link rel="stylesheet" href={themeHref} />
</svelte:head>

<div class="markdown-body">{@html html}</div>
