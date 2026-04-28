<template>
  <div class="task-list">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>训练任务</span>
          <el-button type="primary" @click="$router.push('/create')">创建任务</el-button>
        </div>
      </template>

      <el-table :data="tasks" v-loading="loading" stripe>
        <el-table-column prop="name" label="任务名称" min-width="150" />
        <el-table-column prop="image" label="镜像" min-width="200" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" effect="dark">
              {{ getStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180">
          <template #default="{ row }">
            <el-button size="small" @click="viewDetail(row.id)">查看</el-button>
            <el-button
              size="small"
              type="danger"
              @click="handleCancel(row.id)"
              :disabled="row.status !== 'pending' && row.status !== 'running'">
              取消
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useTaskStore } from '../stores/task'
import { storeToRefs } from 'pinia'
import { ElMessage } from 'element-plus'

const router = useRouter()
const taskStore = useTaskStore()
const { tasks, loading } = storeToRefs(taskStore)

onMounted(() => {
  taskStore.fetchTasks()
})

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

const viewDetail = (id) => {
  router.push(`/task/${id}`)
}

const handleCancel = async (id) => {
  try {
    await taskStore.cancelTask(id)
    ElMessage.success('任务已取消')
  } catch (err) {
    ElMessage.error('取消失败: ' + err.message)
  }
}
</script>

<style scoped>
.task-list {
  max-width: 1200px;
  margin: 0 auto;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>