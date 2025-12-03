<template>
  <div class="status-bar-container" v-if="store.status !== 'idle'">
    <div class="status-bar" :class="statusClass">
      <div class="status-message">
        <span v-if="store.phase === 'main_scan'">{{ store.message }}</span>
        <span v-else-if="store.phase === 'retry_scan'">重试失败域名... ({{ store.failedCount }} / {{ store.totalRetrying }})</span>
        <span v-else>{{ store.message }}</span>
      </div>
      <div class="progress-bar">
        <div class="progress" :style="{ width: progressPercentage }"></div>
      </div>
      <div v-if="store.status === 'done' && store.summary" class="summary-message">
        {{ store.summary }}
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { useScanStore } from '../stores/scan';

const store = useScanStore();

const statusClass = computed(() => {
  switch (store.status) {
    case 'scanning':
      return 'status-scanning';
    case 'done':
      return 'status-done';
    default:
      return '';
  }
});

const progressPercentage = computed(() => {
  return `${store.progress * 100}%`;
});
</script>

<style scoped>
.status-bar-container {
  margin-bottom: 1.5rem;
}

.status-bar {
  padding: 1rem;
  border-radius: 6px;
  color: white;
  transition: background-color 0.3s ease;
}

.status-scanning {
  background-color: var(--primary-color);
}

.status-done {
  background-color: var(--success-color);
}

.status-message {
  margin-bottom: 0.5rem;
  font-weight: bold;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.progress-bar {
  width: 100%;
  background-color: rgba(255, 255, 255, 0.3);
  border-radius: 4px;
  overflow: hidden;
  height: 10px;
}

.progress {
  height: 100%;
  background-color: #ffffff;
  border-radius: 4px;
  transition: width 0.2s ease-in-out;
}

.summary-message {
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px solid rgba(255, 255, 255, 0.3);
  font-size: 0.9rem;
  text-align: center;
}
</style>
