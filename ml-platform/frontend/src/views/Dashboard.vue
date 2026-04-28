<template>
  <div class="dashboard">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">控制台</h1>
        <p class="page-subtitle">欢迎使用 ML 训练平台</p>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon bg-blue">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
            <polyline points="14 2 14 8 20 8"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.total }}</div>
          <div class="stat-label">总任务数</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon bg-green">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
            <polyline points="22 4 12 14.01 9 11.01"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.completed }}</div>
          <div class="stat-label">已完成</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon bg-yellow">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <polyline points="12 6 12 12 16 14"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.running }}</div>
          <div class="stat-label">运行中</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon bg-red">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="15" y1="9" x2="9" y2="15"/>
            <line x1="9" y1="9" x2="15" y2="15"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.failed }}</div>
          <div class="stat-label">失败</div>
        </div>
      </div>
    </div>

    <!-- 快捷操作 -->
    <div class="quick-actions">
      <h2 class="section-title">快捷操作</h2>
      <div class="actions-grid">
        <div class="action-card" @click="$router.push('/create')">
          <div class="action-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="12" y1="5" x2="12" y2="19"/>
              <line x1="5" y1="12" x2="19" y2="12"/>
            </svg>
          </div>
          <div class="action-text">
            <h3>创建任务</h3>
            <p>提交新的训练任务</p>
          </div>
        </div>

        <div class="action-card" @click="$router.push('/tasks')">
          <div class="action-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
              <polyline points="14 2 14 8 20 8"/>
            </svg>
          </div>
          <div class="action-text">
            <h3>查看任务</h3>
            <p>管理所有训练任务</p>
          </div>
        </div>

        <div class="action-card" @click="$router.push('/storage')">
          <div class="action-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
            </svg>
          </div>
          <div class="action-text">
            <h3>数据存储</h3>
            <p>管理训练数据文件</p>
          </div>
        </div>

        <div class="action-card" @click="$router.push('/images')">
          <div class="action-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
              <circle cx="8.5" cy="8.5" r="1.5"/>
              <polyline points="21 15 16 10 5 21"/>
            </svg>
          </div>
          <div class="action-text">
            <h3>镜像管理</h3>
            <p>选择训练环境</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 最近任务 -->
    <div class="recent-tasks">
      <div class="section-header">
        <h2 class="section-title">最近任务</h2>
        <router-link to="/tasks" class="view-all">查看全部</router-link>
      </div>
      <div class="tasks-table-wrapper">
        <el-table :data="recentTasks" stripe class="tasks-table" v-loading="loading">
          <el-table-column prop="name" label="任务名称" min-width="180">
            <template #default="{ row }">
              <div class="task-name-cell">
                <span class="task-name">{{ row.name }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="image" label="镜像" min-width="200" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="image-tag">{{ row.image.split(':')[0] }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="120">
            <template #default="{ row }">
              <span class="status-badge" :class="row.status">
                {{ getStatusLabel(row.status) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="180">
            <template #default="{ row }">
              <span class="time-text">{{ formatTime(row.created_at) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="{ row }">
              <div class="action-buttons">
                <el-button size="small" @click="viewDetail(row.id)">查看</el-button>
                <el-button
                  size="small"
                  type="danger"
                  plain
                  @click="handleCancel(row.id)"
                  v-if="row.status === 'pending' || row.status === 'running'">
                  取消
                </el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useTaskStore } from '../stores/task'
import { storeToRefs } from 'pinia'
import { ElMessage } from 'element-plus'

const router = useRouter()
const taskStore = useTaskStore()
const { tasks, loading } = storeToRefs(taskStore)

const recentTasks = computed(() => {
  return tasks.value.slice(0, 5)
})

const stats = computed(() => {
  return {
    total: tasks.value.length,
    completed: tasks.value.filter(t => t.status === 'completed').length,
    running: tasks.value.filter(t => t.status === 'running').length,
    failed: tasks.value.filter(t => t.status === 'failed').length
  }
})

onMounted(() => {
  taskStore.fetchTasks()
})

const getStatusLabel = (status) => {
  const labels = {
    pending: '等待中',
    running: '运行中',
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

const viewDetail = (id) => {
  router.push(`/task/${id}`)
}

const handleCancel = async (id) => {
  try {
    await taskStore.cancelTask(id)
    ElMessage.success('任务已取消')
  } catch (err) {
    ElMessage.error('取消失败')
  }
}
</script>

<style scoped>
.dashboard {
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
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
}

.header-actions .btn-icon {
  width: 18px;
  height: 18px;
  margin-right: 8px;
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
  transition: transform 0.2s, box-shadow 0.2s;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
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
  background: #eff6ff;
  color: #3b82f6;
}

.bg-green {
  background: #ecfdf5;
  color: #10b981;
}

.bg-yellow {
  background: #fffbeb;
  color: #f59e0b;
}

.bg-red {
  background: #fef2f2;
  color: #ef4444;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.2;
}

.stat-label {
  font-size: 14px;
  color: var(--text-secondary);
  margin-top: 4px;
}

/* 快捷操作 */
.quick-actions {
  margin-bottom: 32px;
}

.section-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 16px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.view-all {
  font-size: 14px;
  color: var(--primary-color);
  text-decoration: none;
}

.view-all:hover {
  text-decoration: underline;
}

.actions-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.action-card {
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  border: 1px solid var(--border-color);
  cursor: pointer;
  transition: all 0.2s;
}

.action-card:hover {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.1);
}

.action-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  background: var(--bg-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary-color);
}

.action-icon svg {
  width: 24px;
  height: 24px;
}

.action-text h3 {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.action-text p {
  font-size: 13px;
  color: var(--text-secondary);
}

/* 最近任务 */
.recent-tasks {
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  padding: 24px;
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--border-color);
}

.tasks-table-wrapper {
  margin-top: 16px;
}

.tasks-table {
  border-radius: var(--radius-md);
}

.task-name-cell {
  display: flex;
  align-items: center;
}

.task-name {
  font-weight: 500;
  color: var(--text-primary);
}

.image-tag {
  font-size: 12px;
  padding: 2px 8px;
  background: #f1f5f9;
  border-radius: 4px;
  color: var(--text-secondary);
}

.status-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 500;
}

.status-badge.pending {
  background: #fef3c7;
  color: #d97706;
}

.status-badge.running {
  background: #dbeafe;
  color: #2563eb;
}

.status-badge.completed {
  background: #d1fae5;
  color: #059669;
}

.status-badge.failed {
  background: #fee2e2;
  color: #dc2626;
}

.status-badge.cancelled {
  background: #f3f4f6;
  color: #6b7280;
}

.time-text {
  font-size: 13px;
  color: var(--text-secondary);
}

.action-buttons {
  display: flex;
  gap: 8px;
}

/* 响应式 */
@media (max-width: 1200px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .actions-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .stats-grid {
    grid-template-columns: 1fr;
  }

  .actions-grid {
    grid-template-columns: 1fr;
  }
}
</style>