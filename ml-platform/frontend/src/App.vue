<template>
  <div class="app-container">
    <!-- 侧边栏 -->
    <aside class="sidebar" :class="{ 'collapsed': isSidebarCollapsed }">
      <div class="sidebar-header">
        <div class="logo">
          <svg class="logo-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2L2 7l10 5 10-5-10-5z"/>
            <path d="M2 17l10 5 10-5"/>
            <path d="M2 12l10 5 10-5"/>
          </svg>
          <span v-if="!isSidebarCollapsed" class="logo-text">ML Platform</span>
        </div>
        <button class="collapse-btn" @click="toggleSidebar">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path v-if="isSidebarCollapsed" d="M9 18l6-6-6-6"/>
            <path v-else d="M15 18l-6-6 6-6"/>
          </svg>
        </button>
      </div>

      <nav class="sidebar-nav">
        <router-link to="/" class="nav-item" :class="{ 'active': route.path === '/' }">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="7" height="7"/>
            <rect x="14" y="3" width="7" height="7"/>
            <rect x="14" y="14" width="7" height="7"/>
            <rect x="3" y="14" width="7" height="7"/>
          </svg>
          <span v-if="!isSidebarCollapsed">Dashboard</span>
        </router-link>

        <router-link to="/tasks" class="nav-item" :class="{ 'active': route.path === '/tasks' }">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
            <polyline points="14 2 14 8 20 8"/>
            <line x1="16" y1="13" x2="8" y2="13"/>
            <line x1="16" y1="17" x2="8" y2="17"/>
          </svg>
          <span v-if="!isSidebarCollapsed">训练任务</span>
        </router-link>

        <router-link to="/create" class="nav-item" :class="{ 'active': route.path === '/create' }">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="16"/>
            <line x1="8" y1="12" x2="16" y2="12"/>
          </svg>
          <span v-if="!isSidebarCollapsed">创建任务</span>
        </router-link>

        <div class="nav-divider"></div>

        <div class="nav-group-label" v-if="!isSidebarCollapsed">资源</div>
        <router-link to="/instances" class="nav-item" :class="{ 'active': route.path === '/instances' }">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
            <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
          </svg>
          <span v-if="!isSidebarCollapsed">实例管理</span>
        </router-link>

        <div class="nav-divider"></div>

        <div class="nav-group-label" v-if="!isSidebarCollapsed">存储</div>
        <router-link to="/storage" class="nav-item" :class="{ 'active': route.path === '/storage' }">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
          </svg>
          <span v-if="!isSidebarCollapsed">对象存储</span>
        </router-link>

        <router-link to="/images" class="nav-item" :class="{ 'active': route.path === '/images' }">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            <circle cx="8.5" cy="8.5" r="1.5"/>
            <polyline points="21 15 16 10 5 21"/>
          </svg>
          <span v-if="!isSidebarCollapsed">镜像管理</span>
        </router-link>
      </nav>

      <div class="sidebar-footer">
        <div class="user-info" v-if="!isSidebarCollapsed">
          <div class="user-avatar">U</div>
          <div class="user-details">
            <div class="user-name">User</div>
            <div class="user-role">Member</div>
          </div>
        </div>
      </div>
    </aside>

    <!-- 主内容区 -->
    <main class="main-content">
      <!-- 内容区 -->
      <div class="content-wrapper">
        <router-view />
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const isSidebarCollapsed = ref(false)

const toggleSidebar = () => {
  isSidebarCollapsed.value = !isSidebarCollapsed.value
}
</script>

<style>
/* CSS变量 - 全局主题 */
:root {
  --primary-color: #4f46e5;
  --primary-hover: #4338ca;
  --secondary-color: #06b6d4;
  --success-color: #10b981;
  --warning-color: #f59e0b;
  --danger-color: #ef4444;
  --bg-primary: #f8fafc;
  --bg-secondary: #ffffff;
  --bg-sidebar: #1e293b;
  --text-primary: #1e293b;
  --text-secondary: #64748b;
  --text-light: #f8fafc;
  --border-color: #e2e8f0;
  --shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.05);
  --shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.1);
  --shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.1);
  --radius-sm: 6px;
  --radius-md: 8px;
  --radius-lg: 12px;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Inter', 'Segoe UI', Roboto, sans-serif;
  background: var(--bg-primary);
  color: var(--text-primary);
  line-height: 1.6;
}

/* 应用容器 */
.app-container {
  display: flex;
  min-height: 100vh;
}

/* 侧边栏 */
.sidebar {
  width: 260px;
  background: var(--bg-sidebar);
  color: var(--text-light);
  display: flex;
  flex-direction: column;
  transition: width 0.3s ease;
  position: fixed;
  height: 100vh;
  z-index: 100;
}

.sidebar.collapsed {
  width: 72px;
}

.sidebar-header {
  padding: 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.logo {
  display: flex;
  align-items: center;
  gap: 12px;
}

.logo-icon {
  width: 32px;
  height: 32px;
  color: var(--primary-color);
}

.logo-text {
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -0.02em;
}

.collapse-btn {
  background: none;
  border: none;
  color: var(--text-light);
  cursor: pointer;
  padding: 4px;
  border-radius: var(--radius-sm);
  opacity: 0.7;
  transition: opacity 0.2s;
}

.collapse-btn:hover {
  opacity: 1;
}

.collapse-btn svg {
  width: 20px;
  height: 20px;
}

/* 导航 */
.sidebar-nav {
  flex: 1;
  padding: 16px 12px;
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  color: rgba(255, 255, 255, 0.7);
  text-decoration: none;
  border-radius: var(--radius-md);
  margin-bottom: 4px;
  transition: all 0.2s;
}

.nav-item:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-light);
}

.nav-item.active {
  background: var(--primary-color);
  color: var(--text-light);
}

.nav-icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.nav-divider {
  height: 1px;
  background: rgba(255, 255, 255, 0.1);
  margin: 16px 0;
}

.nav-group-label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: rgba(255, 255, 255, 0.4);
  padding: 8px 16px;
}

/* 用户信息 */
.sidebar-footer {
  padding: 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: var(--primary-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
}

.user-details {
  flex: 1;
}

.user-name {
  font-size: 14px;
  font-weight: 500;
}

.user-role {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.5);
}

/* 主内容区 */
.main-content {
  flex: 1;
  margin-left: 260px;
  transition: margin-left 0.3s ease;
  display: flex;
  flex-direction: column;
}

.sidebar.collapsed + .main-content {
  margin-left: 72px;
}

/* 顶部栏 */
.topbar {
  height: 64px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  padding: 0 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  position: sticky;
  top: 0;
  z-index: 50;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--text-secondary);
}

.breadcrumb-item {
  color: var(--text-secondary);
}

.breadcrumb-item::before {
  content: '/';
  margin-right: 8px;
  opacity: 0.5;
}

.breadcrumb-item:first-child::before {
  display: none;
}

.topbar-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.btn-icon {
  width: 16px;
  height: 16px;
  margin-right: 6px;
}

/* 内容包装器 */
.content-wrapper {
  padding: 24px;
  flex: 1;
}

/* 响应式 */
@media (max-width: 1024px) {
  .sidebar {
    width: 72px;
  }

  .logo-text,
  .nav-item span,
  .nav-group-label,
  .user-info {
    display: none !important;
  }

  .main-content {
    margin-left: 72px;
  }
}
</style>