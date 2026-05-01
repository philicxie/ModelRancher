<template>
  <div class="instances-page">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">实例管理</h1>
        <p class="page-subtitle">管理您的 GPU 实例并浏览可用资源</p>
      </div>
      <div class="header-actions">
        <el-button @click="refreshAll">
          <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="23 4 23 10 17 10"/>
            <polyline points="1 20 1 14 7 14"/>
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
          </svg>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 第一部分：我的实例 -->
    <div class="section my-instances-section">
      <div class="section-header">
        <h2 class="section-title">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
            <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
            <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
          </svg>
          我的实例
        </h2>
        <el-tag v-if="myInstances.length" type="info" size="small">{{ myInstances.length }} 个</el-tag>
      </div>

      <div v-loading="myInstancesLoading" class="instance-cards">
        <div v-if="myInstances.length === 0 && !myInstancesLoading" class="empty-state">
          <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <rect x="4" y="4" width="16" height="16" rx="2" ry="2"/>
            <rect x="9" y="9" width="6" height="6"/>
          </svg>
          <h3>暂无实例</h3>
          <p>您还没有租赁任何 GPU 实例，可在下方浏览可用资源</p>
        </div>

        <div v-for="instance in myInstances" :key="instance.id" class="instance-card" :class="getStatusClass(instance.status)">
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

    <!-- 第二部分：可用实例查询（可折叠） -->
    <el-collapse v-model="activeCollapse" class="available-collapse">
      <el-collapse-item name="available">
        <template #title>
          <div class="collapse-title">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
              <polygon points="12 2 2 7 12 12 22 7 12 2"/>
              <polyline points="2 17 12 22 22 17"/>
              <polyline points="2 12 12 17 22 12"/>
            </svg>
            <span>浏览可用 GPU 实例</span>
            <el-tag v-if="availableTotal" type="success" size="small" class="count-tag">{{ availableTotal }} 个</el-tag>
          </div>
        </template>

        <!-- 筛选器 -->
        <div class="filters-bar">
          <div class="filter-group">
            <el-select v-model="filters.gpuType" placeholder="GPU 类型" clearable @change="onFilterChange" class="gpu-type-select">
              <el-option v-for="g in gpuTypes" :key="g.value" :label="g.label" :value="g.value" />
            </el-select>
          </div>
          <div class="filter-group">
            <el-select v-model="filters.provider" placeholder="供应商" clearable @change="onFilterChange" class="provider-select">
              <el-option v-for="p in providerOptions" :key="p.value" :label="p.label" :value="p.value" />
            </el-select>
          </div>
          <div class="filter-group">
            <el-input-number v-model="filters.maxPrice" :min="0" :step="0.5" placeholder="最高价格" controls-position="right" @change="onFilterChange" />
            <span class="unit">$/h</span>
          </div>
          <div class="filter-group">
            <el-input-number v-model="filters.minGPUs" :min="1" :max="8" placeholder="最少 GPU" controls-position="right" @change="onFilterChange" />
          </div>
          <div class="filter-group">
            <el-select v-model="filters.sortBy" placeholder="排序" @change="onFilterChange">
              <el-option label="价格从低到高" value="price" />
              <el-option label="价格从高到低" value="price_desc" />
              <el-option label="显存从高到低" value="gpu_ram" />
            </el-select>
          </div>
          <div class="filter-hint" v-if="filtersDebouncing">
            <el-icon class="is-loading"><Loading /></el-icon>
            <span>查询中...</span>
          </div>
        </div>

        <!-- 可用实例列表 -->
        <div v-loading="availableLoading" class="available-grid" :style="availableLoading ? 'min-height: 300px' : ''">
          <div v-if="availableInstances.length === 0 && !availableLoading" class="empty-state">
            <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
            </svg>
            <h3>暂无符合条件的资源</h3>
            <p>请调整筛选条件后重试</p>
          </div>

          <div v-for="item in availableInstances" :key="item.id" class="available-card">
            <div class="card-header">
              <div class="provider-badge" :class="item.provider">
                {{ getProviderInitial(item.provider) }}
              </div>
              <div class="price-block">
                <span class="currency">$</span>
                <span class="price">{{ formatPrice(item.price_per_hour) }}</span>
                <span class="unit">/h</span>
              </div>
            </div>

            <div class="gpu-block">
              <span class="gpu-name">{{ item.gpu_name }}</span>
              <span class="gpu-count">x{{ item.num_gpus }}</span>
            </div>

            <div class="spec-row">
              <span class="spec-item">显存: {{ item.gpu_ram_display }}</span>
              <span class="spec-item">磁盘: {{ formatDisk(item.disk_space) }}GB</span>
            </div>
            <div class="spec-row">
              <span class="spec-item">可靠性: {{ (item.reliability * 100).toFixed(0) }}%</span>
              <span class="spec-item">{{ item.location }}</span>
            </div>

            <div class="feature-tags">
              <el-tag v-for="f in item.features" :key="f" size="small" effect="plain">{{ f }}</el-tag>
            </div>

            <div class="card-footer">
              <el-button type="primary" size="small" @click="showLaunchDialog(item)">启动实例</el-button>
            </div>
          </div>
        </div>
      </el-collapse-item>
    </el-collapse>

    <!-- 启动对话框 -->
    <el-dialog v-model="launchDialogVisible" title="启动实例" width="480px">
      <div v-if="selectedOffer" class="launch-form">
        <el-form label-width="100px">
          <el-form-item label="实例名称">
            <el-input v-model="launchForm.name" placeholder="输入实例名称" />
          </el-form-item>
          <el-form-item label="GPU">
            <el-input :value="`${selectedOffer.gpu_name} x${selectedOffer.num_gpus}`" disabled />
          </el-form-item>
          <el-form-item label="镜像">
            <el-select v-model="launchForm.image" filterable allow-create>
              <el-option v-for="img in recommendedImages" :key="img" :label="img" :value="img" />
            </el-select>
          </el-form-item>
          <el-form-item label="磁盘">
            <el-input-number v-model="launchForm.diskSize" :min="20" :max="500" />
            <span class="unit">GB</span>
          </el-form-item>
        </el-form>
        <div class="price-hint">
          预估: <strong>${{ formatPrice(selectedOffer.price_per_hour) }}/小时</strong>
        </div>
      </div>
      <template #footer>
        <el-button @click="launchDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="launchInstance" :loading="launching">启动</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'

