<template>
  <div class="task-create-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">创建训练任务</h1>
        <p class="page-subtitle">配置训练环境、算力资源和存储路径</p>
      </div>
    </div>

    <el-form :model="form" :rules="rules" ref="formRef" label-position="top" class="create-form">
      <!-- 基本信息 -->
      <section class="form-section">
        <h2 class="section-title">
          <span class="section-num">1</span>
          基本信息
        </h2>
        <div class="section-body">
          <el-form-item label="任务名称" prop="name">
            <el-input v-model="form.name" placeholder="给任务起一个描述性的名称" size="large" clearable />
          </el-form-item>
          <el-form-item label="任务描述">
            <el-input v-model="form.description" type="textarea" :rows="3" placeholder="描述这个训练任务的目的和内容（可选）" size="large" />
          </el-form-item>
        </div>
      </section>

      <!-- 算力配置 -->
      <section class="form-section">
        <h2 class="section-title">
          <span class="section-num">2</span>
          算力配置
        </h2>
        <div class="section-body">
          <!-- 执行模式切换 -->
          <el-form-item label="执行模式">
            <div class="mode-switch">
              <div
                class="mode-btn"
                :class="{ active: !isRemote }"
                @click="setLocalMode"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18">
                  <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
                  <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
                </svg>
                本地执行
              </div>
              <div
                class="mode-btn"
                :class="{ active: isRemote }"
                @click="setRemoteMode"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18">
                  <path d="M12 2L2 7l10 5 10-5-10-5z"/>
                  <path d="M2 17l10 5 10-5"/>
                  <path d="M2 12l10 5 10-5"/>
                </svg>
                远程实例
              </div>
            </div>
          </el-form-item>

          <!-- 本地执行提示 -->
          <div v-if="!isRemote" class="local-hint">
            任务将在当前服务器上以本地 Docker 容器运行，适合调试和小规模测试。
          </div>

          <!-- 远程实例搜索 -->
          <template v-if="isRemote">
            <!-- 已选实例摘要 -->
            <div v-if="selectedInstance" class="selected-instance-bar">
              <div class="selected-info">
                <span class="selected-badge">已选择</span>
                <span class="selected-gpu">{{ selectedInstance.gpu_name }} × {{ selectedInstance.num_gpus }}</span>
                <span class="selected-price">${{ formatPrice(selectedInstance.price_per_hour) }}/h</span>
                <span class="selected-provider" :class="selectedInstance.provider">{{ selectedInstance.provider }}</span>
              </div>
              <el-button link type="danger" size="small" @click="clearSelection">取消选择</el-button>
            </div>

            <!-- 筛选器 -->
            <div class="instance-filters">
              <el-select v-model="filters.gpuType" placeholder="GPU 类型" clearable @change="onFilterChange" size="default" style="width: 160px">
                <el-option v-for="g in gpuTypeOptions" :key="g" :label="g" :value="g" />
              </el-select>
              <el-select v-model="filters.provider" placeholder="供应商" clearable @change="onFilterChange" size="default" style="width: 130px">
                <el-option v-for="p in providerOptions" :key="p.value" :label="p.label" :value="p.value" />
              </el-select>
              <el-input-number v-model="filters.maxPrice" :min="0" :step="0.5" placeholder="最高价格 $/h" controls-position="right" @change="onFilterChange" size="default" style="width: 140px" />
              <el-input-number v-model="filters.minGPUs" :min="1" :max="8" placeholder="最少 GPU" controls-position="right" @change="onFilterChange" size="default" style="width: 110px" />
              <el-select v-model="filters.sortBy" placeholder="排序" @change="onFilterChange" size="default" style="width: 140px">
                <el-option label="价格从低到高" value="price" />
                <el-option label="价格从高到低" value="price_desc" />
                <el-option label="显存从高到低" value="gpu_ram" />
              </el-select>
              <el-button size="default" @click="fetchInstances">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                  <polyline points="23 4 23 10 17 10"/>
                  <polyline points="1 20 1 14 7 14"/>
                  <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                </svg>
                刷新
              </el-button>
            </div>

            <!-- 实例列表 -->
            <div v-loading="instancesLoading" class="instance-grid">
              <div v-if="instances.length === 0 && !instancesLoading" class="instance-empty">
                <p>暂无符合条件的实例，请调整筛选条件</p>
              </div>

              <div
                v-for="inst in instances"
                :key="inst.id"
                class="instance-card"
                :class="{ selected: selectedInstance?.id === inst.id }"
                @click="selectInstance(inst)"
              >
                <div class="inst-header">
                  <div class="inst-provider" :class="inst.provider">{{ getProviderInitial(inst.provider) }}</div>
                  <div class="inst-price">
                    <span class="inst-price-num">${{ formatPrice(inst.price_per_hour) }}</span>
                    <span class="inst-price-unit">/h</span>
                  </div>
                </div>
                <div class="inst-gpu">
                  {{ inst.gpu_name }}
                  <span class="inst-gpu-count">×{{ inst.num_gpus }}</span>
                </div>
                <div class="inst-specs">
                  <span>显存 {{ inst.gpu_ram_display }}</span>
                  <span>磁盘 {{ formatDisk(inst.disk_space) }}GB</span>
                </div>
                <div class="inst-specs">
                  <span>可靠 {{ (inst.reliability * 100).toFixed(0) }}%</span>
                  <span>{{ inst.location }}</span>
                </div>
                <div v-if="selectedInstance?.id === inst.id" class="inst-selected-mark">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" width="14" height="14">
                    <polyline points="20 6 9 17 4 12"/>
                  </svg>
                  已选择
                </div>
              </div>
            </div>

            <!-- 时长和磁盘配置（选中实例后显示） -->
            <div v-if="selectedInstance" class="resource-grid" style="margin-top: 16px">
              <el-form-item label="磁盘大小 (GB)">
                <el-input-number v-model="form.disk_size" :min="10" :max="1000" :step="10" size="large" style="width: 100%" />
              </el-form-item>
              <el-form-item label="运行时长 (小时)">
                <el-input-number v-model="form.duration_hours" :min="1" :max="168" size="large" style="width: 100%" />
              </el-form-item>
            </div>
          </template>
        </div>
      </section>

      <!-- 环境配置 -->
      <section class="form-section">
        <h2 class="section-title">
          <span class="section-num">3</span>
          环境配置
        </h2>
        <div class="section-body">
          <el-form-item label="训练镜像" prop="image">
            <div class="image-select-trigger" @click="openImageSelector" width="1200">
              <el-input
                v-model="form.image"
                placeholder="点击选择镜像"
                size="large"
                readonly
                class="image-readonly-input"
              >
                <template #suffix>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                    <path d="M6 9l6 6 6-6"/>
                  </svg>
                </template>
              </el-input>
            </div>
          </el-form-item>

          <el-form-item label="执行命令" prop="command">
            <el-input
              v-model="form.command"
              type="textarea"
              :rows="4"
              placeholder="python train.py --data-dir /data --output-dir /output --epochs 100"
              size="large"
            />
          </el-form-item>

          <el-form-item label="环境变量">
            <div class="env-vars-grid">
              <div v-for="(env, i) in form.env_vars" :key="i" class="env-row">
                <el-input v-model="form.env_vars[i]" placeholder="KEY=value" size="default" style="width: 100%">
                  <template #append>
                    <el-button @click="removeEnv(i)">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                        <line x1="18" y1="6" x2="6" y2="18"/>
                        <line x1="6" y1="6" x2="18" y2="18"/>
                      </svg>
                    </el-button>
                  </template>
                </el-input>
              </div>
              <el-button link type="primary" @click="addEnv" class="add-env-btn">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                  <line x1="12" y1="5" x2="12" y2="19"/>
                  <line x1="5" y1="12" x2="19" y2="12"/>
                </svg>
                添加环境变量
              </el-button>
            </div>
          </el-form-item>
        </div>
      </section>

      <!-- 存储配置 -->
      <section class="form-section">
        <h2 class="section-title">
          <span class="section-num">4</span>
          存储配置
        </h2>
        <div class="section-body">
          <div class="storage-bindings">
            <div
              v-for="(binding, i) in form.storage_bindings"
              :key="i"
              class="storage-binding-row"
            >
              <el-select v-model="binding.type" size="default" style="width: 100px; flex-shrink: 0;">
                <el-option label="输入" value="input" />
                <el-option label="输出" value="output" />
              </el-select>
              <el-input
                v-model="binding.env_name"
                placeholder="环境变量名"
                size="default"
                style="width: 160px; flex-shrink: 0;"
              />
              <div class="storage-path-trigger" @click="openStorageSelector(i)">
                <el-input
                  v-model="binding.path"
                  placeholder="点击选择存储路径"
                  size="default"
                  readonly
                  style="width: 100%"
                >
                  <template #suffix>
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                      <path d="M6 9l6 6 6-6"/>
                    </svg>
                  </template>
                </el-input>
              </div>
              <el-button link type="danger" size="small" @click="removeStorageBinding(i)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                  <line x1="18" y1="6" x2="6" y2="18"/>
                  <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </el-button>
            </div>
          </div>
          <el-button link type="primary" @click="addStorageBinding" class="add-storage-btn">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
              <line x1="12" y1="5" x2="12" y2="19"/>
              <line x1="5" y1="12" x2="19" y2="12"/>
            </svg>
            添加存储映射
          </el-button>
          <div class="field-hint" style="margin-top: 8px;">
            输入路径会在任务运行前下载到执行环境，输出路径会在任务完成后上传到对象存储。环境变量可在训练命令中使用。
          </div>
        </div>
      </section>

      <!-- 底部操作 -->
      <div class="form-footer">
        <el-button size="large" @click="router.push('/tasks')">取消</el-button>
        <el-button type="primary" size="large" @click="showConfirm = true">
          创建任务
        </el-button>
      </div>
    </el-form>

    <!-- 镜像选择弹窗 -->
    <el-dialog v-model="showImageSelector" title="选择训练镜像" width="640px" :close-on-click-modal="false">
      <div class="image-selector-tabs">
        <div
          v-for="tab in imageTabs"
          :key="tab.key"
          class="image-tab"
          :class="{ active: activeImageTab === tab.key }"
          @click="activeImageTab = tab.key"
        >
          {{ tab.label }}
        </div>
      </div>

      <div v-if="imageLoading" class="image-loading">
        <el-skeleton :rows="4" animated />
      </div>

      <!-- 我的收藏 -->
      <div v-else-if="activeImageTab === 'favorites'" class="image-list">
        <div v-if="favoriteImages.length === 0" class="image-list-empty">
          <p>暂无收藏镜像</p>
          <span>前往镜像管理页面搜索并收藏公开镜像</span>
        </div>
        <div
          v-for="img in favoriteImages"
          :key="img.id"
          class="image-list-item"
          :class="{ selected: form.image === img.image_name }"
          @click="pickImage(img.image_name)"
        >
          <div class="image-list-name">{{ img.image_name }}</div>
          <div class="image-list-desc">{{ img.description || '暂无描述' }}</div>
          <div class="image-list-meta">
            <span v-if="img.is_official" class="official-tag">官方</span>
            <span class="star-count">
              <svg viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
                <path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z"/>
              </svg>
              {{ formatStars(img.star_count) }}
            </span>
          </div>
        </div>
      </div>

      <!-- 私有镜像 -->
      <div v-else-if="activeImageTab === 'private'" class="image-list">
        <div v-if="privateImages.length === 0" class="image-list-empty">
          <p>暂无私有镜像</p>
        </div>
        <div
          v-for="img in privateImages"
          :key="img.name"
          class="image-list-item"
          :class="{ selected: form.image === img.name }"
          @click="pickImage(img.name)"
        >
          <div class="image-list-name">{{ img.name }}</div>
          <div class="image-list-desc">{{ img.description || '暂无描述' }}</div>
          <div class="image-list-meta">
            <span class="private-tag">私有</span>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="showImageSelector = false">取消</el-button>
      </template>
    </el-dialog>

    <!-- 确认弹窗 -->
    <el-dialog v-model="showConfirm" title="确认任务配置" width="560px" :close-on-click-modal="false">
      <div class="confirm-body">
        <div class="confirm-section">
          <h4>基本信息</h4>
          <div class="confirm-row"><span class="confirm-label">任务名称</span><span class="confirm-value">{{ form.name || '-' }}</span></div>
          <div class="confirm-row"><span class="confirm-label">描述</span><span class="confirm-value">{{ form.description || '-' }}</span></div>
        </div>

        <div class="confirm-section">
          <h4>算力配置</h4>
          <div class="confirm-row"><span class="confirm-label">执行模式</span><span class="confirm-value">{{ providerLabel }}</span></div>
          <template v-if="form.provider !== ''">
            <div class="confirm-row"><span class="confirm-label">GPU</span><span class="confirm-value">{{ form.gpu_name }} × {{ form.num_gpus }}</span></div>
            <div class="confirm-row"><span class="confirm-label">磁盘</span><span class="confirm-value">{{ form.disk_size }} GB</span></div>
            <div class="confirm-row"><span class="confirm-label">时长</span><span class="confirm-value">{{ form.duration_hours }} 小时</span></div>
          </template>
        </div>

        <div class="confirm-section">
          <h4>环境配置</h4>
          <div class="confirm-row"><span class="confirm-label">镜像</span><span class="confirm-value code">{{ form.image }}</span></div>
          <div class="confirm-row"><span class="confirm-label">命令</span><span class="confirm-value code">{{ form.command }}</span></div>
          <div class="confirm-row" v-if="form.env_vars.length > 0">
            <span class="confirm-label">环境变量</span>
            <div class="confirm-value env-list">
              <div v-for="(env, i) in form.env_vars" :key="i" class="env-tag">{{ env }}</div>
            </div>
          </div>
        </div>

        <div class="confirm-section">
          <h4>存储配置</h4>
          <div v-for="(binding, i) in form.storage_bindings" :key="i" class="confirm-row">
            <span class="confirm-label">{{ binding.type === 'input' ? '输入' : '输出' }}</span>
            <div class="confirm-value">
              <code class="storage-binding-confirm">
                {{ binding.env_name }} = {{ binding.path || '-' }}
              </code>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="showConfirm = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="loading">确认创建</el-button>
      </template>
    </el-dialog>

    <!-- 存储路径选择弹窗 -->
    <StoragePathSelector
      v-model="showStorageSelector"
      @select="onStorageSelect"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useTaskStore } from '../stores/task'
