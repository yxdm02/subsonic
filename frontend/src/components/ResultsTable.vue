<template>
  <div class="results-container">
    <div class="results-header">
      <h3>扫描结果 ({{ store.results.length }})</h3>
      <div>
        <button @click="exportResults" v-if="store.results.length > 0" class="export-btn">
          导出 CSV
        </button>
        <button @click="clearResults" v-if="store.results.length > 0" class="clear-btn">
          清空结果
        </button>
      </div>
    </div>

    <div v-if="store.results.length === 0" class="no-results">
      <p>暂无结果。开始一次新的扫描来发现子域名。</p>
    </div>

    <div class="table-wrapper" v-else>
      <table>
        <thead>
          <tr>
            <th>子域名</th>
            <th>IP 地址</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="result in store.results" :key="result.Subdomain">
            <td>{{ result.Subdomain }}</td>
            <td>{{ result.IPAddress }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { useScanStore } from '../stores/scan';

const store = useScanStore();

const clearResults = () => {
  store.clearResults();
};
const exportResults = () => {
  const results = store.results;
  if (results.length === 0) {
    return;
  }

  const header = '"Subdomain","IPAddress"\n';
  const csvContent = results.map(row => `"${row.Subdomain}","${row.IPAddress}"`).join('\n');
  const fullCsv = header + csvContent;

  const blob = new Blob([fullCsv], { type: 'text/csv;charset=utf-8;' });
  const link = document.createElement('a');
  const url = URL.createObjectURL(blob);
  link.setAttribute('href', url);
  link.setAttribute('download', 'subsonic_results.csv');
  link.style.visibility = 'hidden';
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
};
</script>

<style scoped>
.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.results-header h3 {
  margin: 0;
  color: #333;
}

.export-btn {
  background-color: var(--success-color);
  color: white;
  padding: 8px 12px;
  font-size: 0.9rem;
  margin-right: 10px;
}
.clear-btn {
  background-color: var(--secondary-color);
  color: white;
  padding: 8px 12px;
  font-size: 0.9rem;
}

.clear-btn:hover {
  background-color: #5a6268;
}

.table-wrapper {
  overflow-x: auto;
  max-height: 500px;
  overflow-y: auto;
  border: 1px solid var(--border-color);
  border-radius: 6px;
}

table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}

th, td {
  padding: 12px 15px;
  border-bottom: 1px solid var(--border-color);
}

thead th {
  background-color: #f8f9fa;
  font-weight: bold;
  position: sticky;
  top: 0;
}

tbody tr:nth-of-type(even) {
  background-color: #f8f9fa;
}

tbody tr:hover {
  background-color: #e9ecef;
}

.no-results {
  text-align: center;
  padding: 3rem;
  color: var(--secondary-color);
  border: 2px dashed var(--border-color);
  border-radius: 6px;
}
</style>