// ========== 状态 ==========
const myInstances = ref([])
const myInstancesLoading = ref(false)

const availableInstances = ref([])
const availableLoading = ref(false)
const availableTotal = ref(0)
const filtersDebouncing = ref(false)

const activeCollapse = ref([]) // 默认折叠

const gpuTypes = ref([])
const providerOptions = ref([])

const filters = ref({
  gpuType: '',
  provider: '',
  maxPrice: null,
  minGPUs: 1,
  sortBy: 'price'
})

const launchDialogVisible = ref(false)
const selectedOffer = ref(null)
const launching = ref(false)
const launchForm = ref({
  name: '',
  image: 'pytorch/pytorch:2.1.0-cuda11.8-cudnn8-runtime',
  diskSize: 100
})

const recommendedImages = [
  'pytorch/pytorch:2.1.0-cuda11.8-cudnn8-runtime',
  'pytorch/pytorch:2.2.0-cuda12.1-cudnn8-runtime',
  'tensorflow/tensorflow:2.15.0-gpu',
  'nvcr.io/nvidia/pytorch:23.10-py3'
]

// 轮询定时器
let pollTimer = null
let debounceTimer = null

// ========== 方法 ==========

// 我的实例
const fetchMyInstances = async () => {
  myInstancesLoading.value = true
  try {
    const userId = 'default'
    const res = await fetch(`/api/v1/offers?user_id=${userId}`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    // 过滤掉已销毁的实例，不在列表中显示
    const instances = (data.instances || []).filter(i => i.status !== 'destroyed')
    myInstances.value = instances
  } catch (err) {
    console.error('fetchMyInstances failed:', err)
    ElMessage.error('获取实例列表失败')
    myInstances.value = []
  } finally {
    myInstancesLoading.value = false
  }
}

// 可用实例
const fetchAvailableInstances = async () => {
  availableLoading.value = true
  filtersDebouncing.value = false
  try {
    const params = new URLSearchParams()
    if (filters.value.gpuType) params.append('gpu_type', filters.value.gpuType)
    if (filters.value.provider) params.append('provider', filters.value.provider)
    if (filters.value.maxPrice) params.append('max_price', filters.value.maxPrice)
    if (filters.value.minGPUs) params.append('min_gpus', filters.value.minGPUs)

    const res = await fetch(`/api/v1/gpu/instances?${params}`)
    const data = await res.json()

    let list = (data.instances || []).map(inst => ({
      id: inst.id,
      provider: inst.provider === 'vast.ai' ? 'vastai' : inst.provider,
      provider_name: inst.provider,
      gpu_name: inst.gpu_type,
      num_gpus: inst.num_gpus,
      gpu_ram_display: inst.gpu_ram >= 1024 ? `${(inst.gpu_ram / 1024).toFixed(0)}GB` : `${inst.gpu_ram}MB`,
      price_per_hour: inst.price,
      currency: 'USD',
      location: inst.location || '未知地区',
      reliability: inst.reliability,
      disk_space: inst.disk_space,
      features: [
        inst.reliability > 0.95 ? '高可靠性' : null,
        inst.disk_space >= 100 ? '大存储' : null
      ].filter(Boolean),
      image: 'pytorch/pytorch:2.1.0-cuda11.8-cudnn8-runtime'
    }))

    // 客户端排序
    if (filters.value.sortBy === 'price') {
      list.sort((a, b) => a.price_per_hour - b.price_per_hour)
    } else if (filters.value.sortBy === 'price_desc') {
      list.sort((a, b) => b.price_per_hour - a.price_per_hour)
    } else if (filters.value.sortBy === 'gpu_ram') {
      list.sort((a, b) => {
        const ra = parseFloat(a.gpu_ram_display) || 0
        const rb = parseFloat(b.gpu_ram_display) || 0
        return rb - ra
      })
    }

    availableInstances.value = list
    availableTotal.value = list.length

    // 更新 GPU 类型下拉选项（从返回数据中提取）
    const gpuSet = new Set(list.map(o => o.gpu_name))
    gpuTypes.value = Array.from(gpuSet).map(g => ({ value: g, label: g }))
  } catch (err) {
    console.error('Failed to fetch available instances:', err)
    ElMessage.error('获取可用实例失败')
  } finally {
    availableLoading.value = false
  }
}

// 防抖筛选
const onFilterChange = () => {
  filtersDebouncing.value = true
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    fetchAvailableInstances()
  }, 500)
}

