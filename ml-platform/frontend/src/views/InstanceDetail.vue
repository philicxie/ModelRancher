<template>
  <div class="instance-detail-page">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="header-content">
        <el-button text @click="$router.push('/my-instances')">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
            <polyline points="15 18 9 12 15 6"/>
          </svg>
          返回
        </el-button>
        <h1 class="page-title">实例详情</h1>
        <p class="page-subtitle">{{ instance.id }}</p>
      </div>
      <div class="header-actions">
        <el-tag :type="getStatusTagType(instance.status)" size="large">
          {{ getStatusText(instance.status) }}
        </el-tag>
      </div>
    </div>

    <!-- 基本信息 -->
    <el-card class="info-card" v-loading="loading">
      <template #header>
        <div class="card-header">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18">
            <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
            <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
          </svg>
          <span>基本信息</span>
        </div>
      </template>
      <div class="info-grid">
        <div class="info-item">
          <span class="info-label">名称</span>
          <span class="info-value">{{ instance.name || '-' }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">GPU</span>
          <span class="info-value">{{ instance.gpu_name }} x{{ instance.num_gpus || 1 }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">提供商</span>
          <span class="info-value">{{ instance.provider }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">价格</span>
          <span class="info-value price">${{ formatPrice(instance.price_per_hour) }}/小时</span>
        </div>
        <div class="info-item">
          <span class="info-label">位置</span>
          <span class="info-value">{{ instance.location || '-' }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">创建时间</span>
          <span class="info-value">{{ formatDate(instance.created_at) }}</span>
        </div>
      </div>

      <!-- SSH 信息 -->
      <div v-if="instance.status === 'running' && instance.ssh_command" class="ssh-section">
        <div class="ssh-label">SSH 连接</div>
        <code class="ssh-cmd">{{ instance.ssh_command }}</code>
        <el-button type="primary" size="small" text @click="copySSHCommand">复制</el-button>
        <template v-if="instance.password">
          <div class="ssh-label" style="margin-left: 16px;">密码</div>
          <code class="ssh-cmd">{{ instance.password }}</code>
          <el-button type="primary" size="small" text @click="copyPassword">复制密码</el-button>
        </template>
      </div>
    </el-card>

    <!-- 监控指标 -->
    <div class="metrics-section">
      <div class="section-header">
        <h2 class="section-title">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
            <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
          </svg>
          实时监控
        </h2>
        <el-button size="small" text @click="refreshMetrics" :loading="metricsLoading">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
            <polyline points="23 4 23 10 17 10"/>
            <polyline points="1 20 1 14 7 14"/>
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
          </svg>
          刷新
        </el-button>
      </div>

      <div v-if="instance.status !== 'running'" class="metrics-not-available">
        <el-alert type="info" :closable="false" show-icon>
          <template #title>
            实例未运行时无法获取监控数据
          </template>
        </el-alert>
      </div>

      <div v-else v-loading="metricsLoading" class="metrics-grid">
        <!-- CPU 利用率 -->
        <el-card class="metric-card">
          <template #header>
            <div class="metric-header">
              <span class="metric-title">CPU 利用率</span>
              <span class="metric-value" :class="getUtilClass(cpuCurrent)">{{ cpuCurrent.toFixed(1) }}%</span>
            </div>
          </template>
          <div ref="cpuChart" class="chart-container"></div>
        </el-card>

        <!-- 内存利用率 -->
        <el-card class="metric-card">
          <template #header>
            <div class="metric-header">
              <span class="metric-title">内存利用率</span>
              <span class="metric-value" :class="getUtilClass(memCurrent)">{{ memCurrent.toFixed(1) }}%</span>
            </div>
          </template>
          <div ref="memChart" class="chart-container"></div>
        </el-card>

        <!-- GPU 利用率 -->
        <el-card class="metric-card">
          <template #header>
            <div class="metric-header">
              <span class="metric-title">GPU 利用率</span>
              <span class="metric-value" :class="getUtilClass(gpuCurrent)">{{ gpuCurrent.toFixed(1) }}%</span>
            </div>
          </template>
          <div ref="gpuChart" class="chart-container"></div>
        </el-card>

        <!-- GPU 显存利用率 -->
        <el-card class="metric-card">
          <template #header>
            <div class="metric-header">
              <span class="metric-title">GPU 显存</span>
              <span class="metric-value" :class="getUtilClass(gpuMemCurrent)">{{ gpuMemCurrent.toFixed(1) }}%</span>
            </div>
          </template>
          <div ref="gpuMemChart" class="chart-container"></div>
        </el-card>

        <!-- 磁盘利用率 -->
        <el-card class="metric-card">
          <template #header>
            <div class="metric-header">
              <span class="metric-title">磁盘利用率</span>
              <span class="metric-value" :class="getUtilClass(diskCurrent)">{{ diskCurrent.toFixed(1) }}%</span>
            </div>
          </template>
          <div ref="diskChart" class="chart-container"></div>
        </el-card>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick, computed } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts'

const route = useRoute()
const instanceId = route.params.id

const loading = ref(false)
const metricsLoading = ref(false)
const instance = ref({})
const metrics = ref(null)

// 图表实例
const cpuChart = ref(null)
const memChart = ref(null)
const gpuChart = ref(null)
const gpuMemChart = ref(null)
const diskChart = ref(null)
const charts = new Map()

// 当前值
const cpuCurrent = computed(() => {
  const points = metrics.value?.cpu_utilization || []
  return points.length ? points[points.length - 1].value : 0
})
const memCurrent = computed(() => {
  const points = metrics.value?.mem_utilization || []
  return points.length ? points[points.length - 1].value : 0
})
const gpuCurrent = computed(() => {
  const points = metrics.value?.gpu_utilization_avg || []
  return points.length ? points[points.length - 1].value : 0
})
const gpuMemCurrent = computed(() => {
  const points = metrics.value?.gpu_mem_utilization_avg || []
  return points.length ? points[points.length - 1].value : 0
})
const diskCurrent = computed(() => {
  const points = metrics.value?.root_disk_utilization || []
  return points.length ? points[points.length - 1].value : 0
})

let pollInterval = null

async function fetchInstance() {
  loading.value = true
  try {
    const res = await axios.get(`/api/v1/offers/${instanceId}`)
    instance.value = res.data
  } catch (err) {
    ElMessage.error('获取实例信息失败: ' + (err.response?.data?.error || err.message))
  } finally {
    loading.value = false
  }
}

async function fetchMetrics() {
  if (instance.value.status !== 'running') return
  metricsLoading.value = true
  try {
    const res = await axios.get(`/api/v1/offers/${instanceId}/metrics`)
    metrics.value = res.data
  } catch (err) {
    console.warn('获取监控数据失败:', err)
    return
  } finally {
    metricsLoading.value = false
  }
  // DOM 更新后再渲染图表，避免 v-loading 遮罩影响尺寸
  await nextTick()
  renderCharts()
}

function refreshMetrics() {
  fetchMetrics()
}

function renderCharts() {
  if (!metrics.value) return

  renderLineChart(cpuChart.value, 'CPU 利用率', metrics.value.cpu_utilization, '#2563eb')
  renderLineChart(memChart.value, '内存利用率', metrics.value.mem_utilization, '#10b981')
  renderLineChart(gpuChart.value, 'GPU 利用率', metrics.value.gpu_utilization_avg, '#f59e0b')
  renderLineChart(gpuMemChart.value, 'GPU 显存', metrics.value.gpu_mem_utilization_avg, '#ef4444')
  renderLineChart(diskChart.value, '磁盘利用率', metrics.value.root_disk_utilization, '#64748b')
}

function renderLineChart(el, title, data, color) {
  if (!el) return

  // 复用已有图表实例，避免频繁 dispose/init
  let chart = charts.get(el)
  if (!chart) {
    chart = echarts.init(el)
    charts.set(el, chart)
  }

  const hasData = data && data.length > 0
  const xData = hasData ? data.map(d => {
    const date = new Date(d.timestamp * 1000)
    return `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}:${date.getSeconds().toString().padStart(2, '0')}`
  }) : []
  const yData = hasData ? data.map(d => Number(d.value.toFixed(1))) : []

  if (!hasData) {
    chart.setOption({
      title: { text: '暂无数据', left: 'center', top: 'center', textStyle: { color: '#94a3b8', fontSize: 14 } },
      xAxis: { show: false },
      yAxis: { show: false },
      series: []
    }, true) // true = notMerge，完全替换
    chart.resize()
    return
  }

  chart.setOption({
    grid: { top: 10, right: 10, bottom: 20, left: 40 },
    title: { show: false },
    tooltip: {
      trigger: 'axis',
      formatter: (params) => {
        const p = params[0]
        return `${p.name}<br/>${p.seriesName}: ${p.value}%`
      }
    },
    xAxis: {
      type: 'category',
      data: xData,
      axisLine: { lineStyle: { color: '#e2e8f0' } },
      axisLabel: { color: '#94a3b8', fontSize: 10 },
      axisTick: { show: false }
    },
    yAxis: {
      type: 'value',
      min: 0,
      max: 100,
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: { lineStyle: { color: '#f1f5f9' } },
      axisLabel: { color: '#94a3b8', fontSize: 10, formatter: '{value}%' }
    },
    series: [{
      name: title,
      type: 'line',
      data: yData,
      smooth: true,
      symbol: 'none',
      lineStyle: { color: color, width: 2 },
      areaStyle: {
        color: {
          type: 'linear',
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: color + '40' },
            { offset: 1, color: color + '05' }
          ]
        }
      }
    }]
  }, true) // true = notMerge，完全替换旧配置
  chart.resize()
}

function getUtilClass(val) {
  if (val >= 90) return 'danger'
  if (val >= 70) return 'warning'
  return 'normal'
}

function getStatusTagType(status) {
  const map = {
    creating: 'warning',
    running: 'success',
    stopped: 'info',
    destroying: 'warning',
    destroyed: 'info',
    failed: 'danger'
  }
  return map[status] || 'info'
}

function getStatusText(status) {
  const map = {
    creating: '创建中',
    running: '运行中',
    stopped: '已停止',
    destroying: '销毁中',
    destroyed: '已销毁',
    failed: '失败'
  }
  return map[status] || status
}

function formatPrice(price) {
  return price ? price.toFixed(2) : '0.00'
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN')
}

function copySSHCommand() {
  navigator.clipboard.writeText(instance.value.ssh_command).then(() => {
    ElMessage.success('已复制到剪贴板')
  })
}

function copyPassword() {
  navigator.clipboard.writeText(instance.value.password).then(() => {
    ElMessage.success('密码已复制到剪贴板')
  })
}

function handleResize() {
  charts.forEach(chart => chart && chart.resize())
}

onMounted(async () => {
  await fetchInstance()
  await fetchMetrics()
  // 每 15 秒轮询一次 metrics
  pollInterval = setInterval(fetchMetrics, 15000)
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
  window.removeEventListener('resize', handleResize)
  Object.values(charts).forEach(chart => chart && chart.dispose())
})
</script>

<style scoped>
.instance-detail-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.03em;
  margin: 0;
}

.page-subtitle {
  font-size: 13px;
  color: #909399;
  margin: 4px 0 0 0;
  font-family: monospace;
}

.info-card {
  margin-bottom: 24px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 700;
  color: var(--text-primary);
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
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
}

.info-value {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.info-value.price {
  color: #d97706;
  font-weight: 700;
}

.ssh-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.ssh-label {
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 500;
}

.ssh-cmd {
  background: #f1f5f9;
  padding: 6px 12px;
  border-radius: var(--radius-md);
  font-family: monospace;
  font-size: 13px;
  color: #475569;
}

.metrics-section {
  margin-top: 24px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.metrics-not-available {
  margin-bottom: 24px;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 16px;
}

.metric-card {
  min-height: 240px;
}

.metric-card :deep(.el-card__header) {
  padding: 12px 16px;
}

.metric-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.metric-title {
  font-size: 14px;
  font-weight: 500;
  color: #64748b;
}

.metric-value {
  font-size: 18px;
  font-weight: 700;
}

.metric-value.normal {
  color: #67c23a;
}

.metric-value.warning {
  color: #e6a23c;
}

.metric-value.danger {
  color: #f56c6c;
}

.chart-container {
  width: 100%;
  height: 160px;
}
</style>
