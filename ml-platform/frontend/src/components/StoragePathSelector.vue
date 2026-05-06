<template>
  <el-dialog
    v-model="visible"
    title="选择存储路径"
    width="720px"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <div class="selector-layout">
      <!-- 左侧桶列表 -->
      <aside class="bucket-panel" v-loading="bucketsLoading">
        <div class="panel-title">存储桶</div>
        <div class="bucket-list">
          <div
            v-for="bucket in buckets"
            :key="bucket.id"
            class="bucket-item"
            :class="{ active: currentBucket?.id === bucket.id }"
            @click="selectBucket(bucket)"
          >
            <svg class="bucket-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
            </svg>
            <span class="bucket-name">{{ bucket.name }}</span>
          </div>
        </div>
      </aside>

      <!-- 右侧文件浏览 -->
      <div class="file-panel" v-loading="loading">
        <!-- 工具栏 -->
        <div v-if="currentBucket" class="file-toolbar">
          <div class="breadcrumb">
            <el-button
              v-if="prefix"
              link
              size="small"
              type="primary"
              @click="goUp"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                <polyline points="15 18 9 12 15 6"/>
              </svg>
              返回上级
            </el-button>
            <span v-else class="root-label">{{ currentBucket.name }}</span>
          </div>
          <el-button
            size="small"
            type="primary"
            plain
            @click="openCreateFolder"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14" style="margin-right: 4px;">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
              <line x1="12" y1="11" x2="12" y2="17"/>
              <line x1="9" y1="14" x2="15" y2="14"/>
            </svg>
            新建文件夹
          </el-button>
        </div>

        <!-- 文件列表 -->
        <el-table
          v-if="currentBucket"
          :data="displayFiles"
          size="small"
          height="320"
          @row-click="onRowClick"
          highlight-current-row
        >
          <el-table-column width="40">
            <template #default="{ row }">
              <svg v-if="row.isDir" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16" style="color: #f59e0b;">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
              </svg>
              <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16" style="color: var(--text-secondary);">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                <polyline points="14 2 14 8 20 8"/>
              </svg>
            </template>
          </el-table-column>
          <el-table-column prop="displayName" label="名称" show-overflow-tooltip />
          <el-table-column width="80" align="right">
            <template #default="{ row }">
              <el-button v-if="row.isDir" link type="primary" size="small" @click.stop="navigateTo(row.key)">
                进入
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <div v-else class="empty-hint">请先在左侧选择一个存储桶</div>
      </div>
    </div>

    <!-- 当前选中路径 -->
    <div class="selected-path" v-if="currentBucket">
      <span class="path-label">当前路径：</span>
      <code class="path-value">{{ fullPath || currentBucket.name + '/' }}</code>
    </div>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="confirm" :disabled="!currentBucket">
        确认选择
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const props = defineProps({
  modelValue: Boolean,
})
const emit = defineEmits(['update:modelValue', 'select'])

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const currentUserId = 'default'
const bucketsLoading = ref(false)
const loading = ref(false)
const buckets = ref([])
const currentBucket = ref(null)
const files = ref([])
const prefix = ref('')

const fullPath = computed(() => {
  if (!currentBucket.value) return ''
  return currentBucket.value.name + '/' + prefix.value
})

const displayFiles = computed(() => {
  if (!currentBucket.value) return []
  const bucketBase = `buckets/${currentBucket.value.name}/`
  const currentPrefix = prefix.value
  const seen = new Set()
  const result = []

  for (const obj of files.value) {
    let relativeToBucket = obj.key
    if (relativeToBucket.startsWith(bucketBase)) {
      relativeToBucket = relativeToBucket.slice(bucketBase.length)
    }
    if (!relativeToBucket.startsWith(currentPrefix)) continue
    const relativePath = relativeToBucket.slice(currentPrefix.length)
    if (!relativePath) continue

    const slashIndex = relativePath.indexOf('/')
    if (slashIndex === -1) {
      result.push({
        key: relativeToBucket,
        displayName: relativePath,
        size: obj.size,
        lastModified: obj.lastModified,
        isDir: false,
      })
    } else {
      const dirName = relativePath.slice(0, slashIndex + 1)
      const dirKey = currentPrefix + dirName
      if (!seen.has(dirKey)) {
        seen.add(dirKey)
        result.push({
          key: dirKey,
          displayName: dirName.slice(0, -1),
          size: 0,
          lastModified: '',
          isDir: true,
        })
      }
    }
  }

  result.sort((a, b) => {
    if (a.isDir && !b.isDir) return -1
    if (!a.isDir && b.isDir) return 1
    return a.displayName.localeCompare(b.displayName)
  })

  return result
})

