<script setup lang="ts">
// 日历面板（第 3 区块）。布局见 docs/ui.md，具体形态「暂时不确定」——这里先立起来：
// 按月网格标出每天创建的笔记，点某天在下方列出当天笔记，再点进编辑器。
// 用 created_at 而不是 updated_at：前者不会被编辑挪走，格子里的分布才稳定。
import { computed, ref } from 'vue';
import type { Note } from '../../api';

const props = defineProps<{
  notes: Note[];
  selectedId: string | null;
  onSelect: (n: Note) => void;
}>();

const WEEKDAYS = ['一', '二', '三', '四', '五', '六', '日'];

const cursor = ref(startOfMonth(new Date()));
const pickedDay = ref<string | null>(null);

function startOfMonth(d: Date) {
  return new Date(d.getFullYear(), d.getMonth(), 1);
}
function dayKey(ts: number) {
  const d = new Date(ts);
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}
function pad(n: number) {
  return String(n).padStart(2, '0');
}

// 日期 → 当天创建的笔记
const byDay = computed(() => {
  const m = new Map<string, Note[]>();
  for (const n of props.notes) {
    const k = dayKey(n.created_at);
    const list = m.get(k);
    if (list) list.push(n);
    else m.set(k, [n]);
  }
  return m;
});

const monthLabel = computed(() => `${cursor.value.getFullYear()} 年 ${cursor.value.getMonth() + 1} 月`);

// 6×7 网格，周一开头
const cells = computed(() => {
  const first = cursor.value;
  const offset = (first.getDay() + 6) % 7; // 周一 = 0
  const start = new Date(first);
  start.setDate(1 - offset);
  return Array.from({ length: 42 }, (_, i) => {
    const d = new Date(start);
    d.setDate(start.getDate() + i);
    return {
      date: d,
      key: `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`,
      inMonth: d.getMonth() === first.getMonth(),
      isToday: dayKey(Date.now()) === `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`,
    };
  });
});

const pickedNotes = computed(() => (pickedDay.value ? byDay.value.get(pickedDay.value) ?? [] : []));

function shiftMonth(n: number) {
  cursor.value = new Date(cursor.value.getFullYear(), cursor.value.getMonth() + n, 1);
  pickedDay.value = null;
}
function today() {
  cursor.value = startOfMonth(new Date());
  pickedDay.value = dayKey(Date.now());
}
function pick(key: string) {
  pickedDay.value = pickedDay.value === key ? null : key;
}
function countOn(key: string) {
  return byDay.value.get(key)?.length ?? 0;
}
function fmt(ts: number) {
  return new Date(ts).toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' });
}
</script>

<template>
  <div class="panel">
    <div class="header">
      <span class="title">日历</span>
      <div class="nav">
        <button class="nav-btn" title="上一月" @click="shiftMonth(-1)"><Icon icon="mdi:chevron-left" width="16" height="16" /></button>
        <button class="today" title="回到今天" @click="today">{{ monthLabel }}</button>
        <button class="nav-btn" title="下一月" @click="shiftMonth(1)"><Icon icon="mdi:chevron-right" width="16" height="16" /></button>
      </div>
    </div>

    <div class="weekdays">
      <span v-for="w in WEEKDAYS" :key="w">{{ w }}</span>
    </div>

    <div class="grid">
      <button
        v-for="c in cells"
        :key="c.key"
        class="cell"
        :class="{ out: !c.inMonth, today: c.isToday, picked: pickedDay === c.key, has: countOn(c.key) > 0 }"
        @click="pick(c.key)"
      >
        <span class="day">{{ c.date.getDate() }}</span>
        <span v-if="countOn(c.key)" class="dot">{{ countOn(c.key) }}</span>
      </button>
    </div>

    <div class="day-notes">
      <div v-if="!pickedDay" class="empty">点一天看看</div>
      <div v-else-if="pickedNotes.length === 0" class="empty">{{ pickedDay }} 没有新建笔记</div>
      <ul v-else>
        <li v-for="n in pickedNotes" :key="n.id">
          <button class="note" :class="{ active: selectedId === n.id }" @click="onSelect(n)">
            <span class="note-title">{{ n.title || '(无标题)' }}</span>
            <span class="note-time">{{ fmt(n.created_at) }}</span>
          </button>
        </li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-muted);
}
.title {
  font-size: 12px;
  font-weight: 600;
  color: var(--fg-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.nav {
  display: flex;
  align-items: center;
  gap: 2px;
}
.nav-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  color: var(--fg-muted);
  background: none;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
.nav-btn:hover {
  background: var(--btn-hover-bg);
  color: var(--fg-default);
}
.today {
  padding: 2px 6px;
  font-size: 12px;
  color: var(--fg-muted);
  background: none;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
.today:hover {
  background: var(--btn-hover-bg);
  color: var(--fg-default);
}
.weekdays,
.grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
  padding: 0 8px;
}
.weekdays {
  padding-top: 8px;
  font-size: 11px;
  color: var(--fg-subtle);
  text-align: center;
}
.cell {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  aspect-ratio: 1;
  padding: 0;
  font-size: 12px;
  color: var(--fg-default);
  background: none;
  border: 1px solid transparent;
  border-radius: 6px;
  cursor: pointer;
}
.cell:hover {
  background: var(--btn-hover-bg);
}
.cell.out {
  color: var(--fg-subtle);
}
.cell.today {
  border-color: var(--border-default);
}
.cell.picked {
  background: var(--accent-subtle);
  color: var(--accent-fg);
  border-color: var(--accent-emphasis);
}
.cell.has .day {
  font-weight: 600;
}
.dot {
  position: absolute;
  bottom: 2px;
  font-size: 9px;
  line-height: 1;
  color: var(--accent-fg);
}
.day-notes {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  margin-top: 8px;
  border-top: 1px solid var(--border-muted);
}
.day-notes ul {
  list-style: none;
  margin: 0;
  padding: 4px 8px;
}
.note {
  display: flex;
  align-items: baseline;
  gap: 8px;
  width: 100%;
  padding: 6px 8px;
  color: var(--fg-default);
  background: none;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  text-align: left;
  cursor: pointer;
}
.note:hover {
  background: var(--btn-hover-bg);
}
.note.active {
  background: var(--accent-subtle);
  color: var(--accent-fg);
}
.note-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.note-time {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--fg-muted);
}
.empty {
  padding: 16px;
  font-size: 13px;
  color: var(--fg-muted);
  text-align: center;
}
</style>