import { ElMessage } from 'element-plus'
import StoragePathSelector from '../components/StoragePathSelector.vue'

const router = useRouter()
const route = useRoute()
const taskStore = useTaskStore()
const formRef = ref(null)
const loading = ref(false)
const showConfirm = ref(false)

const form = reactive({
  name: '',
  description: '',
  provider: '',
  offer_id: '',
  gpu_name: 'CPU',
  num_gpus: 0,
  disk_size: 50,
  duration_hours: 1,
  price_per_hour: 0,
  image: 'pytorch/pytorch:2.0.1-cuda11.7-cudnn8-runtime',
  command: '',
  env_vars: [],
  storage_bindings: [
    { type: 'input', env_name: 'DATA_PATH', path: '' },
    { type: 'output', env_name: 'OUTPUT_PATH', path: '' }
  ]
})

// 实例搜索
const isRemote = ref(false)
const instances = ref([])
const instancesLoading = ref(false)
const selectedInstance = ref(null)
const gpuTypeOptions = ref([])
const providerOptions = ref([])
const filters = reactive({
  gpuType: '',
  provider: '',
  maxPrice: null,
  minGPUs: 1,
  sortBy: 'price'
})

const rules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  image: [{ required: true, message: '请选择或输入镜像', trigger: 'blur' }],
  command: [{ required: true, message: '请输入训练命令', trigger: 'blur' }]
}

