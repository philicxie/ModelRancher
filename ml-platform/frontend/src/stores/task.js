import { defineStore } from 'pinia'
import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1'
})

export const useTaskStore = defineStore('tasks', {
  state: () => ({
    tasks: [],
    currentTask: null,
    loading: false,
    error: null
  }),

  actions: {
    async fetchTasks() {
      this.loading = true
      try {
        const res = await api.get('/tasks')
        this.tasks = res.data
      } catch (err) {
        this.error = err.message
      } finally {
        this.loading = false
      }
    },

    async createTask(taskData) {
      this.loading = true
      try {
        const res = await api.post('/tasks', taskData)
        this.tasks.push(res.data)
        return res.data
      } catch (err) {
        this.error = err.message
        throw err
      } finally {
        this.loading = false
      }
    },

    async getTask(taskId) {
      this.loading = true
      try {
        const res = await api.get(`/tasks/${taskId}`)
        this.currentTask = res.data
        return res.data
      } catch (err) {
        this.error = err.message
        throw err
      } finally {
        this.loading = false
      }
    },

    async cancelTask(taskId) {
      try {
        await api.post(`/tasks/${taskId}/cancel`)
        const task = this.tasks.find(t => t.id === taskId)
        if (task) task.status = 'cancelled'
      } catch (err) {
        this.error = err.message
        throw err
      }
    },

    async deleteTask(taskId) {
      try {
        await api.delete(`/tasks/${taskId}`)
        this.tasks = this.tasks.filter(t => t.id !== taskId)
      } catch (err) {
        this.error = err.message
        throw err
      }
    }
  }
})