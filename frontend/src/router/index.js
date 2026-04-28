import { createRouter, createWebHistory } from 'vue-router'
import TaskList from './views/TaskList.vue'
import TaskCreate from './views/TaskCreate.vue'
import TaskDetail from './views/TaskDetail.vue'

const routes = [
  { path: '/', name: 'TaskList', component: TaskList },
  { path: '/create', name: 'TaskCreate', component: TaskCreate },
  { path: '/task/:id', name: 'TaskDetail', component: TaskDetail, props: true }
]

export default createRouter({
  history: createWebHistory(),
  routes
})