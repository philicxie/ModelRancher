<template>
  <div class="task-detail-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <el-button @click="$router.push('/tasks')" text>
          <svg class="back-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M19 12H5"/>
            <path d="M12 19l-7-7 7-7"/>
          </svg>
          返回任务列表
        </el-button>
        <h1 class="page-title">{{ task?.name || '任务详情' }}</h1>
        <div class="task-status-badge" :class="task?.status" v-if="task">
          <span class="status-dot"></span>
          {{ getStatusLabel(task.status) }}
        </div>
      </div>
      <div class="header-actions" v-if="task">
        <el-button
          v-if="task.status === 'pending' || task.status === 'running'"
          type="danger"
          @click="handleCancel">
          <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="15" y1="9" x2="9" y2="15"/>
            <line x1="9" y1="9" x2="15" y2="15"/>
          </svg>
          取消任务
        </el-button>
      </div>
    </div>

    <div class="detail-content" v-loading="loading">
      <template v-if="task">
        <!-- 基本信息卡片 -->
        <div class="detail-grid">
          <el-card class="detail-card">
            <template #header>
              <div class="card-header">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
                  <polyline points="14 2 14 8 20 8"/>
                </svg>
                <span>基本信息</span>
              </div>
            </template>
            <div class="info-list">
              <div class="info-item">
                <span class="info-label">任务ID</span>
                <span class="info-value mono">{{ task.id }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">任务名称</span>
                <span class="info-value">{{ task.name }}</span>
              </div>
              <div class="info-item" v-if="task.description">
                <span class="info-label">任务描述</span>
                <span class="info-value">{{ task.description }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">创建时间</span>
                <span class="info-value">{{ formatTime(task.created_at) }}</span>
              </div>
              <div class="info-item" v-if="task.started_at">
                <span class="info-label">开始时间</span>
                <span class="info-value">{{ formatTime(task.started_at) }}</span>
              </div>
              <div class="info-item" v-if="task.finished_at">
                <span class="info-label">完成时间</span>
                <span class="info-value">{{ formatTime(task.finished_at) }}</span>
              </div>
            </div>
          </el-card>

          <el-card class="detail-card">
            <template #header>
              <div class="card-header">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                  <circle cx="8.5" cy="8.5" r="1.5"/>
                  <polyline points="21 15 16 10 5 21"/>
                </svg>
                <span>环境配置</span>
              </div>
            </template>
            <div class="info-list">
              <div class="info-item">
                <span class="info-label">Docker镜像</span>
                <div class="image-badge">
                  <span class="image-name">{{ task.image.split(':')[0] }}</span>
                  <span class="image-tag">{{ task.image.split(':')[1] || 'latest' }}</span>
                </div>
              </div>
              <div class="info-item">
                <span class="info-label">容器ID</span>
                <span class="info-value mono" v-if="task.container_id">{{ task.container_id }}</span>
                <span class="info-value text-secondary" v-else>-</span>
              </div>
            </div>
          </el-card>

          <el-card class="detail-card full-width">
            <template #header>
              <div class="card-header">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="4 17 10 11 4 5"/>
                  <line x1="12" y1="19" x2="20" y2="19"/>
                </svg>
                <span>执行命令</span>
              </div>
            </template>
            <div class="command-block">
              <code>{{ task.command }}</code>
              <el-button size="small" text class="copy-btn" @click="copyCommand">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                  <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                </svg>
              </el-button>
            </div>
          </el-card>

          <el-card class="detail-card">
            <template #header>
              <div class="card-header">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
                  <polyline points="14 2 14 8 20 8"/>
                </svg>
                <span>存储配置</span>
              </div>
            </template>
            <div class="info-list">
              <div class="info-item">
                <span class="info-label">数据目录</span>
                <span class="info-value mono" v-if="task.data_path">{{ task.data_path }}</span>
                <span class="info-value text-secondary" v-else>-</span>
              </div>
              <div class="info-item">
                <span class="info-label">输出目录</span>
                <span class="info-value mono" v-if="task.output_path">{{ task.output_path }}</span>
                <span class="info-value text-secondary" v-else>-</span>
              </div>
            </div>
          </el-card>

          <!-- 错误信息 -->
          <el-card class="detail-card full-width error-card" v-if="task.error_msg">
            <template #header>
              <div class="card-header error">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="12" y1="8" x2="12" y2="12"/>
                  <line x1="12" y1="16" x2="12.01" y2="16"/>
                </svg>
                <span>错误信息</span>
              </div>
            </template>
            <div class="error-message">
              {{ task.error_msg }}
            </div>
          </el-card>
        </div>

        <!-- 日志查看器 -->
        <el-card class="log-card">
          <template #header>
            <div class="card-header">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
                <polyline points="14 2 14 8 20 8"/>
                <line x1="16" y1="13" x2="8" y2="13"/>
                <line x1="16" y1="17" x2="8" y2="17"/>
              </svg>
              <span>执行日志</span>
              <div class="log-actions">
                <el-button size="small" text @click="clearLogs">
                  <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                  </svg>
                  清空
                </el-button>
                <el-button size="small" text @click="downloadLogs">
                  <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                    <polyline points="7 10 12 15 17 10"/>
                    <line x1="12" y1="15" x2="12" y2="3"/>
                  </svg>
                  下载
                </el-button>
              </div>
            </div>
          </template>
          <div class="log-viewer" ref="logViewer">
            <div v-for="(log, index) in logs" :key="index" class="log-entry" :class="log.level">
              <span class="log-time">{{ formatLogTime(log.timestamp) }}</span>
              <span class="log-level" :class="log.level">{{ getLogLevel(log.level) }}</span>
              <span class="log-message">{{ log.message }}</span>
            </div>
            <div v-if="logs.length === 0" class="no-logs">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
                <polyline points="14 2 14 8 20 8"/>
              </svg>
              <span>等待日志输出...</span>
            </div>
          </div>
        </el-card>
      </template>

      <div v-else-if="!loading" class="empty-state">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
          <polyline points="14 2 14 8 20 8"/>
          <line x1="9" y1="15" x2="15" y2="15"/>
        </svg>
        <h3>任务不存在</h3>
        <p>该任务可能已被删除或不存在</p>
        <el-button type="primary" @click="$router.push('/tasks')">返回任务列表</el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTaskStore } from '../stores/task'
import { ElMessage } from 'element-plus'

const props = defineProps({
  id: { type: String, required: true }
})

const route = useRoute()
const router = useRouter()
const taskStore = useTaskStore()
const task = ref(null)
const loading = ref(false)
const logs = ref([])
const logViewer = ref(null)
let eventSource = null
let clientId = ''

onMounted(async () => {
  const taskId = props.id || route.params.id
  if (!taskId) {
    router.push('/tasks')
    return
  }

  loading.value = true
  try {
    task.value = await taskStore.getTask(taskId)
  } catch (err) {
    console.error('Failed to fetch task:', err)
    ElMessage.error('获取任务详情失败')
  } finally {
    loading.value = false
  }

  connectLogStream(taskId)
})

onUnmounted(() => {
  if (eventSource) {
    eventSource.close()
  }
  if (clientId && props.id) {
    fetch(`/api/v1/tasks/${props.id}/logs?client_id=${clientId}`, { method: 'DELETE' })
  }
})

const connectLogStream = (taskId) => {
  clientId = 'client-' + Date.now()
  const url = `/api/v1/tasks/${taskId}/logs?client_id=${clientId}`

  eventSource = new EventSource(url)

  eventSource.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      logs.value.push(data)
      nextTick(() => {
        if (logViewer.value) {
          logViewer.value.scrollTop = logViewer.value.scrollHeight
        }
      })
    } catch (err) {
      console.error('Failed to parse log:', err)
    }
  }

  eventSource.onerror = () => {
    console.error('SSE connection error, reconnecting...')
    setTimeout(() => connectLogStream(taskId), 3000)
  }
}

