import { defineStore } from 'pinia';
import { ref } from 'vue';
import { useWebSocket } from '../services/websocket';

export const useScanStore = defineStore('scan', () => {
  const results = ref([]);
  const status = ref('idle'); // idle, scanning, done
  const message = ref('');
  const progress = ref(0);
  const failedCount = ref(0);
  const summary = ref('');
  const phase = ref('idle'); // idle, main_scan, retry_scan, done
  const totalRetrying = ref(0);

  const { sendMessage, on } = useWebSocket();

  on('scan_results', (payload) => {
    // payload is now an array of results
    results.value.push(...payload);
  });

  on('scan_status', (payload) => {
    status.value = payload.status;
    message.value = payload.message;
    progress.value = payload.progress;
    failedCount.value = payload.failed || 0;
    phase.value = payload.phase || 'main_scan';
    totalRetrying.value = payload.total_retrying || 0;

    if (payload.status === 'done') {
      summary.value = payload.summary || '';
      phase.value = 'done';
    }
  });

  function startScan(domain, wordlist, dnsServers, scanOptions) {
    results.value = [];
    status.value = 'scanning';
    message.value = '正在开始扫描...';
    progress.value = 0;
    failedCount.value = 0;
    summary.value = '';
    phase.value = 'main_scan';
    totalRetrying.value = 0;

    const payload = {
      domain,
      concurrency: scanOptions.concurrency,
      adaptive: scanOptions.adaptive,
      maxQPS: scanOptions.maxQPS,
      enable_retry: scanOptions.enableRetry,
    };

    if (Array.isArray(wordlist)) {
      payload.wordlist = wordlist;
    } else {
      payload.wordlist_key = wordlist;
    }

    if (dnsServers && dnsServers.length > 0) {
      payload.dns_servers = dnsServers;
    }

    sendMessage('start_scan', payload);
  }

  function clearResults() {
    results.value = [];
  }

  return {
    results,
    status,
    message,
    progress,
    failedCount,
    summary,
    phase,
    totalRetrying,
    startScan,
    clearResults,
  };
});
