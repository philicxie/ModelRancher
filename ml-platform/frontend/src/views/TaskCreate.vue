<template>
  <div class="task-create-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">创建训练任务</h1>
        <p class="page-subtitle">配置训练环境和参数</p>
      </div>
    </div>

    <!-- 步骤指示器 -->
    <div class="step-indicator">
      <div
        v-for="(step, index) in steps"
        :key="index"
        class="step-item"
        :class="{ 'active': currentStep === index, 'completed': currentStep > index }">
        <div class="step-number">
          <svg v-if="currentStep > index" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
          <span v-else>{{ index + 1 }}</span>
        </div>
        <span class="step-label">{{ step }}</span>
      </div>
    </div>

    <!-- 表单内容 -->
    <div class="form-container">
      <el-card class="form-card">
        <!-- Step 1: 基本信息 -->
        <div v-show="currentStep === 0" class="step-content">
          <h2 class="step-title">基本信息</h2>
          <p class="step-description">填写任务的基本信息</p>

          <el-form :model="form" :rules="rules" ref="formRef" label-position="top">
            <el-form-item label="任务名称" prop="name">
              <el-input
                v-model="form.name"
                placeholder="给任务起一个描述性的名称"
                size="large"
                clearable>
                <template #prefix>
                  <svg class="input-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                  </svg>
                </template>
              </el-input>
            </el-form-item>

            <el-form-item label="任务描述" prop="description">
              <el-input
                v-model="form.description"
                type="textarea"
                :rows="3"
                placeholder="描述这个训练任务的目的和内容（可选）"
                size="large" />
            </el-form-item>
          </el-form>
        </div>

        <!-- Step 2: 环境配置 -->
        <div v-show="currentStep === 1" class="step-content">
          <h2 class="step-title">环境配置</h2>
          <p class="step-description">选择训练环境和镜像</p>

          <el-form :model="form" label-position="top">
            <el-form-item label="Docker镜像">
              <div class="image-grid">
                <div
                  v-for="image in availableImages"
                  :key="image.value"
                  class="image-option"
                  :class="{ 'selected': form.image === image.value }"
                  @click="form.image = image.value">
                  <div class="image-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                      <circle cx="8.5" cy="8.5" r="1.5"/>
                      <polyline points="21 15 16 10 5 21"/>
                    </svg>
                  </div>
                  <div class="image-info">
                    <span class="image-name">{{ image.label }}</span>
                    <span class="image-desc">{{ image.desc }}</span>
                  </div>
                  <div class="selected-check" v-if="form.image === image.value">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                      <polyline points="20 6 9 17 4 12"/>
                    </svg>
                  </div>
                </div>
              </div>
            </el-form-item>

            <el-form-item label="或输入自定义镜像" class="custom-image">
              <el-input
                v-model="form.image"
                placeholder="registry.example.com/my-image:tag"
                size="large">
                <template #prefix>
                  <svg class="input-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
                  </svg>
                </template>
              </el-input>
            </el-form-item>
          </el-form>
        </div>

        <!-- Step 3: 训练命令 -->
        <div v-show="currentStep === 2" class="step-content">
          <h2 class="step-title">训练命令</h2>
          <p class="step-description">配置训练脚本和参数</p>

          <el-form :model="form" label-position="top">
            <el-form-item label="执行命令" prop="command">
              <el-input
                v-model="form.command"
                type="textarea"
                :rows="5"
                placeholder="python train.py --data-dir /data --output-dir /output --epochs 100"
                size="large"
                class="command-input" />
              <div class="form-tip">
                <svg class="tip-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="12" y1="16" x2="12" y2="12"/>
                  <line x1="12" y1="8" x2="12.01" y2="8"/>
                </svg>
                <span>数据目录挂载在 <code>/data</code>，输出目录挂载在 <code>/output</code></span>
              </div>
            </el-form-item>
          </el-form>
        </div>

        <!-- Step 4: 存储配置 -->
        <div v-show="currentStep === 3" class="step-content">
          <h2 class="step-title">存储配置</h2>
          <p class="step-description">配置数据输入和输出位置</p>

          <el-form :model="form" label-position="top">
            <el-form-item label="数据目录 (COS路径)">
              <el-input
                v-model="form.data_path"
                placeholder="my-bucket/train-data/"
                size="large">
                <template #prefix>
                  <svg class="input-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
                    <polyline points="14 2 14 8 20 8"/>
                    <line x1="12" y1="18" x2="12" y2="12"/>
                    <line x1="9" y1="15" x2="15" y2="15"/>
                  </svg>
                </template>
              </el-input>
              <div class="form-tip">训练数据在COS中的路径，将下载到 /data 目录（可选）</div>
            </el-form-item>

            <el-form-item label="输出目录 (COS路径)">
              <el-input
                v-model="form.output_path"
                placeholder="my-bucket/output/"
                size="large">
                <template #prefix>
                  <svg class="input-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                    <polyline points="17 8 12 3 7 8"/>
                    <line x1="12" y1="3" x2="12" y2="15"/>
                  </svg>
                </template>
              </el-input>
              <div class="form-tip">训练结果将上传到此路径（可选）</div>
            </el-form-item>
          </el-form>
        </div>

        <!-- 步骤按钮 -->
        <div class="form-actions">
          <el-button v-if="currentStep > 0" @click="prevStep" size="large">
            <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 12H5"/>
              <path d="M12 19l-7-7 7-7"/>
            </svg>
            上一步
          </el-button>

          <el-button
            v-if="currentStep < steps.length - 1"
            type="primary"
            size="large"
            @click="nextStep">
            下一步
            <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M5 12h14"/>
              <path d="M12 5l7 7-7 7"/>
            </svg>
          </el-button>

          <el-button
            v-if="currentStep === steps.length - 1"
            type="primary"
            size="large"
            @click="handleSubmit"
            :loading="loading">
            <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="20 6 9 17 4 12"/>
            </svg>
            创建任务
          </el-button>
        </div>
      </el-card>

      <!-- 右侧配置预览 -->
      <div class="config-preview" v-if="currentStep < steps.length - 1">
        <el-card class="preview-card">
          <template #header>
            <span>配置预览</span>
          </template>
          <div class="preview-content">
            <div class="preview-item">
              <span class="preview-label">任务名称</span>
              <span class="preview-value">{{ form.name || '-' }}</span>
            </div>
            <div class="preview-item">
              <span class="preview-label">镜像</span>
              <span class="preview-value">{{ form.image || '-' }}</span>
            </div>
            <div class="preview-item">
              <span class="preview-label">命令</span>
              <code class="preview-code">{{ form.command || '-' }}</code>
            </div>
          </div>
        </el-card>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useTaskStore } from '../stores/task'