// 镜像选择
const showImageSelector = ref(false)
const activeImageTab = ref('favorites')
const imageTabs = [
  { key: 'favorites', label: '我的收藏' },
  { key: 'private', label: '私有镜像' }
]
const favoriteImages = ref([])
const privateImages = ref([])
const imageLoading = ref(false)

const providerLabel = computed(() => {
  if (!isRemote.value) return '本地执行'
  if (!selectedInstance.value) return '远程实例（未选择）'
  return `${selectedInstance.value.provider} — ${selectedInstance.value.gpu_name} × ${selectedInstance.value.num_gpus}`
})

// 实例搜索
const setLocalMode = () => {
  isRemote.value = false
  selectedInstance.value = null
  form.provider = ''
  form.offer_id = ''
}

const setRemoteMode = () => {
  isRemote.value = true
  if (instances.value.length === 0) {
    fetchInstances()
  }
}

let debounceTimer = null
const onFilterChange = () => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => fetchInstances(), 400)
}

const fetchInstances = async () => {
  instancesLoading.value = true
  try {
    const params = new URLSearchParams()
    if (filters.gpuType) params.append('gpu_type', filters.gpuType)
    if (filters.provider) params.append('provider', filters.provider)
    if (filters.maxPrice) params.append('max_price', filters.maxPrice)
    if (filters.minGPUs) params.append('min_gpus', filters.minGPUs)

    const res = await fetch(`/api/v1/gpu/instances?${params}`)
    const data = await res.json()

    let list = (data.instances || []).map(inst => ({
      id: inst.id,
      provider: inst.provider === 'vast.ai' ? 'vastai' : inst.provider,
      gpu_name: inst.gpu_type,
      num_gpus: inst.num_gpus,
      gpu_ram_display: inst.gpu_ram >= 1024 ? `${(inst.gpu_ram / 1024).toFixed(0)}GB` : `${inst.gpu_ram}MB`,
      price_per_hour: inst.price,
      location: inst.location || '未知地区',
      reliability: inst.reliability,
      disk_space: inst.disk_space,
    }))

    if (filters.sortBy === 'price') {
      list.sort((a, b) => a.price_per_hour - b.price_per_hour)
    } else if (filters.sortBy === 'price_desc') {
      list.sort((a, b) => b.price_per_hour - a.price_per_hour)
    } else if (filters.sortBy === 'gpu_ram') {
      list.sort((a, b) => {
        const ra = parseFloat(a.gpu_ram_display) || 0
        const rb = parseFloat(b.gpu_ram_display) || 0
        return rb - ra
      })
    }

    instances.value = list

    // 提取去重 GPU 类型
    const gpuSet = new Set(list.map(o => o.gpu_name))
    gpuTypeOptions.value = Array.from(gpuSet)
  } catch (err) {
    console.error('Failed to fetch instances:', err)
  } finally {
    instancesLoading.value = false
  }
}

