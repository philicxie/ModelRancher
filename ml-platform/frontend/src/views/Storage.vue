<template>
  <div class="storage">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">对象存储</h1>
        <p class="page-subtitle">管理腾讯云 COS 存储桶中的文件</p>
      </div>
    </div>

    <div class="storage-layout">
      <!-- 左侧：桶管理区 -->
      <aside class="bucket-panel">
        <div class="panel-header">
          <h3 class="panel-title">存储桶</h3>
          <el-button size="small" @click="showCreateBucket = true">
            <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="12" y1="5" x2="12" y2="19"/>
              <line x1="5" y1="12" x2="19" y2="12"/>
            </svg>
            新建
          </el-button>
        </div>

        <div v-loading="bucketsLoading" class="bucket-list">
          <!-- 我的桶 -->
          <div v-if="myBuckets.length > 0" class="bucket-group">
            <div class="group-label">我的桶</div>
            <div
              v-for="bucket in myBuckets"
              :key="bucket.id"
              class="bucket-item"
              :class="{ active: currentBucket?.id === bucket.id }"
              @click="selectBucket(bucket)"
            >
              <svg class="bucket-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
              </svg>
              <div class="bucket-info">
                <span class="bucket-name">{{ bucket.name }}</span>
                <span class="bucket-tag" :class="bucket.is_public ? 'public' : 'private'">
                  {{ bucket.is_public ? '公共' : '私有' }}
                </span>
              </div>
              <el-button
                v-if="bucket.user_id === currentUserId"
                link
                type="danger"
                size="small"
                class="bucket-delete"
                @click.stop="deleteBucket(bucket)"
              >
                删除
              </el-button>
            </div>
          </div>

          <!-- 公共桶 -->
          <div v-if="publicBuckets.length > 0" class="bucket-group">
            <div class="group-label">公共桶</div>
            <div
              v-for="bucket in publicBuckets"
              :key="bucket.id"
              class="bucket-item"
              :class="{ active: currentBucket?.id === bucket.id }"
              @click="selectBucket(bucket)"
            >
              <svg class="bucket-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
              </svg>
              <div class="bucket-info">
                <span class="bucket-name">{{ bucket.name }}</span>
                <span class="bucket-tag public">公共</span>
              </div>
            </div>
          </div>

          <div v-if="!bucketsLoading && myBuckets.length === 0 && publicBuckets.length === 0" class="bucket-empty">
            暂无存储桶
          </div>
        </div>
      </aside>

      <!-- 右侧：文件列表区 -->
      <main class="file-panel">
        <div v-if="!currentBucket" class="no-bucket-tip">
          <svg class="tip-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
          <p>请从左侧选择一个存储桶</p>
        </div>

        <template v-else>
          <!-- 工具栏 -->
          <div class="toolbar">
            <!-- 面包屑 -->
            <div class="breadcrumb">
              <span class="breadcrumb-bucket" @click="prefix = ''">{{ currentBucket.name }}</span>
              <template v-for="(part, i) in breadcrumbParts" :key="i">
                <span class="breadcrumb-separator">/</span>
                <span
                  class="breadcrumb-item"
                  :class="{ active: i === breadcrumbParts.length - 1 }"
                  @click="navigateTo(breadcrumbPath(i))"
                >{{ part }}</span>
              </template>
            </div>

            <div class="toolbar-actions">
              <el-button size="small" @click="showCreateFolder = true">
                <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                  <line x1="12" y1="11" x2="12" y2="17"/>
                  <line x1="9" y1="14" x2="15" y2="14"/>
                </svg>
                新建文件夹
              </el-button>
              <el-upload
                :action="`/api/v1/storage/upload`"
                :data="{ bucket: currentBucket.name, prefix }"
                :show-file-list="false"
                :on-success="handleUploadSuccess"
                :on-error="handleUploadError"
                :before-upload="beforeUpload"
              >
                <el-button type="primary" size="small">
                  <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                    <polyline points="17 8 12 3 7 8"/>
                    <line x1="12" y1="3" x2="12" y2="15"/>
                  </svg>
                  上传文件
                </el-button>
              </el-upload>
            </div>
          </div>

          <!-- 文件列表 -->
          <div class="file-list-card">
            <el-table
              v-loading="loading"
              :data="displayFiles"
              style="width: 100%"
              class="files-table"
            >
              <el-table-column label="名称" min-width="280">
                <template #default="{ row }">
                  <div class="file-name-cell" @click="row.isDir ? navigateTo(row.key) : null">
                    <svg
                      class="file-icon"
                      :class="{ folder: row.isDir }"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                    >
                      <template v-if="row.isDir">
                        <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                      </template>
                      <template v-else>
                        <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
                        <polyline points="14 2 14 8 20 8"/>
                      </template>
                    </svg>
                    <span class="file-name" :class="{ clickable: row.isDir }">
                      {{ row.displayName }}
                    </span>
                  </div>
                </template>
              </el-table-column>

              <el-table-column label="大小" width="120">
                <template #default="{ row }">
                  <span class="file-size">{{ row.isDir ? '-' : formatSize(row.size) }}</span>
                </template>
              </el-table-column>

              <el-table-column label="修改时间" width="180">
                <template #default="{ row }">
                  <span class="file-time">{{ formatTime(row.lastModified) }}</span>
                </template>
              </el-table-column>

              <el-table-column label="操作" width="140" align="right">
                <template #default="{ row }">
                  <div class="file-actions">
                    <el-button
                      v-if="!row.isDir"
                      link
                      type="primary"
                      size="small"
                      @click="downloadFile(row)"
                    >下载</el-button>
                    <el-button
                      link
                      type="danger"
                      size="small"
                      @click="deleteFile(row)"
                    >删除</el-button>
                  </div>
                </template>
              </el-table-column>
            </el-table>

            <div v-if="!loading && displayFiles.length === 0" class="empty-state">
              <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
              </svg>
              <h3>暂无文件</h3>
              <p>当前目录为空，点击上方按钮上传文件</p>
            </div>
          </div>
        </template>
      </main>
    </div>

    <!-- 新建桶弹窗 -->
    <el-dialog v-model="showCreateBucket" title="新建存储桶" width="400px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="桶名称">
          <el-input v-model="newBucketName" placeholder="请输入桶名称（英文/数字）" />
        </el-form-item>
        <el-form-item label="访问权限">
          <el-radio-group v-model="newBucketIsPublic">
            <el-radio-button :label="false">私有</el-radio-button>
            <el-radio-button :label="true">公共</el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateBucket = false">取消</el-button>
        <el-button type="primary" @click="createBucket">创建</el-button>
      </template>
    </el-dialog>

    <!-- 新建文件夹弹窗 -->
    <el-dialog v-model="showCreateFolder" title="新建文件夹" width="400px" :close-on-click-modal="false">
      <el-input v-model="newFolderName" placeholder="请输入文件夹名称" @keyup.enter="createFolder" />
      <template #footer>
        <el-button @click="showCreateFolder = false">取消</el-button>
        <el-button type="primary" @click="createFolder">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const currentUserId = 'default'