// 刷新全部
const refreshAll = () => {
  fetchMyInstances()
  fetchAvailableInstances()
  ElMessage.success('已刷新')
}

// 格式化
const formatPrice = (price) => {
  const n = parseFloat(price)
  return isNaN(n) ? '0.00' : n.toFixed(2)
}

const formatDisk = (disk) => {
  const n = parseFloat(disk)
  return isNaN(n) ? '0.0' : n.toFixed(1)
}

const getProviderInitial = (p) => {
  const map = { vastai: 'V', autodl: 'A', ppio: 'P', local: 'L' }
  return map[p] || p?.[0]?.toUpperCase() || '?'
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

// 实例操作
const stopInstance = async (instance) => {
  try {
    const res = await fetch(`/api/v1/offers/${instance.id}/stop`, { method: 'POST' })
    if (!res.ok) throw new Error()
    ElMessage.success('实例已停止')
    fetchMyInstances()
  } catch (err) {
    ElMessage.error('停止失败')
  }
}

const startInstance = async (instance) => {
  try {
    const res = await fetch(`/api/v1/offers/${instance.id}/start`, { method: 'POST' })
    if (!res.ok) throw new Error()
    ElMessage.success('实例启动中')
    fetchMyInstances()
  } catch (err) {
    ElMessage.error('启动失败')
  }
}

const destroyInstance = async (instance) => {
  try {
    const res = await fetch(`/api/v1/offers/${instance.id}`, { method: 'DELETE' })
    if (!res.ok) throw new Error()
    ElMessage.success('实例已销毁')
    myInstances.value = myInstances.value.filter(i => i.id !== instance.id)
  } catch (err) {
    ElMessage.error('销毁失败')
  }
}

// 启动对话框
const showLaunchDialog = (offer) => {
  selectedOffer.value = offer
  launchForm.value.name = `inst-${Date.now().toString(36).slice(-6)}`
  launchDialogVisible.value = true
}

const launchInstance = async () => {
  if (!launchForm.value.name) {
    ElMessage.warning('请输入实例名称')
    return
  }
  launching.value = true
  try {
    const res = await fetch('/api/v1/offers', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        provider: selectedOffer.value.provider,
        offer_id: selectedOffer.value.id,
        user_id: 'default',
        name: launchForm.value.name,
        image: launchForm.value.image,
        disk_size: launchForm.value.diskSize,
        duration_hours: 24,
        price_per_hour: selectedOffer.value.price_per_hour,
        gpu_name: selectedOffer.value.gpu_name,
        num_gpus: selectedOffer.value.num_gpus,
        location: selectedOffer.value.location
      })
    })
    if (!res.ok) throw new Error('启动失败')
    const data = await res.json()
    myInstances.value.unshift(data)
    launchDialogVisible.value = false
    ElMessage.success('租赁已创建，实例启动中...')
  } catch (err) {
    console.error(err)
    ElMessage.error('启动实例失败')
  } finally {
    launching.value = false
  }
}