const fetchProviderOptions = async () => {
  try {
    const res = await fetch('/api/v1/gpu/providers')
    if (!res.ok) return
    const data = await res.json()
    const names = { vastai: 'Vast.ai', autodl: 'AutoDL', ppio: 'PPIO', local: '本地' }
    providerOptions.value = (data.providers || []).map(p => ({
      value: p.name,
      label: names[p.name] || p.name
    }))
  } catch (err) {
    console.error('Failed to fetch providers:', err)
  }
}

const openImageSelector = async () => {
  showImageSelector.value = true
  imageLoading.value = true
  try {
    const [favRes, privRes] = await Promise.all([
      fetch(`/api/v1/images/favorites?user_id=default`),
      fetch('/api/v1/images/private?provider=ppio')
    ])
    if (favRes.ok) {
      const favData = await favRes.json()
      favoriteImages.value = favData.favorites || []
    }
    if (privRes.ok) {
      const privData = await privRes.json()
      privateImages.value = privData.images || []
    }
  } catch (err) {
    console.error('Failed to load images:', err)
    ElMessage.error('加载镜像列表失败')
  } finally {
    imageLoading.value = false
  }
}

const pickImage = (imageName) => {
  form.image = imageName
  showImageSelector.value = false
  ElMessage.success(`已选择镜像: ${imageName}`)
}