import { ElMessage } from 'element-plus'

const router = useRouter()
const taskStore = useTaskStore()
const formRef = ref(null)
const currentStep = ref(0)
const loading = ref(false)

const steps = ['基本信息', '环境配置', '训练命令', '存储配置']

const form = reactive({
  name: '',
  description: '',
  image: 'pytorch/pytorch:2.0.1-cuda11.7-cudnn8-runtime',
  command: '',
  data_path: '',
  output_path: ''
})

const availableImages = [
  { label: 'PyTorch 2.0', value: 'pytorch/pytorch:2.0.1-cuda11.7-cudnn8-runtime', desc: 'CUDA 11.7 + cuDNN 8' },
  { label: 'TensorFlow 2.13', value: 'tensorflow/tensorflow:2.13.0-gpu', desc: 'GPU 支持版本' },
  { label: 'Python 3.10', value: 'python:3.10-slim', desc: '轻量级基础镜像' },
  { label: 'Python 3.9', value: 'python:3.9-slim', desc: '轻量级基础镜像' },
  { label: 'JAX', value: 'jax:latest-cuda11-pjax', desc: 'Google JAX 框架' }
]

const rules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  command: [{ required: true, message: '请输入训练命令', trigger: 'blur' }]
}

const nextStep = async () => {
  if (currentStep.value === 0) {
    try {
      await formRef.value.validate()
      currentStep.value++
    } catch (e) {
      return
    }
  } else {
    currentStep.value++
  }
}