// 获取可用 Provider 列表
const fetchProviders = async () => {
  try {
    const res = await fetch('/api/v1/gpu/providers')
    if (!res.ok) return
    const data = await res.json()
    const providers = data.providers || []
    const names = { vastai: 'Vast.ai', autodl: 'AutoDL', ppio: 'PPIO', local: '本地' }
    providerOptions.value = providers.map(p => ({
      value: p.name,
      label: names[p.name] || p.name
    }))
  } catch (err) {
    console.error('Failed to fetch providers:', err)
  }
}

// 生命周期
onMounted(() => {
  fetchMyInstances()
  fetchProviders()
  fetchAvailableInstances()
  pollTimer = setInterval(() => {
    fetchMyInstances()
  }, 30000)
})

onUnmounted(() => {
  clearTimeout(debounceTimer)
  clearInterval(pollTimer)
})
</script>

<style scoped>
.instances-page {
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
  font-weight: 600;
  color: var(--text-primary);
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

/* 区块 */
.section {
  margin-bottom: 32px;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.section-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
}

/* 我的实例卡片 */
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
  transition: all 0.2s;
}

.instance-card.status-running { border-color: #10b981; }
.instance-card.status-starting { border-color: #f59e0b; }
.instance-card.status-destroying { border-color: #ef4444; opacity: 0.7; }
.instance-card.status-error { border-color: #ef4444; }

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
  font-weight: 600;
  color: var(--text-primary);
  font-size: 15px;
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

.ssh-block {
  background: var(--bg-primary);
  border-radius: var(--radius-md);
  padding: 10px 12px;
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.ssh-label {
  color: var(--text-secondary);
  white-space: nowrap;
}

.ssh-cmd {
  flex: 1;
  font-family: monospace;
  color: var(--primary-color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-actions {
  display: flex;
  gap: 8px;
}

/* 折叠面板 */
.available-collapse {
  margin-top: 8px;
}

:deep(.available-collapse .el-collapse-item__header) {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.collapse-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.count-tag {
  margin-left: 8px;
}

/* 筛选器 */
.filters-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding: 12px;
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 6px;
}

.gpu-type-select {
  width: 180px;
}

.provider-select {
  width: 150px;
}

.unit {
  font-size: 12px;
  color: var(--text-secondary);
}

.filter-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-secondary);
  margin-left: auto;
}

/* 可用实例卡片 */
.available-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.available-card {
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  padding: 16px;
  border: 1px solid var(--border-color);
  transition: all 0.2s;
}

.available-card:hover {
  border-color: var(--primary-color);
}

.provider-badge {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 14px;
  color: white;
}

.provider-badge.vastai { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
.provider-badge.autodl { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); }
.provider-badge.ppio { background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); }
.provider-badge.local { background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%); }

.price-block {
  text-align: right;
}

.price-block .currency { font-size: 12px; color: var(--text-secondary); }
.price-block .price { font-size: 22px; font-weight: 700; color: var(--primary-color); }
.price-block .unit { font-size: 12px; color: var(--text-secondary); }

.gpu-block {
  display: flex;
  align-items: baseline;
  gap: 6px;
  margin: 10px 0;
  padding: 8px 0;
  border-top: 1px solid var(--border-color);
  border-bottom: 1px solid var(--border-color);
}

.gpu-name { font-weight: 600; color: var(--text-primary); font-size: 15px; }
.gpu-count { font-size: 14px; color: var(--text-secondary); }

.spec-row {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 4px;
}

.feature-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 10px 0;
}

.card-footer {
  margin-top: 8px;
}

/* 空状态 */
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
}

/* 对话框 */
.launch-form .unit {
  margin-left: 8px;
}

.price-hint {
  margin-top: 12px;
  text-align: right;
  font-size: 14px;
  color: var(--text-secondary);
}

.price-hint strong {
  color: var(--primary-color);
}
</style>
