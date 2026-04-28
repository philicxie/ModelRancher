<template>
  <div class="task-create">
    <el-card>
      <template #header>
        <span>创建训练任务</span>
      </template>

      <el-form :model="form" label-width="120px" :rules="rules" ref="formRef">
        <el-form-item label="任务名称" prop="name">
          <el-input v-model="form.name" placeholder="输入任务名称" />
        </el-form-item>

        <el-form-item label="任务描述" prop="description">
          <el-input v-model="form.description" type="textarea" rows="3" placeholder="输入任务描述" />
        </el-form-item>

        <el-form-item label="Docker镜像" prop="image">
          <el-select v-model="form.image" placeholder="选择镜像" style="width: 100%">
            <el-option label="PyTorch 2.0" value="pytorch/pytorch:2.0.1-cuda11.7-cudnn8-runtime" />
            <el-option label="TensorFlow 2.13" value="tensorflow/tensorflow:2.13.0" />
            <el-option label="Python 3.10" value="python:3.10-slim" />
            <el-option label="Python 3.9" value="python:3.9-slim" />
          </el-select>
        </el-form-item>

        <el-form-item label="训练命令" prop="command">
          <el-input
            v-model="form.command"
            type="textarea"
            rows="4"
            placeholder="python train.py --data-dir /data --output-dir /output" />
          <div class="form-tip">
            命令将在容器中执行，数据目录挂载在 /data，输出目录挂载在 /output
          </div>
        </el-form-item>

        <el-form-item label="数据目录(COS)" prop="dataPath">
          <el-input v-model="form.dataPath" placeholder="cos://bucket/data/" />
          <div class="form-tip">可选，指定训练数据在COS中的路径</div>
        </el-form-item>

        <el-form-item label="输出目录(COS)" prop="outputPath">
          <el-input v-model="form.outputPath" placeholder="cos://bucket/output/" />
          <div class="form-tip">可选，训练结果将上传到此路径</div>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleSubmit" :loading="loading">创建任务</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>
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
const loading = ref(false)

const form = reactive({
  name: '',
  description: '',
  image: 'python:3.10-slim',
  command: '',
  data_path: '',
  output_path: ''
})

const rules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  image: [{ required: true, message: '请选择镜像', trigger: 'change' }],
  command: [{ required: true, message: '请输入训练命令', trigger: 'blur' }]
}

const handleSubmit = async () => {
  try {
    await formRef.value.validate()
    loading.value = true

    const taskData = {
      name: form.name,
      description: form.description,
      image: form.image,
      command: form.command,
      data_path: form.dataPath || '',
      output_path: form.outputPath || ''
    }

    await taskStore.createTask(taskData)
    ElMessage.success('任务创建成功')
    router.push('/')
  } catch (err) {
    if (err !== false) {
      ElMessage.error('创建失败: ' + err.message)
    }
  } finally {
    loading.value = false
  }
}

const handleReset = () => {
  formRef.value?.resetFields()
}
</script>

<style scoped>
.task-create {
  max-width: 800px;
  margin: 0 auto;
}
.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>