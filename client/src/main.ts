import './app.css';
import { createApp } from 'vue';
import { Icon } from '@iconify/vue/offline';
import App from './App.vue';
import { registerSW } from './lib/sw';
import './lib/icons'; // 初始化即加载全量 mdi 图标集
import { loadIconRules } from './lib/rules';

const app = createApp(App);
app.component('Icon', Icon);
app.mount('#app');
registerSW();

// 图标匹配规则由后端下发；拉不到就退回兜底图标，不阻塞启动
loadIconRules();
