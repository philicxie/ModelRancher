<template>
  <div class="browse-page">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">浏览 GPU 实例</h1>
        <p class="page-subtitle">搜索并租赁可用的 GPU 资源</p>
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
      </div>
    </div>

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
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'

const router = useRouter()

const availableInstances = ref([])
const availableLoading = ref(false)
const availableTotal = ref(0)
const filtersDebouncing = ref(false)

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

let debounceTimer = null

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

    const gpuSet = new Set(list.map(o => o.gpu_name))
    gpuTypes.value = Array.from(gpuSet).map(g => ({ value: g, label: g }))
  } catch (err) {
    console.error('Failed to fetch available instances:', err)
    ElMessage.error('获取可用实例失败')
  } finally {
    availableLoading.value = false
  }
}

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

const onFilterChange = () => {
  filtersDebouncing.value = true
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    fetchAvailableInstances()
  }, 500)
}

const refresh = () => {
  fetchAvailableInstances()
  ElMessage.success('已刷新')
}

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
    launchDialogVisible.value = false
    ElMessage.success('租赁已创建，实例启动中...')
    router.push('/my-instances')
  } catch (err) {
    console.error(err)
    ElMessage.error('启动实例失败')
  } finally {
    launching.value = false
  }
}

onMounted(() => {
  fetchProviders()
  fetchAvailableInstances()
})
</script>

<style scoped>
.browse-page {
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

.available-card .card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
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
