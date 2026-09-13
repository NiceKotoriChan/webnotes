/// <reference types="vite/client" />

// 让本文件成为模块，使下方成为对 vue 的“模块增强”而非覆盖
export {};

declare module 'vue' {
  export interface GlobalComponents {
    Icon: (typeof import('@iconify/vue/offline'))['Icon'];
  }
}
