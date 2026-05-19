import axios from 'axios'
import { ElMessage } from 'element-plus'
import { t } from '../locales'

const api = axios.create({
  baseURL: '/api',
  timeout: 10000
})

// Interceptor cho request
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Interceptor cho response
api.interceptors.response.use(
  (response) => {
    return response
  },
  (error) => {
    const silentError = error.config?.silentError === true
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.href = '/login'
    } else if (!silentError) {
      ElMessage.error(error.response?.data?.error || t('common.requestFailed'))
    }
    return Promise.reject(error)
  }
)

export default api
