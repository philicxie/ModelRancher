<template>
  <div class="work-orders">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">历史工单</h1>
        <p class="page-subtitle">管理所有实例租赁记录与开销统计</p>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon bg-blue">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
            <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.total }}</div>
          <div class="stat-label">总工单数</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon bg-purple">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="1" x2="12" y2="23"/>
            <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">${{ stats.totalCost.toFixed(2) }}</div>
          <div class="stat-label">累计开销</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon bg-green">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.running }}</div>
          <div class="stat-label">运行中</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon bg-gray">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.ended }}</div>
          <div class="stat-label">已结束</div>
        </div>
      </div>
    </div>

    <!-- 工单表格 -->
    <div class="orders-section">
      <div class="section-header">
        <h2 class="section-title">工单列表</h2>
        <el-radio-group v-model="filterStatus" size="small">
          <el-radio-button label="">全部</el-radio-button>
          <el-radio-button label="running">运行中</el-radio-button>
          <el-radio-button label="ended">已结束</el-radio-button>
        </el-radio-group>
      </div>

      <el-table
        v-loading="loading"
        :data="filteredOrders"
        style="width: 100%"
        class="orders-table"
        row-key="id"
      >
        <el-table-column label="名称" min-width="140">
          <template #default="{ row }">
            <div class="order-name-cell">
              <span class="order-name">{{ row.name || '未命名实例' }}</span>
              <span class="order-id">{{ row.id.slice(0, 8) }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="提供商" width="90">
          <template #default="{ row }">
            <span class="provider-tag" :class="row.provider">{{ row.provider }}</span>
          </template>
        </el-table-column>

        <el-table-column label="GPU配置" width="160">
          <template #default="{ row }">
            <div class="gpu-config">
              <span class="gpu-name">{{ row.gpu_name || '-' }}</span>
              <span class="gpu-count">×{{ row.num_gpus }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="创建参数" min-width="180">
          <template #default="{ row }">
            <div class="params-cell">
              <div class="param-item">
                <span class="param-label">镜像</span>
                <span class="param-value" :title="row.image">{{ row.image || '-' }}</span>
              </div>
              <div class="param-item">
                <span class="param-label">磁盘</span>
                <span class="param-value">{{ row.disk_size || 0 }} GB</span>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="起止时间" min-width="200">
          <template #default="{ row }">
            <div class="time-cell">
              <div class="time-row">
                <span class="time-dot start"/>
                <span class="time-text">{{ formatTime(row.started_at) || '未启动' }}</span>
              </div>
              <div class="time-row">
                <span class="time-dot end"/>
                <span class="time-text">{{ formatEndTime(row) }}</span>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <span class="status-badge" :class="row.status">{{ statusLabel(row.status) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="运行时长" width="110">
          <template #default="{ row }">
            <span class="duration-text">{{ formatDuration(row.duration_hours) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="总开销" width="110" align="right">
          <template #default="{ row }">
            <span class="cost-value">${{ row.total_cost.toFixed(2) }}</span>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="!loading && filteredOrders.length === 0" class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
          <line x1="9" y1="9" x2="15" y2="15"/>
          <line x1="15" y1="9" x2="9" y2="15"/>
        </svg>
        <h3>暂无工单</h3>
        <p>您还没有创建过实例租赁工单</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'

const loading = ref(true)
const orders = ref([])
const filterStatus = ref('')

const stats = computed(() => {
  const total = orders.value.length
  const totalCost = orders.value.reduce((sum, o) => sum + (o.total_cost || 0), 0)
  const running = orders.value.filter(o => ['running', 'starting', 'creating'].includes(o.status)).length
  const ended = orders.value.filter(o => ['destroyed', 'stopped', 'failed'].includes(o.status)).length
  return { total, totalCost, running, ended }
})

const filteredOrders = computed(() => {
  if (!filterStatus.value) return orders.value
  if (filterStatus.value === 'running') {
    return orders.value.filter(o => ['running', 'starting', 'creating'].includes(o.status))
  }
  if (filterStatus.value === 'ended') {
    return orders.value.filter(o => ['destroyed', 'stopped', 'failed'].includes(o.status))
  }
  return orders.value
})

const fetchOrders = async () => {
  loading.value = true
  try {
    const userId = 'default'
    const res = await fetch(`/api/v1/work-orders?user_id=${userId}`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    orders.value = data.orders || []
  } catch (err) {
    console.error('fetchOrders failed:', err)
  } finally {
    loading.value = false
  }
}

const formatTime = (t) => {
  if (!t) return null
  const d = new Date(t)
  return `${d.getFullYear()}-${(d.getMonth()+1).toString().padStart(2,'0')}-${d.getDate().toString().padStart(2,'0')} ${d.getHours().toString().padStart(2,'0')}:${d.getMinutes().toString().padStart(2,'0')}`
}

const formatEndTime = (row) => {
  if (['destroyed', 'stopped', 'failed'].includes(row.status)) {
    const t = row.destroyed_at || row.stopped_at
    return formatTime(t) || '已结束'
  }
  return '运行中...'
}

const formatDuration = (hours) => {
  if (!hours || hours <= 0) return '-'
  const h = Math.floor(hours)
  const m = Math.floor((hours - h) * 60)
  if (h >= 24) {
    const d = Math.floor(h / 24)
    const rh = h % 24
    return `${d}天${rh}小时`
  }
  if (h === 0) return `${m}分钟`
  return `${h}小时${m}分钟`
}

const statusLabel = (status) => {
  const map = {
    running: '运行中',
    creating: '创建中',
    starting: '启动中',
    stopped: '已停止',
    destroying: '销毁中',
    destroyed: '已结束',
    failed: '失败',
  }
  return map[status] || status
}

onMounted(() => {
  fetchOrders()
})
</script>

<style scoped>
.work-orders {
  max-width: 1400px;
  margin: 0 auto;
}

/* 页面标题 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 32px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.03em;
  margin-bottom: 8px;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
}

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 32px;
}

.stat-card {
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  padding: 24px;
  display: flex;
  align-items: center;
  gap: 20px;
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--border-color);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.stat-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 12px 24px -8px rgba(15, 23, 42, 0.12);
  border-color: rgba(56, 189, 248, 0.25);
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-icon svg {
  width: 28px;
  height: 28px;
}

.bg-blue {
  background: linear-gradient(135deg, #eff6ff, #dbeafe);
  color: #2563eb;
}

.bg-purple {
  background: linear-gradient(135deg, #f3e8ff, #e9d5ff);
  color: #7c3aed;
}

.bg-green {
  background: linear-gradient(135deg, #ecfdf5, #d1fae5);
  color: #059669;
}

.bg-gray {
  background: linear-gradient(135deg, #f1f5f9, #e2e8f0);
  color: #475569;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.2;
  letter-spacing: -0.02em;
}

.stat-label {
  font-size: 14px;
  color: var(--text-secondary);
  margin-top: 4px;
}

/* 工单列表区域 */
.orders-section {
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  padding: 24px;
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--border-color);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.02em;
  margin: 0;
}

/* 表格自定义 */
.orders-table :deep(.el-table__header th.el-table__cell) {
  background: var(--bg-primary);
  color: var(--text-secondary);
  font-weight: 700;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  padding: 12px 16px;
}

.orders-table :deep(.el-table__row td.el-table__cell) {
  padding: 14px 16px;
}

/* 单元格样式 */
.order-name-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.order-name {
  font-weight: 700;
  color: var(--text-primary);
  font-size: 14px;
  letter-spacing: -0.01em;
}

.order-id {
  font-size: 12px;
  color: var(--text-secondary);
  font-family: 'JetBrains Mono', monospace;
}

.provider-tag {
  display: inline-flex;
  align-items: center;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
}

.provider-tag.vastai { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; }
.provider-tag.ppio { background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); color: white; }
.provider-tag.local { background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%); color: #064e3b; }

.gpu-config {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.gpu-name {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 14px;
}

.gpu-count {
  font-size: 13px;
  color: var(--text-secondary);
}

.params-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.param-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.param-label {
  color: var(--text-secondary);
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  flex-shrink: 0;
}

.param-value {
  color: var(--text-primary);
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 140px;
}

.time-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.time-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.time-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.time-dot.start { background: #10b981; }
.time-dot.end { background: #94a3b8; }

.time-text {
  font-size: 13px;
  color: var(--text-secondary);
  font-family: 'JetBrains Mono', monospace;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.status-badge.running { background: #dbeafe; color: #2563eb; }
.status-badge.creating { background: #fef3c7; color: #d97706; }
.status-badge.starting { background: #fef3c7; color: #d97706; }
.status-badge.stopped { background: #f1f5f9; color: #475569; }
.status-badge.destroying { background: #fee2e2; color: #dc2626; }
.status-badge.destroyed { background: #d1fae5; color: #059669; }
.status-badge.failed { background: #fee2e2; color: #dc2626; }

.duration-text {
  font-size: 13px;
  color: var(--text-primary);
  font-weight: 500;
  font-family: 'JetBrains Mono', monospace;
}

.cost-value {
  font-size: 14px;
  font-weight: 700;
  color: #7c3aed;
  font-family: 'JetBrains Mono', monospace;
  letter-spacing: -0.01em;
}

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 64px 20px;
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
  font-weight: 600;
  margin-bottom: 6px;
  color: var(--text-primary);
}

.empty-state p {
  font-size: 13px;
}

/* 响应式 */
@media (max-width: 1200px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }

  .section-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
}
</style>
