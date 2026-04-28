<template>
  <div class="task-list-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">训练任务</h1>
        <p class="page-subtitle">管理所有训练任务</p>
      </div>
      <el-button type="primary" size="large" @click="$router.push('/create')">
        <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"/>
          <line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        新建任务
      </el-button>
    </div>

    <!-- 筛选工具栏 -->
    <div class="filter-bar">
      <div class="filter-left">
        <el-input
          v-model="searchQuery"
          placeholder="搜索任务名称..."
          class="search-input"
          clearable>
          <template #prefix>
            <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/>
              <path d="m21 21-4.35-4.35"/>
            </svg>
          </template>
        </el-input>

        <el-select v-model="statusFilter" placeholder="状态筛选" class="status-select">
          <el-option label="全部状态" value="" />
          <el-option label="等待中" value="pending" />
          <el-option label="运行中" value="running" />
          <el-option label="已完成" value="completed" />
          <el-option label="失败" value="failed" />
        </el-select>
      </div>

      <div class="filter-right">
        <span class="result-count">共 {{ filteredTasks.length }} 个任务</span>
      </div>
    </div>

    <!-- 任务列表 -->
    <div class="tasks-container">
      <div v-if="loading" class="loading-state">
        <el-skeleton :rows="5" animated />
      </div>

      <div v-else-if="filteredTasks.length === 0" class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
          <polyline points="14 2 14 8 20 8"/>
          <line x1="9" y1="15" x2="15" y2="15"/>
        </svg>
        <h3>暂无任务</h3>
        <p>创建您的第一个训练任务开始使用</p>
        <el-button type="primary" @click="$router.push('/create')">创建任务</el-button>
      </div>

      <div v-else class="tasks-grid">
        <div
          v-for="task in filteredTasks"
          :key="task.id"
          class="task-card"
          :class="task.status">
          <div class="task-card-header">
            <div class="task-status-indicator">
              <span class="status-dot" :class="task.status"></span>
              <span class="status-text">{{ getStatusLabel(task.status) }}</span>
            </div>
            <el-dropdown trigger="click">
              <el-button size="small" text>
                <svg class="more-icon" viewBox="0 0 24 24" fill="currentColor">
                  <circle cx="12" cy="5" r="2"/>
                  <circle cx="12" cy="12" r="2"/>
                  <circle cx="12" cy="19" r="2"/>
                </svg>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="viewDetail(task.id)">查看详情</el-dropdown-item>
                  <el-dropdown-item
                    v-if="task.status === 'pending' || task.status === 'running'"
                    @click="handleCancel(task.id)"
                    divided>取消任务</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>

          <div class="task-card-body">
            <h3 class="task-name">{{ task.name }}</h3>
            <p class="task-description" v-if="task.description">{{ task.description }}</p>

            <div class="task-meta">
              <div class="meta-item">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                  <circle cx="8.5" cy="8.5" r="1.5"/>
                  <polyline points="21 15 16 10 5 21"/>
                </svg>
                <span class="image-tag">{{ task.image.split(':')[0] }}</span>
              </div>
              <div class="meta-item">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"/>
                  <polyline points="12 6 12 12 16 14"/>
                </svg>
                <span>{{ formatTime(task.created_at) }}</span>
              </div>
            </div>

            <div class="task-progress" v-if="task.status === 'running'">
              <div class="progress-bar">
                <div class="progress-fill" style="width: 45%"></div>
              </div>
              <span class="progress-text">训练中...</span>
            </div>
          </div>

          <div class="task-card-footer">
            <el-button size="small" @click="viewDetail(task.id)">
              查看详情
            </el-button>
            <el-button
              v-if="task.status === 'pending' || task.status === 'running'"
              size="small"
              type="danger"
              plain
              @click="handleCancel(task.id)">
              取消
            </el-button>
          </div>
        </div>
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

const searchQuery = ref('')
const statusFilter = ref('')

const filteredTasks = computed(() => {
  let result = tasks.value

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(t => t.name.toLowerCase().includes(query))
  }

  if (statusFilter.value) {
    result = result.filter(t => t.status === statusFilter.value)
  }

  return result
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
.task-list-page {
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
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
}

.btn-icon {
  width: 18px;
  height: 18px;
  margin-right: 8px;
}

/* 筛选工具栏 */
.filter-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  padding: 16px 20px;
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-color);
}

.filter-left {
  display: flex;
  gap: 12px;
}

.search-input {
  width: 280px;
}

.search-icon {
  width: 16px;
  height: 16px;
  color: var(--text-secondary);
}

.status-select {
  width: 140px;
}

.result-count {
  font-size: 14px;
  color: var(--text-secondary);
}

/* 任务列表 */
.tasks-container {
  min-height: 400px;
}

.loading-state {
  padding: 20px;
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-color);
}

.empty-icon {
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

/* 任务卡片网格 */
.tasks-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 20px;
}

.task-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  overflow: hidden;
  transition: all 0.2s;
}

.task-card:hover {
  box-shadow: var(--shadow-md);
  border-color: var(--primary-color);
}

.task-card.running {
  border-left: 4px solid #3b82f6;
}

.task-card.completed {
  border-left: 4px solid #10b981;
}

.task-card.failed {
  border-left: 4px solid #ef4444;
}

.task-card.pending {
  border-left: 4px solid #f59e0b;
}

.task-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-primary);
}

.task-status-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-dot.pending {
  background: #f59e0b;
}

.status-dot.running {
  background: #3b82f6;
  animation: pulse 1.5s infinite;
}

.status-dot.completed {
  background: #10b981;
}

.status-dot.failed {
  background: #ef4444;
}

.status-dot.cancelled {
  background: #6b7280;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.status-text {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
}

.more-icon {
  width: 16px;
  height: 16px;
}

.task-card-body {
  padding: 20px;
}

.task-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.task-description {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 16px;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.task-meta {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-secondary);
}

.meta-item svg {
  width: 14px;
  height: 14px;
}

.image-tag {
  padding: 2px 8px;
  background: #f1f5f9;
  border-radius: 4px;
}

.task-progress {
  margin-top: 12px;
}

.progress-bar {
  height: 4px;
  background: #e2e8f0;
  border-radius: 2px;
  overflow: hidden;
  margin-bottom: 8px;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #4f46e5, #06b6d4);
  border-radius: 2px;
  transition: width 0.3s;
}

.progress-text {
  font-size: 12px;
  color: #3b82f6;
  font-weight: 500;
}

.task-card-footer {
  display: flex;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
  background: var(--bg-primary);
}

/* 响应式 */
@media (max-width: 768px) {
  .filter-bar {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }

  .filter-left {
    flex-direction: column;
  }

  .search-input {
    width: 100%;
  }

  .tasks-grid {
    grid-template-columns: 1fr;
  }
}
</style>