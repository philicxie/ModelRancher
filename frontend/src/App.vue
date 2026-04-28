<template>
  <div id="app">
    <el-container>
      <el-header>
        <h1>ML训练平台</h1>
        <el-menu mode="horizontal" :default-active="activeIndex" @select="handleMenuSelect">
          <el-menu-item index="tasks">任务列表</el-menu-item>
          <el-menu-item index="create">创建任务</el-menu-item>
        </el-menu>
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const activeIndex = computed(() => {
  if (route.path === '/' || route.path === '/tasks') return 'tasks'
  if (route.path === '/create') return 'create'
  return 'tasks'
})

const handleMenuSelect = (index) => {
  if (index === 'tasks') router.push('/')
  if (index === 'create') router.push('/create')
}
</script>

<style>
#app {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}
.el-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #545c64;
  color: #fff;
}
.el-header h1 {
  margin: 0;
  font-size: 20px;
}
.el-header .el-menu {
  border-bottom: none;
  background: transparent;
}
.el-header .el-menu-item {
  color: #fff !important;
}
.el-header .el-menu-item:hover,
.el-header .el-menu-item.is-active {
  background: rgba(255,255,255,0.1);
}
.el-main {
  padding: 20px;
  background: #f5f5f5;
}
</style>