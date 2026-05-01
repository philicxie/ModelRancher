import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import TaskList from '../views/TaskList.vue'
import TaskCreate from '../views/TaskCreate.vue'
import TaskDetail from '../views/TaskDetail.vue'
import Storage from '../views/Storage.vue'
import Images from '../views/Images.vue'
import Instances from '../views/Instances.vue'
import InstanceDetail from '../views/InstanceDetail.vue'

const routes = [
  { path: '/', name: 'Dashboard', component: Dashboard },
  { path: '/tasks', name: 'TaskList', component: TaskList },
  { path: '/create', name: 'TaskCreate', component: TaskCreate },
  { path: '/task/:id', name: 'TaskDetail', component: TaskDetail, props: true },
  { path: '/storage', name: 'Storage', component: Storage },
  { path: '/images', name: 'Images', component: Images },
  { path: '/instances', name: 'Instances', component: Instances },
  { path: '/instance/:id', name: 'InstanceDetail', component: InstanceDetail, props: true }
]

export default createRouter({
  history: createWebHistory(),
  routes
})