const bucketsLoading = ref(false)
const loading = ref(false)
const buckets = ref([])
const currentBucket = ref(null)
const files = ref([])
const prefix = ref('')

const showCreateBucket = ref(false)
const newBucketName = ref('')
const newBucketIsPublic = ref(false)

const showCreateFolder = ref(false)
const newFolderName = ref('')

const myBuckets = computed(() => buckets.value.filter(b => b.user_id === currentUserId))
const publicBuckets = computed(() => buckets.value.filter(b => b.user_id !== currentUserId && b.is_public))

const breadcrumbParts = computed(() => {
  if (!prefix.value) return []
  let p = prefix.value
  if (p.endsWith('/')) p = p.slice(0, -1)
  return p.split('/').filter(Boolean)
})

const breadcrumbPath = (index) => {
  const parts = breadcrumbParts.value
  return parts.slice(0, index + 1).join('/') + '/'
}

// 把COS扁平列表处理成当前目录下的文件/文件夹视图
// 后端返回完整COS路径如 buckets/name/xxx，前端用 bucketBase 去掉前缀得到相对于桶的路径
const displayFiles = computed(() => {
  if (!currentBucket.value) return []
  const bucketBase = `buckets/${currentBucket.value.name}/`
  const currentPrefix = prefix.value
  const seen = new Set()
  const result = []

  for (const obj of files.value) {
    // 去掉桶前缀，得到相对于桶的路径
    let relativeToBucket = obj.key
    if (relativeToBucket.startsWith(bucketBase)) {
      relativeToBucket = relativeToBucket.slice(bucketBase.length)
    }

    // 只显示当前 prefix 下的对象
    if (!relativeToBucket.startsWith(currentPrefix)) continue

    const relativePath = relativeToBucket.slice(currentPrefix.length)
    if (!relativePath) continue

    const slashIndex = relativePath.indexOf('/')
    if (slashIndex === -1) {
      result.push({
        key: relativeToBucket, // 相对于桶的路径，用于下载/删除
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
          key: dirKey, // 相对于桶的路径，用于导航
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
}

const formatSize = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatTime = (t) => {
  if (!t) return '-'
  const d = new Date(t)
  return `${d.getFullYear()}-${(d.getMonth()+1).toString().padStart(2,'0')}-${d.getDate().toString().padStart(2,'0')} ${d.getHours().toString().padStart(2,'0')}:${d.getMinutes().toString().padStart(2,'0')}`
}

const downloadFile = async (row) => {
  try {
    const res = await fetch(
      `/api/v1/storage/download-url?bucket=${encodeURIComponent(currentBucket.value.name)}&key=${encodeURIComponent(row.key)}`
    )
    if (!res.ok) throw new Error()
    const data = await res.json()
    if (data.url) window.open(data.url, '_blank')
  } catch {
    ElMessage.error('获取下载链接失败')
  }
}

const deleteFile = async (row) => {
  try {
    await ElMessageBox.confirm(`确定要删除 "${row.displayName}" 吗？`, '确认删除', {
      confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning'
    })
  } catch { return }

  try {
    const res = await fetch(
      `/api/v1/storage/files?bucket=${encodeURIComponent(currentBucket.value.name)}&key=${encodeURIComponent(row.key)}`,
      { method: 'DELETE' }
    )
    if (!res.ok) throw new Error()
    ElMessage.success('删除成功')
    fetchFiles()
  } catch {
    ElMessage.error('删除失败')
  }
}

const createBucket = async () => {
  const name = newBucketName.value.trim()
  if (!name) {
    ElMessage.warning('请输入桶名称')
    return
  }
  if (!/^[a-zA-Z0-9_-]+$/.test(name)) {
    ElMessage.warning('桶名称只能包含字母、数字、下划线和横线')
    return
  }
  try {
    const res = await fetch(`/api/v1/storage/buckets?user_id=${currentUserId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, is_public: newBucketIsPublic.value }),
    })
    if (!res.ok) throw new Error()
    ElMessage.success('桶创建成功')
    showCreateBucket.value = false
    newBucketName.value = ''
    newBucketIsPublic.value = false
    fetchBuckets()
  } catch {
    ElMessage.error('创建桶失败，名称可能已存在')
  }
}

const deleteBucket = async (bucket) => {
  try {
    await ElMessageBox.confirm(`确定要删除桶 "${bucket.name}" 吗？桶内所有文件将被清空。`, '确认删除', {
      confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning'
    })
  } catch { return }

  try {
    const res = await fetch(`/api/v1/storage/buckets/${bucket.id}?user_id=${currentUserId}`, {
      method: 'DELETE',
    })
    if (!res.ok) throw new Error()
    ElMessage.success('桶删除成功')
    if (currentBucket.value?.id === bucket.id) {
      currentBucket.value = null
      files.value = []
    }
    fetchBuckets()
  } catch {
    ElMessage.error('删除桶失败')
  }
}

const createFolder = async () => {
  const name = newFolderName.value.trim()
  if (!name) {
    ElMessage.warning('请输入文件夹名称')
    return
  }
  if (name.includes('/') || name.includes('\\')) {
    ElMessage.warning('文件夹名称不能包含斜杠')
    return
  }
  try {
    const res = await fetch('/api/v1/storage/folders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ bucket: currentBucket.value.name, prefix: prefix.value, name }),
    })
    if (!res.ok) throw new Error()
    ElMessage.success('文件夹创建成功')
    showCreateFolder.value = false
    newFolderName.value = ''
    fetchFiles()
  } catch {
    ElMessage.error('创建文件夹失败')
  }
}

const beforeUpload = (file) => {
  const maxSize = 100 * 1024 * 1024
  if (file.size > maxSize) {
    ElMessage.error('文件大小超过 100MB 限制')
    return false
  }
  return true
}

const handleUploadSuccess = () => {
  ElMessage.success('上传成功')
  fetchFiles()
}

const handleUploadError = () => {
  ElMessage.error('上传失败')
}

onMounted(() => {
  fetchBuckets()
})
</script>

<style scoped>
.storage {
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
  margin-bottom: 8px;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
}

/* 左右布局 */
.storage-layout {
  display: flex;
  gap: 20px;
  height: calc(100vh - 180px);
}

/* 左侧桶面板 */
.bucket-panel {
  width: 280px;
  flex-shrink: 0;
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 16px 12px;
  border-bottom: 1px solid var(--border-color);
}

.panel-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.bucket-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.bucket-group {
  margin-bottom: 12px;
}

.group-label {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-secondary);
  padding: 8px 8px 4px;
}

.bucket-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 8px;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
}

.bucket-item:hover {
  background: var(--bg-primary);
}

.bucket-item.active {
  background: linear-gradient(135deg, #eff6ff, #dbeafe);
}

.bucket-item.active .bucket-name {
  color: #2563eb;
  font-weight: 600;
}

.bucket-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  color: #f59e0b;
}

.bucket-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.bucket-name {
  font-size: 13px;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.bucket-tag {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: var(--radius-full);
  font-weight: 600;
  align-self: flex-start;
}

.bucket-tag.private {
  background: #f1f5f9;
  color: #475569;
}

.bucket-tag.public {
  background: #dbeafe;
  color: #2563eb;
}

.bucket-delete {
  opacity: 0;
  transition: opacity 0.2s;
}

.bucket-item:hover .bucket-delete {
  opacity: 1;
}

.bucket-empty {
  text-align: center;
  padding: 32px 16px;
  color: var(--text-secondary);
  font-size: 13px;
}

/* 右侧文件面板 */
.file-panel {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.no-bucket-tip {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  gap: 12px;
}

.tip-icon {
  width: 64px;
  height: 64px;
  color: var(--border-color);
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 14px;
}

.breadcrumb-bucket {
  color: var(--primary-color);
  cursor: pointer;
  font-weight: 600;
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  transition: background 0.2s;
}

.breadcrumb-bucket:hover {
  background: var(--primary-light);
}

.breadcrumb-item {
  color: var(--primary-color);
  cursor: pointer;
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  transition: background 0.2s;
}

.breadcrumb-item:hover {
  background: var(--primary-light);
}

.breadcrumb-item.active {
  color: var(--text-primary);
  font-weight: 600;
  cursor: default;
}

.breadcrumb-item.active:hover {
  background: transparent;
}

.breadcrumb-separator {
  color: var(--text-secondary);
  opacity: 0.5;
}

/* 覆盖 App.vue 全局面包屑样式，避免 ::before 伪元素与手动 / 重复 */
.breadcrumb .breadcrumb-item::before {
  display: none;
}

.toolbar-actions {
  display: flex;
  gap: 10px;
}

.btn-icon {
  width: 14px;
  height: 14px;
  margin-right: 4px;
  vertical-align: middle;
}

.file-list-card {
  flex: 1;
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  padding: 20px;
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--border-color);
  overflow: auto;
}

.files-table :deep(.el-table__header th.el-table__cell) {
  background: var(--bg-primary);
  color: var(--text-secondary);
  font-weight: 700;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  padding: 10px 14px;
}

.files-table :deep(.el-table__row td.el-table__cell) {
  padding: 10px 14px;
}

.file-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.file-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  color: var(--text-secondary);
}

.file-icon.folder {
  color: #f59e0b;
}

.file-name {
  font-weight: 500;
  color: var(--text-primary);
  font-size: 14px;
}

.file-name.clickable {
  cursor: pointer;
  color: var(--primary-color);
}

.file-name.clickable:hover {
  text-decoration: underline;
}

.file-size {
  font-size: 13px;
  color: var(--text-secondary);
  font-family: 'JetBrains Mono', monospace;
}

.file-time {
  font-size: 13px;
  color: var(--text-secondary);
}

.file-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.empty-state {
  text-align: center;
  padding: 48px 20px;
  color: var(--text-secondary);
}

.empty-icon {
  width: 56px;
  height: 56px;
  margin-bottom: 12px;
  color: var(--border-color);
}

.empty-state h3 {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
  color: var(--text-primary);
}

.empty-state p {
  font-size: 13px;
}

/* 响应式 */
@media (max-width: 768px) {
  .storage-layout {
    flex-direction: column;
    height: auto;
  }

  .bucket-panel {
    width: 100%;
    max-height: 300px;
  }
}
</style>