const fetchBuckets = async () => {
  bucketsLoading.value = true
  try {
    const res = await fetch(`/api/v1/storage/buckets?user_id=${currentUserId}`)
    if (!res.ok) throw new Error()
    const data = await res.json()
    buckets.value = data.buckets || []
  } catch {
    ElMessage.error('获取桶列表失败')
  } finally {
    bucketsLoading.value = false
  }
}

const selectBucket = (bucket) => {
  currentBucket.value = bucket
  prefix.value = ''
  fetchFiles()
}

const fetchFiles = async () => {
  if (!currentBucket.value) return
  loading.value = true
  try {
    const res = await fetch(
      `/api/v1/storage/files?bucket=${encodeURIComponent(currentBucket.value.name)}&prefix=${encodeURIComponent(prefix.value)}`
    )
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    files.value = data || []
  } catch (err) {
    console.error('fetchFiles failed:', err)
    ElMessage.error('获取文件列表失败')
  } finally {
    loading.value = false
  }
}

const navigateTo = (newPrefix) => {
  prefix.value = newPrefix
  fetchFiles()
}

const goUp = () => {
  if (!prefix.value) return
  const idx = prefix.value.slice(0, -1).lastIndexOf('/')
  if (idx === -1) {
    prefix.value = ''
  } else {
    prefix.value = prefix.value.slice(0, idx + 1)
  }
  fetchFiles()
}

const openCreateFolder = () => {
  ElMessageBox.prompt('请输入文件夹名称', '新建文件夹', {
    confirmButtonText: '创建',
    cancelButtonText: '取消',
    inputPattern: /^[^/\\]+$/,
    inputErrorMessage: '文件夹名称不能包含斜杠',
  }).then(({ value }) => {
    const name = value.trim()
    if (!name) {
      ElMessage.warning('文件夹名称不能为空')
      return
    }
    createFolder(name)
  }).catch(() => {})
}

const createFolder = async (name) => {
  loading.value = true
  try {
    const res = await fetch('/api/v1/storage/folders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        bucket: currentBucket.value.name,
        prefix: prefix.value,
        name: name,
      }),
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `HTTP ${res.status}`)
    }
    ElMessage.success('文件夹创建成功')
    fetchFiles()
  } catch (err) {
    console.error('createFolder failed:', err)
    ElMessage.error('创建文件夹失败: ' + err.message)
  } finally {
    loading.value = false
  }
}

const onRowClick = (row) => {
  if (row.isDir) {
    navigateTo(row.key)
  }
}

const confirm = () => {
  if (!currentBucket.value) return
  emit('select', fullPath.value)
  visible.value = false
}

watch(visible, (v) => {
  if (v) {
    prefix.value = ''
    currentBucket.value = null
    fetchBuckets()
  }
})
</script>

<style scoped>
.selector-layout {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 16px;
  min-height: 360px;
}

.bucket-panel,
.file-panel {
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.panel-title {
  padding: 12px 16px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
}

.bucket-list {
  padding: 8px;
  max-height: 320px;
  overflow-y: auto;
}

.bucket-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s;
  font-size: 13px;
}

.bucket-item:hover {
  background: var(--bg-secondary);
}

.bucket-item.active {
  background: rgba(79, 70, 229, 0.08);
  color: var(--primary-color);
  font-weight: 500;
}

.bucket-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.bucket-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-panel {
  display: flex;
  flex-direction: column;
}

.file-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.breadcrumb {
  display: flex;
  align-items: center;
  font-size: 13px;
}

.root-label {
  color: var(--text-secondary);
  font-weight: 500;
}

.empty-hint {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  font-size: 13px;
}

.selected-path {
  margin-top: 16px;
  padding: 10px 12px;
  background: var(--bg-secondary);
  border-radius: var(--radius-sm);
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.path-label {
  color: var(--text-secondary);
  flex-shrink: 0;
}

.path-value {
  color: var(--text-primary);
  font-family: 'JetBrains Mono', monospace;
  word-break: break-all;
}

</style>
