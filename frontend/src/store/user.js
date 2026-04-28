/**
 * 用户状态管理 - Pinia Store
 * 管理用户登录状态、Token、角色权限等
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api/modules/auth'

export const useUserStore = defineStore('user', () => {
  // State
  const token = ref(localStorage.getItem('token') || '')
  const userInfo = ref(JSON.parse(localStorage.getItem('userInfo') || 'null'))
  const roles = ref(JSON.parse(localStorage.getItem('roles') || '[]'))

  // Getters
  const isLoggedIn = computed(() => !!token.value)
  const username = computed(() => userInfo.value?.username || '')
  const email = computed(() => userInfo.value?.email || '')
  const realName = computed(() => userInfo.value?.real_name || '')

  // 判断是否拥有指定角色（超级管理员拥有所有权限）
  const hasRole = (roleCode) => {
    return roles.value.includes(roleCode) || roles.value.includes('super_admin')
  }

  // Actions
  async function login(credentials) {
    const res = await authApi.login(credentials)
    const data = res.data

    token.value = data.token
    userInfo.value = data.user
    roles.value = data.roles || []

    _persistSession()
    return res
  }

  function logout() {
    token.value = ''
    userInfo.value = null
    roles.value = []
    localStorage.removeItem('token')
    localStorage.removeItem('userInfo')
    localStorage.removeItem('roles')
  }

  // 应用启动时恢复会话（仅从本地存储读取，不验证Token有效性）
  function restoreSession() {
    const savedToken = localStorage.getItem('token')
    if (savedToken) {
      token.value = savedToken
      const savedUser = localStorage.getItem('userInfo')
      const savedRoles = localStorage.getItem('roles')
      if (savedUser) userInfo.value = JSON.parse(savedUser)
      if (savedRoles) roles.value = JSON.parse(savedRoles)
    }
  }

  function updateUserInfo(newInfo) {
    userInfo.value = { ...userInfo.value, ...newInfo }
    localStorage.setItem('userInfo', JSON.stringify(userInfo.value))
  }

  function _persistSession() {
    localStorage.setItem('token', token.value)
    localStorage.setItem('userInfo', JSON.stringify(userInfo.value))
    localStorage.setItem('roles', JSON.stringify(roles.value))
  }

  return {
    token, userInfo, roles,
    isLoggedIn, username, email, realName,
    hasRole,
    login, logout, restoreSession, updateUserInfo,
  }
})