const formatStars = (count) => {
  if (count >= 1000000) return (count / 1000000).toFixed(1) + 'M'
  if (count >= 1000) return (count / 1000).toFixed(1) + 'k'
  return count.toString()
}

const selectInstance = (inst) => {
  selectedInstance.value = inst
  form.provider = inst.provider
  form.offer_id = inst.id
  form.gpu_name = inst.gpu_name
  form.num_gpus = inst.num_gpus
  form.price_per_hour = inst.price_per_hour
  form.disk_size = Math.max(form.disk_size, Math.ceil(inst.disk_space / 10) * 10)
}

const clearSelection = () => {
  selectedInstance.value = null
  form.provider = ''
  form.offer_id = ''
  form.gpu_name = 'CPU'
  form.num_gpus = 0
  form.price_per_hour = 0
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

const addEnv = () => {
  form.env_vars.push('')
}

const removeEnv = (index) => {
  form.env_vars.splice(index, 1)
}

// 存储路径选择器
const showStorageSelector = ref(false)
const currentStorageIndex = ref(0)

const openStorageSelector = (index) => {
  currentStorageIndex.value = index
  showStorageSelector.value = true
}

const onStorageSelect = (path) => {
  form.storage_bindings[currentStorageIndex.value].path = path
}

const addStorageBinding = () => {
  form.storage_bindings.push({ type: 'input', env_name: '', path: '' })
}

const removeStorageBinding = (index) => {
  form.storage_bindings.splice(index, 1)
}

const handleSubmit = async () => {
  try {
    await formRef.value.validate()
  } catch {
    showConfirm.value = false
    ElMessage.error('请完善表单信息')
    return
  }

  try {
    loading.value = true
    // 从 storage_bindings 提取兼容字段
    const firstInput = form.storage_bindings.find(b => b.type === 'input')
    const firstOutput = form.storage_bindings.find(b => b.type === 'output')

    const taskData = {
      name: form.name,
      description: form.description,
      image: form.image,
      command: form.command,
      data_path: firstInput?.path || '',
      output_path: firstOutput?.path || '',
      env_vars: form.env_vars.filter(Boolean),
      storage_bindings: form.storage_bindings.filter(b => b.path && b.env_name),
      provider: form.provider,
      offer_id: form.offer_id,
      gpu_name: form.gpu_name,
      num_gpus: form.num_gpus,
      disk_size: form.disk_size,
      duration_hours: form.duration_hours
    }

    await taskStore.createTask(taskData)
    showConfirm.value = false
    ElMessage.success('任务创建成功')
    router.push('/tasks')
  } catch (err) {
    ElMessage.error('创建失败: ' + (err.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchProviderOptions()
  loadCopiedTask()
})

const loadCopiedTask = () => {
  const raw = sessionStorage.getItem('taskCopyData')
  if (!raw) return
  try {
    const data = JSON.parse(raw)
    sessionStorage.removeItem('taskCopyData')

    if (data.name) form.name = data.name
    if (data.description !== undefined) form.description = data.description
    if (data.image) form.image = data.image
    if (data.command !== undefined) form.command = data.command
    if (Array.isArray(data.env_vars)) form.env_vars = [...data.env_vars]

    // 算力配置留空，强制本地模式让用户重新选择
    setLocalMode()

    // 存储绑定：优先使用复制的，没有则根据 data_path/output_path 回退
    if (Array.isArray(data.storage_bindings) && data.storage_bindings.length > 0) {
      form.storage_bindings = data.storage_bindings.map(b => ({
        type: b.type || 'input',
        env_name: b.env_name || '',
        path: b.path || ''
      }))
    } else if (data.data_path || data.output_path) {
      const bindings = []
      if (data.data_path) {
        bindings.push({ type: 'input', env_name: 'DATA_PATH', path: data.data_path })
      }
      if (data.output_path) {
        bindings.push({ type: 'output', env_name: 'OUTPUT_PATH', path: data.output_path })
      }
      if (bindings.length > 0) form.storage_bindings = bindings
    }
  } catch (err) {
    console.error('Failed to load copied task:', err)
  }
}
</script>

<style scoped>
.task-create-page {
  width: 100%;
  margin: 0 auto;
  padding: 0 40px;
  box-sizing: border-box;
}

.page-header {
  margin-bottom: 32px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.03em;
  margin-bottom: 4px;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
}

/* 表单分区 */
.form-section {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  margin-bottom: 24px;
  overflow: hidden;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 24px;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  background: rgba(79, 70, 229, 0.03);
  border-bottom: 1px solid var(--border-color);
}

.section-num {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--primary-color);
  color: white;
  font-size: 13px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.section-body {
  padding: 24px;
}

/* 模式切换 */
.mode-switch {
  display: flex;
  gap: 8px;
}

.mode-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  border: 2px solid var(--border-color);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.2s;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  background: var(--bg-primary);
}

.mode-btn:hover {
  border-color: var(--primary-color);
}

.mode-btn.active {
  border-color: var(--primary-color);
  background: rgba(79, 70, 229, 0.05);
  color: var(--primary-color);
}

.local-hint {
  padding: 16px;
  background: rgba(79, 70, 229, 0.04);
  border-radius: var(--radius-md);
  font-size: 13px;
  color: var(--text-secondary);
  border: 1px dashed var(--border-color);
}

/* 已选实例摘要 */
.selected-instance-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: rgba(79, 70, 229, 0.06);
  border: 1px solid var(--primary-color);
  border-radius: var(--radius-md);
  margin-bottom: 16px;
}

.selected-info {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.selected-badge {
  background: var(--primary-color);
  color: white;
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 4px;
}

.selected-gpu {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 14px;
}

.selected-price {
  color: #2563eb;
  font-weight: 600;
  font-size: 14px;
}

.selected-provider {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 4px;
  color: white;
  text-transform: uppercase;
}

.selected-provider.vastai { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
.selected-provider.autodl { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); }
.selected-provider.ppio { background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); }
.selected-provider.local { background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%); }

