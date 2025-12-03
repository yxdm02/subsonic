<template>
  <div class="scan-form-container">
    <h2>扫描配置</h2>
    <form @submit.prevent="submitScan">
      <div class="form-grid">
        <div class="form-group full-width">
          <label for="domain">目标域名:</label>
          <input type="text" id="domain" v-model="domain" placeholder="example.com" required />
        </div>

        <div class="form-group">
          <label>字典选择:</label>
          <div class="wordlist-options">
            <select v-model="wordlistSource" @change="wordlistSelectionChanged">
              <option value="common_speak">内置字典 (common_speak)</option>
              <option value="custom_file">自定义文件</option>
            </select>
            <input type="file" ref="fileInput" @change="handleFileUpload" v-if="wordlistSource === 'custom_file'" class="file-input"/>
          </div>
           <span v-if="customFileStatus" class="upload-status">{{ customFileStatus }}</span>
        </div>

        <div class="form-group">
          <label for="dns-servers">DNS 服务器 (可选):</label>
          <input type="text" id="dns-servers" v-model="dnsServers" placeholder="8.8.8.8, 1.1.1.1" />
        </div>

        <div class="form-group">
          <label for="concurrency">并发数:</label>
          <input type="number" id="concurrency" v-model.number="scanOptions.concurrency" min="1" :disabled="scanOptions.adaptive" />
        </div>
        
        <div class="form-group">
          <label for="max-qps">最大 QPS (0为不限制):</label>
          <input type="number" id="max-qps" v-model.number="scanOptions.maxQPS" min="0" />
        </div>

        <div class="form-group adaptive-mode full-width">
          <div>
            <input type="checkbox" id="adaptive" v-model="scanOptions.adaptive" />
            <label for="adaptive">自适应模式 (将忽略并发数)</label>
          </div>
          <div>
            <input type="checkbox" id="enableRetry" v-model="scanOptions.enableRetry" />
            <label for="enableRetry">开启失败重试</label>
          </div>
        </div>
      </div>

      <button type="submit" :disabled="isScanning || (wordlistSource === 'custom_file' && !uploadedFileKey)">
        {{ isScanning ? '扫描中...' : '开始扫描' }}
      </button>
    </form>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue';
import { useScanStore } from '../stores/scan';

const store = useScanStore();
const domain = ref('');
const wordlistSource = ref('common_speak');
const dnsServers = ref('');
const fileInput = ref(null);
const uploadedFileKey = ref('');
const customFileStatus = ref('');

const scanOptions = ref({
  concurrency: 100,
  adaptive: false,
  maxQPS: 1000, // Default QPS
  enableRetry: true, // Default to true
});

const isScanning = computed(() => store.status === 'scanning');

const wordlistSelectionChanged = () => {
  uploadedFileKey.value = '';
  customFileStatus.value = '';
  if (fileInput.value) {
    fileInput.value.value = '';
  }
};

const handleFileUpload = async (event) => {
  const file = event.target.files[0];
  if (!file) {
    return;
  }

  customFileStatus.value = '正在上传...';
  uploadedFileKey.value = '';

  const formData = new FormData();
  formData.append('wordlist', file);

  try {
    const response = await fetch('/api/upload-wordlist', {
      method: 'POST',
      body: formData,
    });

    if (!response.ok) {
      throw new Error(`Upload failed with status: ${response.status}`);
    }

    const result = await response.json();
    uploadedFileKey.value = result.wordlist_key;
    customFileStatus.value = `上传成功: ${file.name}`;
  } catch (error) {
    console.error('File upload error:', error);
    customFileStatus.value = '上传失败，请检查后台日志。';
  }
};

const submitScan = () => {
  let wordlistPayload;
  if (wordlistSource.value === 'custom_file') {
    if (!uploadedFileKey.value) {
      alert('请先上传一个自定义字典文件。');
      return;
    }
    wordlistPayload = uploadedFileKey.value;
  } else {
    wordlistPayload = wordlistSource.value;
  }

  const dnsServersArray = dnsServers.value.split(',').map(s => s.trim()).filter(Boolean);
  store.startScan(domain.value, wordlistPayload, dnsServersArray, scanOptions.value);
};
</script>

<style scoped>
.scan-form-container {
  width: 100%;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
}

.form-group.full-width {
  grid-column: 1 / -1;
}

.form-group label {
  margin-bottom: 8px;
  font-weight: bold;
  color: #555;
}

.wordlist-options {
  display: flex;
  align-items: center;
  gap: 10px;
}

.file-input {
  flex-grow: 1;
}

.upload-status {
  margin-top: 8px;
  font-size: 0.9rem;
  color: var(--secondary-color);
}

.adaptive-mode {
  flex-direction: row;
  align-items: center;
  justify-content: flex-start;
  gap: 20px; /* Add some space between the checkboxes */
  padding-top: 1rem;
}

.adaptive-mode input {
  width: auto;
  margin-right: 10px;
}

button {
  width: 100%;
  padding: 12px;
  background-color: var(--primary-color);
  color: white;
}

button:hover {
  background-color: var(--primary-hover-color);
}
</style>
