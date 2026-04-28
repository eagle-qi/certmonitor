import { createRouter, createWebHistory } from 'vue-router'
import NProgress from 'nprogress'
import 'nprogress/nprogress.css'
import { useUserStore } from '@/store/user'

NProgress.configure({ showSpinner: false })

const routes = [
  // ==================== 登录页(无需认证)====================
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false },
  },

  // ==================== 主布局框架(需认证) ====================
  {
    path: '/',
    component: () => import('@/layout/MainLayout.vue'),
    meta: { requiresAuth: true },
    redirect: '/dashboard',
    children: [
      // 仪表盘
      { path: 'dashboard', name: 'Dashboard', component: () => import('@/views/Dashboard.vue'), meta: { title: '工作台' } },

      // --- 资产管理 ---
      { path: 'assets', name: 'AssetList', component: () => import('@/views/asset/AssetList.vue'), meta: { title: '资产列表' } },
      { path: 'assets/import', name: 'AssetImport', component: () => import('@/views/asset/AssetImport.vue'), meta: { title: '批量导入' } },
      { path: 'assets/:id', name: 'AssetDetail', component: () => import('@/views/asset/AssetDetail.vue'), meta: { title: '资产详情' } },

      // --- 探测任务 ---
      { path: 'detect/company', name: 'DetectCompany', component: () => import('@/views/detect/DetectCompany.vue'), meta: { title: '公网探测' } },
      { path: 'detect/intranet', name: 'DetectIntranet', component: () => import('@/views/detect/DetectIntranet.vue'), meta: { title: '内网探测' } },

      // --- 证书管理 ---
      { path: 'certificates', name: 'CertList', component: () => import('@/views/cert/CertList.vue'), meta: { title: '证书监控' } },
      { path: 'certificates/apply', name: 'CertApply', component: () => import('@/views/cert/CertApply.vue'), meta: { title: '申请证书' } },
      { path: 'certificates/:id', name: 'CertDetail', component: () => import('@/views/cert/CertDetail.vue'), meta: { title: '证书详情' } },

      // --- 预警通知 ---
      { path: 'messages', name: 'Messages', component: () => import('@/views/system/Messages.vue'), meta: { title: '消息中心' } },

      // --- 系统管理(管理员) ---
      { path: 'system/users', name: 'UserManagement', component: () => import('@/views/system/UserManagement.vue'), meta: { title: '用户管理', role: ['super_admin'] } },
      { path: 'system/roles', name: 'RoleManagement', component: () => import('@/views/system/RoleManagement.vue'), meta: { title: '角色管理', role: ['super_admin'] } },
      { path: 'system/config', name: 'SystemConfig', component: () => import('@/views/system/SystemConfig.vue'), meta: { title: '系统配置', role: ['super_admin'] } },
      { path: 'system/logs', name: 'OperationLogs', component: () => import('@/views/system/OperationLogs.vue'), meta: { title: '操作日志', role: ['super_admin'] } },
      { path: 'system/alerts', name: 'AlertRules', component: () => import('@/views/system/AlertRules.vue'), meta: { title: '告警规则', role: ['super_admin', 'cert_admin'] } },

      // --- 统计报表 ---
      { path: 'statistics', name: 'Statistics', component: () => import('@/views/statistics/Statistics.vue'), meta: { title: '统计报表' } },

      // --- 个人中心 ---
      { path: 'profile', name: 'Profile', component: () => import('@/views/system/Profile.vue'), meta: { title: '个人中心' } },
    ],
  },

  // 404 兜底
  { path: '/:pathMatch(.*)*', name: 'NotFound', component: () => import('@/views/NotFound.vue') },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 路由守卫：认证检查
router.beforeEach((to, from, next) => {
  NProgress.start()

  const userStore = useUserStore()

  if (to.meta.requiresAuth !== false && !userStore.isLoggedIn) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
  } else if (to.name === 'Login' && userStore.isLoggedIn) {
    next({ name: 'Dashboard' })
  } else if (to.meta.role && !to.meta.role.some(r => userStore.hasRole(r))) {
    // 权限不足，跳转首页或提示无权访问
    next(false)
    ElMessage.warning('权限不足，无法访问该页面')
  } else {
    next()
  }
})

router.afterEach(() => {
  NProgress.done()
  document.title = `${document.title || ''}` // 可根据路由meta.title动态设置标题
})

export default router
