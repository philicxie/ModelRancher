<template>
  <div class="app-container">
    <!-- 侧边栏 -->
    <aside class="sidebar" :class="{ 'collapsed': isSidebarCollapsed }">
      <div class="sidebar-header">
        <div class="logo" >
          <div class="logo-icon-wrap" @click="toggleSidebar">
            <svg class="logo-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 2L2 7l10 5 10-5-10-5z"/>
              <path d="M2 17l10 5 10-5"/>
              <path d="M2 12l10 5 10-5"/>
            </svg>
            <svg class="logo-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path v-if="isSidebarCollapsed" d="M9 18l6-6-6-6"/>
              <path v-else d="M15 18l-6-6 6-6"/>
            </svg>
          </div>
          
        </div>
        <span class="logo-text">ModelRancher</span>
      </div>

      <nav class="sidebar-nav">
        <router-link to="/" class="nav-item" :class="{ 'active': route.path === '/' }">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="7" height="7"/>
            <rect x="14" y="3" width="7" height="7"/>
            <rect x="14" y="14" width="7" height="7"/>
            <rect x="3" y="14" width="7" height="7"/>
          </svg>
          <span class="nav-text">Dashboard</span>
        </router-link>

        <router-link to="/tasks" class="nav-item" :class="{ 'active': route.path === '/tasks' }">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
            <polyline points="14 2 14 8 20 8"/>
            <line x1="16" y1="13" x2="8" y2="13"/>
            <line x1="16" y1="17" x2="8" y2="17"/>
          </svg>
          <span class="nav-text">训练任务</span>
        </router-link>

        <router-link to="/create" class="nav-item" :class="{ 'active': route.path === '/create' }">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="16"/>
            <line x1="8" y1="12" x2="16" y2="12"/>
          </svg>
          <span class="nav-text">创建任务</span>
        </router-link>

        <div class="nav-divider"></div>

        <router-link to="/my-instances" class="nav-item" :class="{ 'active': route.path === '/my-instances' }">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
            <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
          </svg>
          <span class="nav-text">实例管理</span>
        </router-link>
        <router-link to="/browse" class="nav-item" :class="{ 'active': route.path === '/browse' }">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="12 2 2 7 12 12 22 7 12 2"/>
            <polyline points="2 17 12 22 22 17"/>
            <polyline points="2 12 12 17 22 12"/>
          </svg>
          <span class="nav-text">算力市场</span>
        </router-link>

        <div class="nav-divider"></div>

        <router-link to="/storage" class="nav-item" :class="{ 'active': route.path === '/storage' }">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
          </svg>
          <span class="nav-text">对象存储</span>
        </router-link>

        <router-link to="/images" class="nav-item" :class="{ 'active': route.path === '/images' }">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            <circle cx="8.5" cy="8.5" r="1.5"/>
            <polyline points="21 15 16 10 5 21"/>
          </svg>
          <span class="nav-text">镜像管理</span>
        </router-link>
      </nav>

      <div class="sidebar-footer">
        <div class="user-info">
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
  /* 主色调 - 深蓝渐变系 */
  --primary-color: #2563eb;
  --primary-hover: #1d4ed8;
  --primary-light: #dbeafe;

  /* 强调色 - Sky/Cyan */
  --accent-sky: #38bdf8;
  --accent-cyan: #22d3ee;

  /* 辅助色 */
  --secondary-color: #06b6d4;
  --success-color: #10b981;
  --warning-color: #f59e0b;
  --danger-color: #ef4444;

  /* 背景色 */
  --bg-primary: #faf8f5;
  --bg-secondary: #ffffff;
  --bg-sidebar: #0f172a;

  /* 文字颜色 */
  --text-primary: #1e293b;
  --text-secondary: #64748b;
  --text-light: #f8fafc;

  /* 边框和阴影 */
  --border-color: #e2e8f0;
  --shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.05);
  --shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.1);
  --shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.1);
  --shadow-sidebar: 4px 0 24px rgba(0, 0, 0, 0.12);

  /* 圆角 */
  --radius-sm: 6px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-full: 9999px;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body {
  overflow-x: hidden;
}

