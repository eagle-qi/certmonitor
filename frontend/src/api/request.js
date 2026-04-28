/**
 * Axios 封装 - 统一请求拦截器
 */
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '@/store/user'
import router from '@/router'

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 30000, // 探测任务可能耗时较长
  headers: { 'Content-Type': 'application/json' },
})

// 请求拦截器：自动附加 JWT Token
request.interceptors.request.use(
  (config) => {
    const userStore = useUserStore()
    if (userStore.token) {
      config.headers.Authorization = `Bearer ${userStore.token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器：统一错误处理、Token过期处理等
request.interceptors.response.use(
  (response) => {
    const res = response.data

    // 业务层面的错误码判断
    if (res.code && res.code !== 200) {
      ElMessage.error(res.message || '请求失败')
      
      // Token 过期或无效 -> 跳转登录页
      if ([401, 10001].includes(res.code)) {
        const userStore = useUserStore()
        userStore.logout()
        router.push(`/login?redirect=${router.currentRoute.value.fullPath}`)
      }

      return Promise.reject(new Error(res.message || 'Error'))
    }

    return res
  },
  (error) => {
    if (error.response) {
      const status = error.response.status
      switch (status) {
        case 401:
          ElMessage.error('登录已过期，请重新登录')
          const userStore = useUserStore()
          userStore.logout()
          router.push('/login')
          break
        case 403:
          ElMessage.error('权限不足，无法执行此操作')
          break
        case 404:
          ElMessage.error('请求的资源不存在')
          break
        case 422:
          ElMessage.error('参数校验失败')
          break
        case 429:
          ElMessage.error('请求过于频繁，请稍后再试')
          break
        case 500:
          ElMessage.error('服务器内部错误，请联系管理员')
          break
        default:
          ElMessage.error(error.response.data?.message || `请求失败(${status})`)
      }
    } else if (error.code === 'ECONNABORTED') {
      ElMessage.error('请求超时，请稍后重试')
    } else {
      ElMessage.error('网络异常，请检查网络连接')
    }
    return Promise.reject(error)
  }
)

export default request

// 导出各模块 API 方法
export * from './modules/auth'
export * from './modules/asset'
export * from './modules/certificate'
export * from './modules/detect'
export * from './modules/message'
export * from './modules/system'
