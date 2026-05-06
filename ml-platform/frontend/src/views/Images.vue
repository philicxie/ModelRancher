<template>
  <div class="images-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">镜像管理</h1>
        <p class="page-subtitle">搜索公开镜像，管理私有镜像和收藏</p>
      </div>
    </div>

    <!-- Tab 切换 -->
    <div class="tab-bar">
      <div
        v-for="tab in tabs"
        :key="tab.key"
        class="tab-item"
        :class="{ active: activeTab === tab.key }"
        @click="activeTab = tab.key"
      >
        <span class="tab-label">{{ tab.label }}</span>
        <span v-if="tab.key === 'favorites' && favoriteCount > 0" class="tab-badge">{{ favoriteCount }}</span>
      </div>
    </div>

    <!-- 公开镜像 -->
    <template v-if="activeTab === 'public'">
      <div class="search-bar">
        <el-input
          v-model="searchQuery"
          placeholder="搜索 Docker Hub 镜像，如 pytorch、tensorflow..."
          size="large"
          clearable
          @keyup.enter="doSearch"
        >
          <template #prefix>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18">
              <circle cx="11" cy="11" r="8"/>
              <path d="m21 21-4.3-4.3"/>
            </svg>
          </template>
          <template #append>
            <el-button type="primary" @click="doSearch">搜索</el-button>
          </template>
        </el-input>
      </div>

      <div v-if="searchLoading" class="loading-state">
        <el-skeleton :rows="4" animated />
      </div>

      <div v-else-if="searchResults.length > 0" class="images-grid">
        <div
          v-for="img in searchResults"
          :key="img.name"
          class="image-card"
        >
          <div class="image-header-row">
            <div class="image-name-block">
              <h3 class="image-name" :title="img.name">{{ img.name }}</h3>
              <el-tag v-if="img.is_official" size="small" type="success" effect="plain">官方</el-tag>
            </div>
            <el-button
              circle
              :type="img.is_favorited ? 'warning' : 'default'"
              @click.stop="toggleFavorite(img)"
            >
              <svg v-if="img.is_favorited" viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
                <path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"/>
              </svg>
              <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/>
              </svg>
            </el-button>
          </div>

          <p class="image-desc">{{ img.description || '暂无描述' }}</p>

          <div class="image-footer-row">
            <span class="image-stars">
              <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
                <path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z"/>
              </svg>
              {{ formatStars(img.star_count) }}
            </span>
          </div>
        </div>
      </div>

      <div v-else-if="hasSearched && !searchLoading" class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="11" cy="11" r="8"/>
          <path d="m21 21-4.3-4.3"/>
        </svg>
        <h3>未找到相关镜像</h3>
        <p>尝试更换关键词搜索</p>
      </div>

      <div v-else class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="11" cy="11" r="8"/>
          <path d="m21 21-4.3-4.3"/>
        </svg>
        <h3>搜索 Docker Hub 公开镜像</h3>
        <p>输入关键词，发现海量训练环境镜像</p>
      </div>
    </template>

    <!-- 私有镜像 -->
    <template v-if="activeTab === 'private'">
      <div v-if="privateLoading" class="loading-state">
        <el-skeleton :rows="4" animated />
      </div>

      <div v-else-if="privateImages.length > 0" class="images-grid">
        <div
          v-for="img in privateImages"
          :key="img.name"
          class="image-card private-card"
        >
          <div class="image-header-row">
            <div class="image-name-block">
              <h3 class="image-name" :title="img.name">{{ shortName(img.name) }}</h3>
            </div>
          </div>
          <p class="image-desc">{{ img.description || '暂无描述' }}</p>
          <div class="image-footer-row">
            <span class="image-meta">更新于 {{ formatTime(img.updated_at) }}</span>
          </div>
        </div>
      </div>

      <div v-else class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
          <circle cx="8.5" cy="8.5" r="1.5"/>
          <polyline points="21 15 16 10 5 21"/>
        </svg>
        <h3>暂无私有镜像</h3>
        <p>ghci.io/philicxie 下暂未找到镜像</p>
      </div>
    </template>

    <!-- 我的收藏 -->
    <template v-if="activeTab === 'favorites'">
      <div v-if="favoritesLoading" class="loading-state">
        <el-skeleton :rows="4" animated />
      </div>

      <div v-else-if="favorites.length > 0" class="images-grid">
        <div
          v-for="img in favorites"
          :key="img.id"
          class="image-card"
        >
          <div class="image-header-row">
            <div class="image-name-block">
              <h3 class="image-name" :title="img.image_name">{{ img.image_name }}</h3>
              <el-tag v-if="img.is_official" size="small" type="success" effect="plain">官方</el-tag>
            </div>
            <el-button
              circle
              type="warning"
              @click.stop="removeFavorite(img)"
            >
              <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
                <path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"/>
              </svg>
            </el-button>
          </div>
          <p class="image-desc">{{ img.description || '暂无描述' }}</p>
          <div class="image-footer-row">
            <span class="image-stars">
              <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
                <path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z"/>
              </svg>
              {{ formatStars(img.star_count) }}
            </span>
          </div>
        </div>
      </div>

      <div v-else class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/>
        </svg>
        <h3>暂无收藏</h3>
        <p>在公开镜像中搜索并收藏常用镜像，方便快速使用</p>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