/* 实例筛选器 */
.instance-filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  padding: 12px;
  background: var(--bg-primary);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
}

/* 实例网格 */
.instance-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
  max-height: 400px;
  overflow-y: auto;
  padding-right: 4px;
}

.instance-empty {
  grid-column: 1 / -1;
  text-align: center;
  padding: 32px;
  color: var(--text-secondary);
  font-size: 13px;
}

.instance-card {
  padding: 14px;
  background: var(--bg-primary);
  border: 2px solid var(--border-color);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
}

.instance-card:hover {
  border-color: rgba(56, 189, 248, 0.5);
  transform: translateY(-1px);
}

.instance-card.selected {
  border-color: var(--primary-color);
  background: rgba(79, 70, 229, 0.04);
}

.inst-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.inst-provider {
  width: 26px;
  height: 26px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 12px;
  color: white;
}

.inst-provider.vastai { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
.inst-provider.autodl { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); }
.inst-provider.ppio { background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); }
.inst-provider.local { background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%); }

.inst-price-num {
  font-size: 18px;
  font-weight: 700;
  color: #2563eb;
}

.inst-price-unit {
  font-size: 11px;
  color: var(--text-secondary);
}

.inst-gpu {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 14px;
  margin-bottom: 2px;
}

.inst-gpu-count {
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 400;
}

