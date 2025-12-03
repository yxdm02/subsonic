<template>
  <div id="app-container">
    <header class="app-header">
      <h1>SubSonic</h1>
      <p>高性能实时子域名扫描平台</p>
    </header>

    <main>
      <div class="card">
        <ScanForm />
      </div>
      <div class="card">
        <StatusBar />
        <ResultsTable />
      </div>
    </main>

    <footer class="app-footer">
      <p>&copy; 2025 SubSonic Project. All Rights Reserved.</p>
    </footer>
  </div>
</template>

<script setup>
import ScanForm from './components/ScanForm.vue';
import StatusBar from './components/StatusBar.vue';
import ResultsTable from './components/ResultsTable.vue';
</script>

<style scoped>
#app-container {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.app-header {
  text-align: center;
  padding: 1rem 0;
  border-bottom: 1px solid var(--border-color);
  margin-bottom: 2rem;
}

.app-header h1 {
  margin: 0;
  font-size: 2.5rem;
}

.app-header p {
  margin: 0;
  color: var(--secondary-color);
}

main {
  flex: 1;
}

.app-footer {
  text-align: center;
  padding: 1rem 0;
  margin-top: 2rem;
  border-top: 1px solid var(--border-color);
  color: var(--secondary-color);
  font-size: 0.9rem;
}
</style>
