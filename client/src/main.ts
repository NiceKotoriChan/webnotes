import './app.css';
import { createApp } from 'vue';
import { Icon } from '@iconify/vue/offline';
import App from './App.vue';
import { registerSW } from './lib/sw';
import './lib/icons'; // 注册首屏子集图标

const app = createApp(App);
app.component('Icon', Icon);
app.mount('#app');
registerSW();
