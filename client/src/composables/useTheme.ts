import { ref, watch } from 'vue';

export type Theme = 'light' | 'dark';
const THEME_KEY = 'webnotes:theme';

function initialTheme(): Theme {
  const t = localStorage.getItem(THEME_KEY);
  if (t === 'dark' || t === 'light') return t;
  return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

const theme = ref<Theme>(initialTheme());
watch(
  theme,
  (t) => {
    document.documentElement.classList.toggle('dark', t === 'dark');
    localStorage.setItem(THEME_KEY, t);
  },
  { immediate: true },
);

export function useTheme() {
  const toggle = () => (theme.value = theme.value === 'dark' ? 'light' : 'dark');
  return { theme, toggle };
}
