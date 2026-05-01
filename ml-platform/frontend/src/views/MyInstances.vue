<template>
  <div class="my-instances-page">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">我的实例</h1>
        <p class="page-subtitle">管理您的 GPU 实例</p>
      </div>
      <div class="header-actions">
        <el-button @click="refresh">
          <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="23 4 23 10 17 10"/>
            <polyline points="1 20 1 14 7 14"/>
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
          </svg>
          刷新
        </el-button>
        <el-button type="primary" @click="$router.push('/browse')">
          <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="12 2 2 7 12 12 22 7 12 2"/>
            <polyline points="2 17 12 22 22 17"/>
            <polyline points="2 12 12 17 22 12"/>
          </svg>
          浏览实例
        </el-button>
      </div>
    </div>

    <!-- 实例列表 -->
    <div v-loading="loading" class="instance-cards">
      <div v-if="instances.length === 0 && !loading" class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="4" y="4" width="16" height="16" rx="2" ry="2"/>
          <rect x="9" y="9" width="6" height="6"/>
        </svg>
        <h3>暂无实例</h3>
        <p>您还没有租赁任何 GPU 实例</p>
        <el-button type="primary" @click="$router.push('/browse')">去浏览实例</el-button>
      </div>

      <div v-for="instance in instances" :key="instance.id" class="instance-card" :class="getStatusClass(instance.status)">
        <div class="card-header">
          <div class="instance-title">
            <span class="instance-name">{{ instance.name || instance.id }}</span>
            <span class="instance-id">{{ instance.id }}</span>
          </div>
          <el-tag :type="getStatusTagType(instance.status)" size="small">
            {{ getStatusText(instance.status) }}
          </el-tag>
        </div>

        <div class="card-body">
          <div class="info-row">
            <span class="info-label">GPU</span>
            <span class="info-value">{{ instance.gpu_name }} x{{ instance.num_gpus || 1 }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">提供商</span>
            <span class="info-value">{{ instance.provider }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">价格</span>
            <span class="info-value price">${{ formatPrice(instance.price_per_hour) }}/小时</span>
          </div>
          <div v-if="instance.location" class="info-row">
            <span class="info-label">位置</span>
            <span class="info-value">{{ instance.location }}</span>
          </div>
        </div>

        <div class="card-actions">
          <el-button size="small" text @click="$router.push(`/instance/${instance.id}`)">
            详情
          </el-button>
          <template v-if="instance.status === 'running'">
            <el-button size="small" @click="stopInstance(instance)">停止</el-button>
          </template>
          <template v-else-if="instance.status === 'stopped'">
            <el-button type="primary" size="small" @click="startInstance(instance)">启动</el-button>
          </template>
          <template v-else-if="instance.status === 'destroying'">
            <el-button size="small" disabled>
              <el-icon class="is-loading"><Loading /></el-icon>
              销毁中...
            </el-button>
          </template>
          <el-button v-if="instance.status !== 'destroying' && instance.status !== 'destroyed'" type="danger" size="small" @click="destroyInstance(instance)">销毁</el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'

const instances = ref([])
const loading = ref(false)
let pollTimer = null

const fetchInstances = async () => {
  loading.value = true
  try {
    const userId = 'default'
    const res = await fetch(`/api/v1/offers?user_id=${userId}`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    instances.value = (data.instances || []).filter(i => i.status !== 'destroyed')
  } catch (err) {
    console.error('fetchInstances failed:', err)
    ElMessage.error('获取实例列表失败')
    instances.value = []
  } finally {
    loading.value = false
  }
}

const refresh = () => {
  fetchInstances()
  ElMessage.success('已刷新')
}

const formatPrice = (price) => {
  const n = parseFloat(price)
  return isNaN(n) ? '0.00' : n.toFixed(2)
}

const getStatusClass = (status) => ({
  'status-running': status === 'running',
  'status-stopped': status === 'stopped',
  'status-starting': status === 'starting' || status === 'pending' || status === 'creating',
  'status-destroying': status === 'destroying',
  'status-error': status === 'failed' || status === 'destroyed'
})

const getStatusTagType = (status) => {
  const map = { running: 'success', stopped: 'info', starting: 'warning', pending: 'warning', creating: 'warning', destroying: 'danger', destroyed: 'info', failed: 'danger' }
  return map[status] || ''
}

const getStatusText = (status) => {
  const map = { running: '运行中', stopped: '已停止', starting: '启动中', pending: '创建中', creating: '创建中', destroying: '销毁中', destroyed: '已销毁', failed: '失败' }
  return map[status] || status
}

const stopInstance = async (instance) => {
  try {
    const res = await fetch(`/api/v1/offers/${instance.id}/stop`, { method: 'POST' })
    if (!res.ok) throw new Error()
    ElMessage.success('实例已停止')
    fetchInstances()
  } catch (err) {
    ElMessage.error('停止失败')
  }
}

const startInstance = async (instance) => {
  try {
    const res = await fetch(`/api/v1/offers/${instance.id}/start`, { method: 'POST' })
    if (!res.ok) throw new Error()
    ElMessage.success('实例启动中')
    fetchInstances()
  } catch (err) {
    ElMessage.error('启动失败')
  }
}

const destroyInstance = async (instance) => {
  try {
    const res = await fetch(`/api/v1/offers/${instance.id}`, { method: 'DELETE' })
    if (!res.ok) throw new Error()
    ElMessage.success('实例已销毁')
    instances.value = instances.value.filter(i => i.id !== instance.id)
  } catch (err) {
    ElMessage.error('销毁失败')
  }
}

onMounted(() => {
  fetchInstances()
  pollTimer = setInterval(fetchInstances, 30000)
})

onUnmounted(() => {
  clearInterval(pollTimer)
})
</script>

<style scoped>
.my-instances-page {
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.03em;
  margin-bottom: 6px;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
}

.btn-icon {
  width: 16px;
  height: 16px;
  margin-right: 6px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.instance-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 16px;
}

.instance-card {
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  padding: 16px;
  border: 2px solid var(--border-color);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.instance-card:hover {
  box-shadow: 0 8px 24px -8px rgba(15, 23, 42, 0.1);
}

.instance-card.status-running { border-color: #10b981; box-shadow: 0 0 0 1px rgba(16, 185, 129, 0.08); }
.instance-card.status-starting { border-color: #f59e0b; box-shadow: 0 0 0 1px rgba(245, 158, 11, 0.08); }
.instance-card.status-destroying { border-color: #ef4444; opacity: 0.7; }
.instance-card.status-error { border-color: #ef4444; box-shadow: 0 0 0 1px rgba(239, 68, 68, 0.08); }

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.instance-title {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.instance-name {
  font-weight: 700;
  color: var(--text-primary);
  font-size: 15px;
  letter-spacing: -0.01em;
}

.instance-id {
  font-size: 12px;
  color: var(--text-secondary);
  font-family: monospace;
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
}

.info-label {
  color: var(--text-secondary);
}

.info-value {
  color: var(--text-primary);
  font-weight: 500;
}

.info-value.price {
  color: var(--primary-color);
}

.card-actions {
  display: flex;
  gap: 8px;
}

.empty-state {
  grid-column: 1 / -1;
  text-align: center;
  padding: 48px 20px;
  color: var(--text-secondary);
}

.empty-icon {
  width: 64px;
  height: 64px;
  margin-bottom: 16px;
  color: var(--border-color);
}

.empty-state h3 {
  font-size: 16px;
  margin-bottom: 6px;
  color: var(--text-primary);
}

.empty-state p {
  font-size: 13px;
  margin-bottom: 16px;
}
</style>