body {
  font-family: 'Space Grotesk', 'Noto Sans SC', -apple-system, BlinkMacSystemFont, sans-serif;
  background: var(--bg-primary);
  color: var(--text-primary);
  line-height: 1.6;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

/* 应用容器 */
.app-container {
  display: flex;
  min-height: 100vh;
  overflow: hidden;
}

/* 侧边栏 */
.sidebar {
  width: 175px;
  background: var(--bg-sidebar);
  background-image: linear-gradient(180deg, #0f172a 0%, #111827 50%, #0f172a 100%);
  color: var(--text-light);
  display: flex;
  flex-direction: column;
  transition: width 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  position: sticky;
  top: 0;
  height: 100vh;
  flex-shrink: 0;
  overflow: hidden;
  z-index: 100;
}

.sidebar.collapsed {
  width: 64px;
}

.sidebar-header {
  padding: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  min-height: 64px;
}

.logo {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow: hidden;
  cursor: pointer;
  padding: 6px 8px;
  margin: -6px -8px;
  border-radius: var(--radius-md);
  transition: background 0.2s;
}

.logo:hover {
  background: rgba(255, 255, 255, 0.08);
}

.logo-icon-wrap {
  position: relative;
  width: 24px;
  height: 24px;
  flex-shrink: 0;
}

.logo-icon {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 24px;
  height: 24px;
  color: var(--accent-sky);
  opacity: 1;
  transition: opacity 0.15s ease;
}

.logo-arrow {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 20px;
  height: 20px;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.logo:hover .logo-icon {
  opacity: 0;
}

.logo:hover .logo-arrow {
  opacity: 1;
}

.sidebar.collapsed .logo-text {
  max-width: 0;
  opacity: 0;
}

.logo-text,
.nav-text,
.user-details,
.nav-group-label {
  white-space: nowrap;
  overflow: hidden;
  max-width: 200px;
  opacity: 1;
  transition: max-width 0.25s ease 0.15s, opacity 0.2s ease 0.2s;
}

.sidebar.collapsed .logo-text,
.sidebar.collapsed .nav-text,
.sidebar.collapsed .user-details,
.sidebar.collapsed .nav-group-label {
  max-width: 0;
  opacity: 0;
  transition: max-width 0.12s ease, opacity 0.08s ease;
}

.logo-icon {
  width: 32px;
  height: 32px;
  color: var(--accent-sky);
}

.logo-text {
  font-size: 16px;
  font-weight: 600;
  letter-spacing: -0.02em;
  white-space: nowrap;
}

/* 导航 */
.sidebar-nav {
  flex: 1;
  padding: 12px 8px;
  overflow-y: auto;
  overflow-x: hidden;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  color: rgba(255, 255, 255, 0.7);
  text-decoration: none;
  border-radius: var(--radius-md);
  margin-bottom: 4px;
  transition: all 0.25s ease;
  white-space: nowrap;
}

.sidebar.collapsed .nav-item {
  justify-content: center;
  padding: 10px 0;
  gap: 0;
}

.nav-item:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-light);
  box-shadow: 0 0 16px rgba(56, 189, 248, 0.12);
}

.nav-item.active {
  background: linear-gradient(135deg, #2563eb 0%, #06b6d4 100%);
  color: var(--text-light);
  box-shadow: 0 4px 14px rgba(37, 99, 235, 0.35);
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
  font-weight: 600;
  color: rgba(255, 255, 255, 0.4);
  padding: 8px 16px;
}

/* 用户信息 */
.sidebar-footer {
  padding: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.sidebar.collapsed .sidebar-footer {
  padding: 12px 0;
  display: flex;
  justify-content: center;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.sidebar.collapsed .user-info {
  justify-content: center;
}

.user-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: linear-gradient(135deg, #2563eb, #06b6d4);
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
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
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
  .nav-text,
  .nav-group-label {
    display: none !important;
  }
}
</style>