const currentUserId = 'default'

const tabs = [
  { key: 'public', label: '公开镜像' },
  { key: 'private', label: '私有镜像' },
  { key: 'favorites', label: '我的收藏' },
]

const activeTab = ref('public')

// 公开镜像搜索
const searchQuery = ref('')
const searchResults = ref([])
const searchLoading = ref(false)
const hasSearched = ref(false)

// 私有镜像
const privateImages = ref([])
const privateLoading = ref(false)

// 收藏
const favorites = ref([])
const favoritesLoading = ref(false)
const favoriteCount = computed(() => favorites.value.length)

// 监听 Tab 切换，自动加载数据
watch(activeTab, (tab) => {
  if (tab === 'private') {
    fetchPrivateImages()
  } else if (tab === 'favorites') {
    fetchFavorites()
  }
})

const doSearch = async () => {
  const q = searchQuery.value.trim()
  if (!q) {
    searchResults.value = []
    hasSearched.value = false
    return
  }
  searchLoading.value = true
  hasSearched.value = true
  try {
    const res = await fetch(
      `/api/v1/images/search?q=${encodeURIComponent(q)}&user_id=${currentUserId}`
    )
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    searchResults.value = data.images || []
  } catch (err) {
    console.error('search images failed:', err)
    ElMessage.error('搜索失败')
    searchResults.value = []
  } finally {
    searchLoading.value = false
  }
}

const fetchPrivateImages = async () => {
  if (privateImages.value.length > 0) return
  privateLoading.value = true
  try {
    const res = await fetch('/api/v1/images/private?provider=ppio')
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    privateImages.value = data.images || []
  } catch (err) {
    console.error('fetch private images failed:', err)
    ElMessage.error('获取私有镜像失败')
  } finally {
    privateLoading.value = false
  }
}

const fetchFavorites = async () => {
  favoritesLoading.value = true
  try {
    const res = await fetch(`/api/v1/images/favorites?user_id=${currentUserId}`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    favorites.value = data.favorites || []
  } catch (err) {
    console.error('fetch favorites failed:', err)
    ElMessage.error('获取收藏失败')
  } finally {
    favoritesLoading.value = false
  }
}

const toggleFavorite = async (img) => {
  if (img.is_favorited) {
    // 取消收藏
    try {
      const res = await fetch(
        `/api/v1/images/favorites?user_id=${currentUserId}&image_name=${encodeURIComponent(img.name)}`,
        { method: 'DELETE' }
      )
      if (!res.ok) throw new Error()
      img.is_favorited = false
      // 如果当前在收藏列表中，刷新收藏
      if (activeTab.value === 'favorites') {
        await fetchFavorites()
      }
      ElMessage.success('已取消收藏')
    } catch {
      ElMessage.error('取消收藏失败')
    }
  } else {
    // 添加收藏
    try {
      const res = await fetch('/api/v1/images/favorites', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          user_id: currentUserId,
          image_name: img.name,
          description: img.description,
          star_count: img.star_count,
          is_official: img.is_official,
        }),
      })
      if (!res.ok) throw new Error()
      img.is_favorited = true
      ElMessage.success('收藏成功')
    } catch {
      ElMessage.error('收藏失败')
    }
  }
}