.inst-specs {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: var(--text-secondary);
  margin-top: 4px;
}

.inst-selected-mark {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 10px;
  padding-top: 8px;
  border-top: 1px solid var(--border-color);
  font-size: 12px;
  font-weight: 600;
  color: var(--primary-color);
}

/* 资源网格 */
.resource-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

/* 镜像选择触发器 */
.image-select-trigger {
  cursor: pointer;
}

.image-readonly-input :deep(.el-input__wrapper) {
  cursor: pointer;
  background: var(--bg-primary);
  width: 600px;
}

/* 镜像选择弹窗 */
.image-selector-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 16px;
  background: var(--bg-secondary);
  padding: 4px;
  border-radius: var(--radius-lg);
  width: fit-content;
}

.image-tab {
  padding: 8px 20px;
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  transition: all 0.2s;
  user-select: none;
}

.image-tab:hover {
  color: var(--text-primary);
}

.image-tab.active {
  background: white;
  color: var(--primary-color);
  box-shadow: var(--shadow-sm);
}

.image-loading {
  padding: 20px;
}

.image-list {
  max-height: 400px;
  overflow-y: auto;
  padding-right: 4px;
}

.image-list-empty {
  text-align: center;
  padding: 40px 20px;
  color: var(--text-secondary);
}