const prevStep = () => {
  if (currentStep.value > 0) {
    currentStep.value--
  }
}

const handleSubmit = async () => {
  if (!form.name || !form.command) {
    ElMessage.error('请填写必填项')
    return
  }

  try {
    loading.value = true
    const taskData = {
      name: form.name,
      description: form.description,
      image: form.image,
      command: form.command,
      data_path: form.data_path || '',
      output_path: form.output_path || ''
    }

    await taskStore.createTask(taskData)
    ElMessage.success('任务创建成功')
    router.push('/tasks')
  } catch (err) {
    ElMessage.error('创建失败: ' + (err.message || '未知错误'))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.task-create-page {
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 32px;
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

/* 步骤指示器 */
.step-indicator {
  display: flex;
  justify-content: center;
  margin-bottom: 32px;
  padding: 24px;
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-color);
}

.step-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 24px;
  position: relative;
}

.step-item:not(:last-child)::after {
  content: '';
  position: absolute;
  right: -24px;
  width: 48px;
  height: 2px;
  background: var(--border-color);
}

.step-item.completed:not(:last-child)::after {
  background: var(--primary-color);
}

.step-number {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--bg-primary);
  border: 2px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
  color: var(--text-secondary);
  transition: all 0.3s;
}

.step-item.active .step-number {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.step-item.completed .step-number {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.step-number svg {
  width: 16px;
  height: 16px;
}

.step-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
}

.step-item.active .step-label {
  color: var(--text-primary);
}

/* 表单容器 */
.form-container {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 24px;
}

.form-card {
  border-radius: var(--radius-lg);
}

.step-content {
  padding: 8px 0;
}

.step-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.step-description {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 24px;
}

.input-icon {
  width: 18px;
  height: 18px;
  color: var(--text-secondary);
}

/* 镜像选择 */
.image-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.image-option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: var(--bg-primary);
  border: 2px solid var(--border-color);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
}

.image-option:hover {
  border-color: var(--primary-color);
}

.image-option.selected {
  border-color: var(--primary-color);
  background: rgba(79, 70, 229, 0.05);
}

.image-icon {
  width: 40px;
  height: 40px;
  background: var(--bg-secondary);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary-color);
}

.image-icon svg {
  width: 20px;
  height: 20px;
}

.image-info {
  flex: 1;
}

.image-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  display: block;
  margin-bottom: 2px;
}

.image-desc {
  font-size: 12px;
  color: var(--text-secondary);
}

.selected-check {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 24px;
  height: 24px;
  background: var(--primary-color);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.selected-check svg {
  width: 14px;
  height: 14px;
}

.custom-image {
  margin-top: 16px;
}

.command-input code {
  background: var(--bg-primary);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  color: var(--primary-color);
}

.form-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 8px;
}

.tip-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

/* 步骤按钮 */
.form-actions {
  display: flex;
  justify-content: space-between;
  margin-top: 32px;
  padding-top: 24px;
  border-top: 1px solid var(--border-color);
}

.btn-icon {
  width: 18px;
  height: 18px;
  margin-right: 8px;
}

/* 配置预览 */
.config-preview {
  position: sticky;
  top: 24px;
}

.preview-card {
  border-radius: var(--radius-lg);
}

.preview-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.preview-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.preview-label {
  font-size: 12px;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.preview-value {
  font-size: 14px;
  color: var(--text-primary);
  font-weight: 500;
}

.preview-code {
  font-size: 12px;
  font-family: monospace;
  background: var(--bg-primary);
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  word-break: break-all;
}

/* 响应式 */
@media (max-width: 1024px) {
  .form-container {
    grid-template-columns: 1fr;
  }

  .config-preview {
    display: none;
  }

  .image-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .step-indicator {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }

  .step-item:not(:last-child)::after {
    display: none;
  }
}
</style>