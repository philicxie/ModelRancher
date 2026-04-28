<template>
  <div class="task-detail">
    <el-button @click="$router.push('/')" style="margin-bottom: 16px">
      返回列表
    </el-button>

    <el-card v-loading="loading">
      <template #header>
        <span>任务详情</span>
      </template>

      <div v-if="task">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="任务名称">{{ task.name }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusType(task.status)" effect="dark">
              {{ getStatusLabel(task.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="镜像" :span="2">{{ task.image }}</el-descriptions-item>
          <el-descriptions-item label="命令" :span="2">
            <code>{{ task.command }}</code>
          </el-descriptions-item>
          <el-descriptions-item label="数据目录">{{ task.data_path || '-' }}</el-descriptions-item>
          <el-descriptions-item label="输出目录">{{ task.output_path || '-' }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTime(task.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="容器ID">{{ task.container_id || '-' }}</el-descriptions-item>
        </el-descriptions>

        <div v-if="task.error_msg" class="error-msg">
          <el-alert type="error" :title="task.error_msg" show-icon :closable="false" />
        </div>

        <div class="log-section">
          <h3>执行日志</h3>
          <div class="log-viewer" ref="logViewer">
            <div v-for="(log, index) in logs" :key="index" :class="['log-entry', log.level]">
              <span class="log-time">{{ formatLogTime(log.timestamp) }}</span>
              <span class="log-message">{{ log.message }}</span>
            </div>
            <div v-if="logs.length === 0" class="no-logs">
              等待日志输出...
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="!loading">
        <el-empty description="任务不存在" />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { useTaskStore } from '../stores/task'
import axios from 'axios'

const props = defineProps({
  id: { type: String, required: true }
})

const route = useRoute()
const taskStore = useTaskStore()
const task = ref(null)
const loading = ref(false)
const logs = ref([])
const logViewer = ref(null)
let eventSource = null
let clientId = ''

onMounted(async () => {
  const taskId = props.id || route.params.id
  if (!taskId) return

  loading.value = true
  try {
    task.value = await taskStore.getTask(taskId)
  } catch (err) {
    console.error('Failed to fetch task:', err)
  } finally {
    loading.value = false
  }

  // 连接日志流
  connectLogStream(taskId)
})

onUnmounted(() => {
  if (eventSource) {
    eventSource.close()
  }
  if (clientId) {
    axios.delete(`/api/v1/tasks/${props.id}/logs?client_id=${clientId}`)
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

const getStatusType = (status) => {
  const types = {
    pending: 'info',
    running: 'primary',
    completed: 'success',
    failed: 'danger',
    cancelled: 'warning'
  }
  return types[status] || 'info'
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

const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

const formatLogTime = (time) => {
  if (!time) return ''
  return new Date(time).toLocaleTimeString('zh-CN')
}
</script>

<style scoped>
.task-detail {
  max-width: 1000px;
  margin: 0 auto;
}
.error-msg {
  margin: 16px 0;
}
.log-section {
  margin-top: 24px;
}
.log-section h3 {
  margin-bottom: 12px;
}
.log-viewer {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 16px;
  border-radius: 4px;
  max-height: 400px;
  overflow-y: auto;
  font-family: 'Consolas', monospace;
  font-size: 13px;
}
.log-entry {
  padding: 2px 0;
}
.log-entry .log-time {
  color: #858585;
  margin-right: 8px;
}
.log-entry.error .log-message {
  color: #f48771;
}
.log-entry.warn .log-message {
  color: #dcdcaa;
}
.no-logs {
  color: #858585;
  text-align: center;
  padding: 20px;
}
</style>