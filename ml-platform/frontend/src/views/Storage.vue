<template>
  <div class="storage-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">对象存储</h1>
        <p class="page-subtitle">管理训练数据和模型输出</p>
      </div>
    </div>

    <div class="storage-container">
      <el-card class="storage-card">
        <template #header>
          <div class="card-header">
            <span>文件管理</span>
            <el-button type="primary" size="small">
              <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                <polyline points="17 8 12 3 7 8"/>
                <line x1="12" y1="3" x2="12" y2="15"/>
              </svg>
              上传文件
            </el-button>
          </div>
        </template>

        <div class="path-navigator">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/storage' }">根目录</el-breadcrumb-item>
          </el-breadcrumb>
        </div>

        <el-table :data="files" stripe class="files-table">
          <el-table-column label="名称" min-width="300">
            <template #default="{ row }">
              <div class="file-name-cell" @click="handleClick(row)">
                <svg v-if="row.isDir" class="file-icon folder" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M10 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/>
                </svg>
                <svg v-else class="file-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
                </svg>
                <span class="file-name">{{ row.name }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="size" label="大小" width="120">
            <template #default="{ row }">
              {{ row.isDir ? '-' : formatSize(row.size) }}
            </template>
          </el-table-column>
          <el-table-column prop="lastModified" label="修改时间" width="180">
            <template #default="{ row }">
              {{ formatTime(row.lastModified) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <div class="action-buttons">
                <el-button size="small" link type="primary" @click="handleDownload(row)">下载</el-button>
                <el-button size="small" link type="danger" @click="handleDelete(row)">删除</el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'

const files = ref([
  { name: 'train_data', size: 0, lastModified: new Date(), isDir: true },
  { name: 'models', size: 0, lastModified: new Date(), isDir: true },
  { name: 'output', size: 0, lastModified: new Date(), isDir: true },
  { name: 'dataset.csv', size: 1048576, lastModified: new Date(), isDir: false },
  { name: 'requirements.txt', size: 256, lastModified: new Date(), isDir: false }
])

const formatSize = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatTime = (time) => {
  return new Date(time).toLocaleString('zh-CN')
}

const handleClick = (row) => {
  if (row.isDir) {
    ElMessage.info('导航到: ' + row.name)
  }
}

const handleDownload = (row) => {
  ElMessage.success('开始下载: ' + row.name)
}

const handleDelete = (row) => {
  ElMessage.warning('删除功能待实现')
}
</script>

<style scoped>
.storage-page {
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 24px;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
}

.storage-container {
  margin-top: 0;
}

.storage-card {
  border-radius: var(--radius-lg);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.btn-icon {
  width: 16px;
  height: 16px;
  margin-right: 6px;
}

.path-navigator {
  padding: 12px 0;
  border-bottom: 1px solid var(--border-color);
  margin-bottom: 16px;
}

.files-table {
  margin-top: 0;
}

.file-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
}

.file-name-cell:hover .file-name {
  color: var(--primary-color);
}

.file-icon {
  width: 20px;
  height: 20px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.file-icon.folder {
  color: #f59e0b;
}

.file-name {
  font-weight: 500;
  transition: color 0.2s;
}

.action-buttons {
  display: flex;
  gap: 8px;
}
</style>