.image-list-empty p {
  font-size: 15px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.image-list-empty span {
  font-size: 12px;
}

.image-list-item {
  padding: 14px 16px;
  border: 2px solid var(--border-color);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.2s;
  background: var(--bg-primary);
  margin-bottom: 10px;
}

.image-list-item:hover {
  border-color: var(--primary-color);
}

.image-list-item.selected {
  border-color: var(--primary-color);
  background: rgba(79, 70, 229, 0.05);
}

.image-list-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  font-family: 'JetBrains Mono', monospace;
  word-break: break-all;
  margin-bottom: 4px;
}

.image-list-desc {
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 8px;
  line-height: 1.4;
}

.image-list-meta {
  display: flex;
  align-items: center;
  gap: 8px;
}

.official-tag {
  background: #dcfce7;
  color: #166534;
  font-size: 11px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 4px;
}

.private-tag {
  background: #e0e7ff;
  color: #3730a3;
  font-size: 11px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 4px;
}

.star-count {
  display: flex;
  align-items: center;
  gap: 3px;
  font-size: 12px;
  color: var(--text-secondary);
}

.star-count svg {
  color: #f59e0b;
}

/* 环境变量 */
.env-vars-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  width: 100%;
}

.env-row {
  width: 100%;
  margin-bottom: 0;
}

.add-env-btn {
  grid-column: 1 / -1;
  justify-self: start;
  margin-top: 4px;
}

/* 存储配置 */
.storage-bindings {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
}

.storage-binding-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
}

.storage-path-trigger {
  flex: 1;
  cursor: pointer;
}

.storage-path-trigger :deep(.el-input__wrapper) {
  cursor: pointer;
  background: var(--bg-primary);
}

.add-storage-btn {
  margin-top: 8px;
}

/* 字段提示 */
.field-hint {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 6px;
}

.field-hint code {
  background: var(--bg-primary);
  padding: 1px 5px;
  border-radius: 3px;
  font-size: 11px;
  color: var(--primary-color);
  font-family: 'JetBrains Mono', monospace;
}

/* 底部操作 */
.form-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 24px 0 40px;
}

/* 确认弹窗 */
.confirm-body {
  max-height: 480px;
  overflow-y: auto;
}

.confirm-section {
  margin-bottom: 20px;
}

.confirm-section h4 {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0 0 10px 0;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-color);
}

.confirm-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  padding: 6px 0;
  font-size: 13px;
}

.confirm-label {
  color: var(--text-secondary);
  flex-shrink: 0;
  min-width: 80px;
}

.confirm-value {
  color: var(--text-primary);
  font-weight: 500;
  text-align: right;
  word-break: break-all;
}

.confirm-value.code {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  background: var(--bg-secondary);
  padding: 4px 8px;
  border-radius: var(--radius-sm);
}

.storage-binding-confirm {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  background: var(--bg-secondary);
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  word-break: break-all;
}

.env-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  text-align: right;
}

.env-tag {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  background: var(--bg-secondary);
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  word-break: break-all;
}

/* 响应式 */
@media (max-width: 768px) {
  .resource-grid {
    grid-template-columns: 1fr;
  }

  .image-grid {
    grid-template-columns: 1fr;
  }

  .provider-options {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