const removeFavorite = async (img) => {
  try {
    const res = await fetch(
      `/api/v1/images/favorites?user_id=${currentUserId}&image_name=${encodeURIComponent(img.image_name)}`,
      { method: 'DELETE' }
    )
    if (!res.ok) throw new Error()
    await fetchFavorites()
    // 同步更新搜索结果中的收藏状态
    const found = searchResults.value.find(s => s.name === img.image_name)
    if (found) found.is_favorited = false
    ElMessage.success('已取消收藏')
  } catch {
    ElMessage.error('取消收藏失败')
  }
}

const formatStars = (count) => {
  if (count >= 1000000) return (count / 1000000).toFixed(1) + 'M'
  if (count >= 1000) return (count / 1000).toFixed(1) + 'k'
  return count.toString()
}

const formatTime = (t) => {
  if (!t) return '-'
  const d = new Date(t)
  const now = new Date()
  const diff = now - d
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  if (days === 0) return '今天'
  if (days === 1) return '昨天'
  if (days < 30) return `${days} 天前`
  if (days < 365) return `${Math.floor(days / 30)} 个月前`
  return `${Math.floor(days / 365)} 年前`
}

const shortName = (name) => {
  if (!name) return ''
  const parts = name.split('/')
  return parts[parts.length - 1]
}

onMounted(() => {
  // 默认加载收藏数量，但不切换Tab时不显示
})
</script>

<style scoped>
.images-page {
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 24px;
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

/* Tab 切换 */
.tab-bar {
  display: flex;
  gap: 4px;
  margin-bottom: 24px;
  background: var(--bg-secondary);
  padding: 4px;
  border-radius: var(--radius-lg);
  width: fit-content;
}

.tab-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 20px;
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  transition: all 0.2s;
  user-select: none;
}

.tab-item:hover {
  color: var(--text-primary);
}

.tab-item.active {
  background: white;
  color: var(--primary-color);
  box-shadow: var(--shadow-sm);
}

.tab-badge {
  background: var(--primary-color);
  color: white;
  font-size: 11px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 10px;
  min-width: 18px;
  text-align: center;
}

/* 搜索栏 */
.search-bar {
  margin-bottom: 24px;
}

.search-bar :deep(.el-input__wrapper) {
  box-shadow: 0 0 0 1px var(--border-color) inset;
  border-radius: var(--radius-lg);
  padding: 4px 12px;
}

.search-bar :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--primary-color) inset;
}

.search-bar :deep(.el-input-group__append) {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
  border-radius: 0 var(--radius-lg) var(--radius-lg) 0;
  padding: 0 24px;
}

/* 镜像网格 */
.images-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.image-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: 20px;
  transition: all 0.2s;
}

.image-card:hover {
  border-color: var(--primary-color);
  box-shadow: var(--shadow-md);
}

.image-header-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 12px;
}

.image-name-block {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  min-width: 0;
}

.image-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  word-break: break-all;
  line-height: 1.4;
}

.image-desc {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
  margin: 0 0 12px 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  min-height: 40px;
}

.image-footer-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-color);
}

.image-stars {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--text-secondary);
}

.image-stars svg {
  color: #f59e0b;
}

.image-meta {
  font-size: 12px;
  color: var(--text-secondary);
}

/* 私有镜像卡片 */
.private-card .image-name {
  font-family: 'JetBrains Mono', monospace;
  font-size: 14px;
}

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 80px 20px;
  color: var(--text-secondary);
}

.empty-icon {
  width: 64px;
  height: 64px;
  margin: 0 auto 16px;
  opacity: 0.3;
}

.empty-state h3 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.empty-state p {
  font-size: 14px;
  margin: 0;
}

/* 加载状态 */
.loading-state {
  padding: 40px 20px;
}
</style>
