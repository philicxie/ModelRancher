<template>
  <div class="images-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">镜像管理</h1>
        <p class="page-subtitle">选择和管理训练环境</p>
      </div>
    </div>

    <!-- 镜像分类 -->
    <div class="image-categories">
      <el-radio-group v-model="selectedCategory" size="large">
        <el-radio-button label="all">全部</el-radio-button>
        <el-radio-button label="pytorch">PyTorch</el-radio-button>
        <el-radio-button label="tensorflow">TensorFlow</el-radio-button>
        <el-radio-button label="custom">自定义</el-radio-button>
      </el-radio-group>
    </div>

    <!-- 镜像列表 -->
    <div class="images-grid">
      <div
        v-for="image in filteredImages"
        :key="image.id"
        class="image-card"
        :class="{ 'selected': selectedImage === image.id }"
        @click="selectImage(image)">
        <div class="image-header">
          <div class="image-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
              <circle cx="8.5" cy="8.5" r="1.5"/>
              <polyline points="21 15 16 10 5 21"/>
            </svg>
          </div>
          <div class="image-category-tag" :class="image.category">{{ image.category }}</div>
        </div>

        <h3 class="image-name">{{ image.name }}</h3>
        <p class="image-version">{{ image.version }}</p>
        <p class="image-description">{{ image.description }}</p>

        <div class="image-footer">
          <span class="image-size">{{ image.size }}</span>
          <span class="image-tag-count">{{ image.tags.length }} tags</span>
        </div>

        <div class="selected-indicator" v-if="selectedImage === image.id">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
        </div>
      </div>
    </div>

    <!-- 选中镜像详情 -->
    <div class="selected-info" v-if="selectedImage">
      <el-card class="detail-card">
        <template #header>
          <span>已选镜像</span>
        </template>
        <div class="detail-content">
          <p class="detail-name">{{ getSelectedImageInfo().name }}</p>
          <p class="detail-version">{{ getSelectedImageInfo().version }}</p>
          <el-button type="primary" @click="$router.push('/create')">
            使用此镜像创建任务
          </el-button>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

const selectedCategory = ref('all')
const selectedImage = ref(null)

const images = ref([
  {
    id: 1,
    name: 'PyTorch',
    version: '2.0.1-cuda11.7-cudnn8-runtime',
    description: 'PyTorch 2.0 官方镜像，预装 CUDA 11.7 和 cuDNN 8',
    category: 'pytorch',
    size: '~8GB',
    tags: ['2.0.1', '2.0.0', '1.13.1']
  },
  {
    id: 2,
    name: 'TensorFlow',
    version: '2.13.0-gpu',
    description: 'TensorFlow 2.13 GPU 版本，预装 CUDA 11.8',
    category: 'tensorflow',
    size: '~7GB',
    tags: ['2.13.0', '2.12.0', '2.11.0']
  },
  {
    id: 3,
    name: 'Python',
    version: '3.10-slim',
    description: '轻量级 Python 3.10 基础镜像，适合简单任务',
    category: 'custom',
    size: '~500MB',
    tags: ['3.10', '3.9', '3.8']
  },
  {
    id: 4,
    name: 'JAX',
    version: 'latest-cuda11-pjax',
    description: 'Google JAX 框架，包含 CUDA 和 cuDNN',
    category: 'pytorch',
    size: '~9GB',
    tags: ['latest']
  },
  {
    id: 5,
    name: 'PyTorch',
    version: '1.13.1-cpu',
    description: 'PyTorch 1.13 CPU 版本，无 GPU 支持',
    category: 'pytorch',
    size: '~5GB',
    tags: ['1.13.1', '1.13.0']
  },
  {
    id: 6,
    name: 'MXNet',
    version: '1.9.1-gpu',
    description: 'Apache MXNet 深度学习框架 GPU 版本',
    category: 'tensorflow',
    size: '~6GB',
    tags: ['1.9.1']
  }
])

const filteredImages = computed(() => {
  if (selectedCategory.value === 'all') {
    return images.value
  }
  return images.value.filter(img => img.category === selectedCategory.value)
})

const selectImage = (image) => {
  selectedImage.value = image.id
}

const getSelectedImageInfo = () => {
  return images.value.find(img => img.id === selectedImage.value) || {}
}
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

.image-categories {
  margin-bottom: 24px;
}

.images-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
  margin-bottom: 24px;
}

.image-card {
  background: var(--bg-secondary);
  border: 2px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: 20px;
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
}

.image-card:hover {
  border-color: var(--primary-color);
  box-shadow: var(--shadow-md);
}

.image-card.selected {
  border-color: var(--primary-color);
  background: linear-gradient(135deg, rgba(79, 70, 229, 0.05), transparent);
}

.image-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}

.image-icon {
  width: 48px;
  height: 48px;
  background: var(--bg-primary);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary-color);
}

.image-icon svg {
  width: 24px;
  height: 24px;
}

.image-category-tag {
  font-size: 11px;
  padding: 4px 8px;
  border-radius: 4px;
  font-weight: 500;
  text-transform: uppercase;
}

.image-category-tag.pytorch {
  background: #fee2e2;
  color: #dc2626;
}

.image-category-tag.tensorflow {
  background: #fef3c7;
  color: #d97706;
}

.image-category-tag.custom {
  background: #dbeafe;
  color: #2563eb;
}

.image-name {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.image-version {
  font-size: 13px;
  color: var(--primary-color);
  margin-bottom: 8px;
  font-family: monospace;
}

.image-description {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.5;
  margin-bottom: 16px;
}

.image-footer {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--text-secondary);
  padding-top: 12px;
  border-top: 1px solid var(--border-color);
}

.selected-indicator {
  position: absolute;
  top: 16px;
  right: 16px;
  width: 28px;
  height: 28px;
  background: var(--primary-color);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.selected-indicator svg {
  width: 16px;
  height: 16px;
}

.selected-info {
  margin-top: 24px;
}

.detail-card {
  border-radius: var(--radius-lg);
}

.detail-content {
  text-align: center;
  padding: 16px;
}

.detail-name {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.detail-version {
  font-size: 14px;
  color: var(--primary-color);
  margin-bottom: 16px;
  font-family: monospace;
}
</style>