const getStatusLabel = (status) => {
  const labels = {
    pending: '等待中',
    running: '执行中',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消'
  }
  return labels[status] || status
}

const getLogLevel = (level) => {
  const levels = {
    info: 'INFO',
    warn: 'WARN',
    error: 'ERROR',
    debug: 'DEBUG'
  }
  return levels[level] || 'INFO'
}

const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

const formatLogTime = (time) => {
  if (!time) return ''
  return new Date(time).toLocaleTimeString('zh-CN')
}

const copyCommand = async () => {
  if (task.value?.command) {
    await navigator.clipboard.writeText(task.value.command)
    ElMessage.success('命令已复制到剪贴板')
  }
}

const clearLogs = () => {
  logs.value = []
}

const downloadLogs = () => {
  if (logs.value.length === 0) {
    ElMessage.warning('暂无日志可下载')
    return
  }

  const logContent = logs.value.map(log =>
    `[${formatLogTime(log.timestamp)}] [${getLogLevel(log.level)}] ${log.message}`
  ).join('\n')

  const blob = new Blob([logContent], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `task-${task.value?.id || 'logs'}.log`
  a.click()
  URL.revokeObjectURL(url)
}

const handleCancel = async () => {
  try {
    await taskStore.cancelTask(props.id || route.params.id)
    ElMessage.success('任务已取消')
    task.value = await taskStore.getTask(props.id || route.params.id)
  } catch (err) {
    ElMessage.error('取消失败')
  }
}
</script>

<style scoped>
.task-detail-page {
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.back-icon {
  width: 16px;
  height: 16px;
  margin-right: 8px;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.task-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 500;
  width: fit-content;
}

.task-status-badge.pending {
  background: #fef3c7;
  color: #d97706;
}

.task-status-badge.running {
  background: #dbeafe;
  color: #2563eb;
}

.task-status-badge.completed {
  background: #d1fae5;
  color: #059669;
}

.task-status-badge.failed {
  background: #fee2e2;
  color: #dc2626;
}

.task-status-badge.cancelled {
  background: #f3f4f6;
  color: #6b7280;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}

.task-status-badge.running .status-dot {
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.btn-icon {
  width: 16px;
  height: 16px;
  margin-right: 6px;
}

.detail-content {
  min-height: 400px;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
  margin-bottom: 24px;
}

.detail-card {
  border-radius: var(--radius-lg);
}

.detail-card.full-width {
  grid-column: 1 / -1;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 600;
  color: var(--text-primary);
}

.card-header svg {
  width: 20px;
  height: 20px;
  color: var(--primary-color);
}

.card-header.error svg {
  color: var(--danger-color);
}

.info-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-label {
  font-size: 12px;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.info-value {
  font-size: 14px;
  color: var(--text-primary);
  font-weight: 500;
}

.info-value.mono {
  font-family: 'Consolas', monospace;
}

.info-value.text-secondary {
  color: var(--text-secondary);
}

.image-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  width: fit-content;
}

.image-name {
  font-weight: 600;
  color: var(--text-primary);
}

.image-tag {
  font-size: 12px;
  color: var(--primary-color);
  font-family: monospace;
}

.command-block {
  position: relative;
  background: #1e293b;
  border-radius: var(--radius-md);
  padding: 16px;
}

.command-block code {
  font-family: 'Consolas', monospace;
  font-size: 13px;
  color: #e2e8f0;
  word-break: break-all;
  display: block;
  padding-right: 40px;
}

.copy-btn {
  position: absolute;
  top: 8px;
  right: 8px;
  color: #94a3b8;
}

.copy-btn:hover {
  color: white;
}

.copy-btn svg {
  width: 16px;
  height: 16px;
}

.error-card {
  border: 1px solid #fecaca;
  background: #fef2f2;
}

.error-card :deep(.el-card__header) {
  background: #fee2e2;
  border-bottom: 1px solid #fecaca;
}

.error-message {
  color: #dc2626;
  font-size: 14px;
  line-height: 1.6;
}

/* 日志查看器 */
.log-card {
  border-radius: var(--radius-lg);
}

.log-card :deep(.el-card__header) {
  padding: 16px 20px;
}

.log-actions {
  margin-left: auto;
  display: flex;
  gap: 8px;
}

.log-viewer {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 16px;
  border-radius: var(--radius-md);
  max-height: 450px;
  overflow-y: auto;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.log-entry {
  display: flex;
  gap: 12px;
  padding: 4px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.log-entry:last-child {
  border-bottom: none;
}

.log-time {
  color: #6a9955;
  flex-shrink: 0;
}

.log-level {
  flex-shrink: 0;
  padding: 0 6px;
  border-radius: 3px;
  font-size: 11px;
  font-weight: 600;
}

.log-level.info {
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
}

.log-level.warn {
  background: rgba(250, 204, 21, 0.2);
  color: #fbbf24;
}

.log-level.error {
  background: rgba(248, 113, 113, 0.2);
  color: #f87171;
}

.log-level.debug {
  background: rgba(156, 163, 175, 0.2);
  color: #9ca3af;
}

.log-message {
  flex: 1;
  word-break: break-word;
}

.no-logs {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: #6b7280;
  gap: 12px;
}

.no-logs svg {
  width: 48px;
  height: 48px;
  opacity: 0.5;
}

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 80px 20px;
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-color);
}

.empty-state svg {
  width: 64px;
  height: 64px;
  color: var(--text-secondary);
  margin-bottom: 16px;
  opacity: 0.5;
}

.empty-state h3 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.empty-state p {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 24px;
}

/* 响应式 */
@media (max-width: 1024px) {
  .detail-grid {
    grid-template-columns: 1fr;
  }

  .detail-card.full-width {
    grid-column: 1;
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .header-actions {
    width: 100%;
  }

  .header-actions .el-button {
    width: 100%;
  }
}
</style>
