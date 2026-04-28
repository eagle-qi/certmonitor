<template>
  <div class="main-layout">
    <!-- 侧边栏 -->
    <aside class="sidebar" :class="{ collapsed: isCollapsed }">
      <div class="logo">
        <h2 v-if="!isCollapsed">CertMonitor</h2>
        <span v-else>CM</span>
      </div>

      <el-menu
        :default-active="activeMenu"
        :collapse="isCollapse"
        router
        background-color="#001529"
        text-color="#ffffffa6"
        active-text-color="#409eff"
      >
        <el-menu-item index="/dashboard">
          <el-icon><Odometer /></el-icon>
          <template #title>工作台</template>
        </el-menu-item>

        <el-sub-menu index="assets">
          <template #title><el-icon><FolderOpened /></el-icon><span>资产管理</span></template>
          <el-menu-item index="/assets">资产列表</el-menu-item>
          <el-menu-item index="/assets/import">批量导入</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="detect">
          <template #title><el-icon><Search /></el-icon><span>探测任务</span></template>
          <el-menu-item index="/detect/company">公网探测</el-menu-item>
          <el-menu-item index="/detect/intranet">内网探测</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="certificates">
          <template #title><el-icon><Lock /></el-icon><span>证书管理</span></template>
          <el-menu-item index="/certificates">证书监控</el-menu-item>
          <el-menu-item index="/certificates/apply">申请证书</el-menu-item>
        </el-sub-menu>

        <el-menu-item index="/messages">
          <el-icon><Bell /></el-icon>
          <template #title>消息中心</template>
        </el-menu-item>

        <el-sub-menu index="system" v-if="userStore.hasRole('super_admin')">
          <template #title><el-icon><Setting /></el-icon><span>系统管理</span></template>
          <el-menu-item index="/system/users">用户管理</el-menu-item>
          <el-menu-item index="/system/roles">角色管理</el-menu-item>
          <el-menu-item index="/system/config">系统配置</el-menu-item>
          <el-menu-item index="/system/logs">操作日志</el-menu-item>
          <el-menu-item index="/system/alerts">告警规则</el-menu-item>
        </el-sub-menu>

        <el-menu-item index="/statistics">
          <el-icon><DataAnalysis /></el-icon>
          <template #title>统计报表</template>
        </el-menu-item>
      </el-menu>
    </aside>

    <!-- 主区域 -->
    <div class="main-area">
      <header class="header">
        <div class="left">
          <el-icon class="collapse-btn" @click="toggleSidebar"><Fold v-if="!isCollapsed" /><Expand v-else /></el-icon>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item>{{ currentTitle || '首页' }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="right">
          <el-badge :value="unreadCount" :hidden="!unreadCount" :max="99">
            <el-icon class="icon-btn" @click="$router.push('/messages')"><Bell /></el-icon>
          </el-badge>
          <el-dropdown trigger="click">
            <span class="user-info">
              <el-avatar :size="28">{{ userStore.username?.charAt(0)?.toUpperCase() }}</el-avatar>
              <span class="username">{{ userStore.username }}</span>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="$router.push('/profile')">个人中心</el-dropdown-item>
                <el-dropdown-item divided @click="handleLogout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { useUserStore } from '@/store/user'
import {
  Odometer, FolderOpened, Search, Lock, Bell, Setting,
  DataAnalysis, Fold, Expand
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const isCollapsed = ref(false)
const unreadCount = ref(0)
const isCollapse = computed(() => isCollapsed.value)
const activeMenu = computed(() => route.path)

const currentTitle = computed(() => route.meta.title || '')

const toggleSidebar = () => {
  isCollapsed.value = !isCollapsed.value
}

const handleLogout = () => {
  ElMessageBox.confirm('确定要退出登录吗？', '提示', { type: 'warning' })
    .then(() => { userStore.logout(); router.push('/login') })
    .catch(() => {})
}
</script>

<style lang="scss" scoped>
.main-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.sidebar {
  width: var(--sidebar-width);
  background: #001529;
  transition: width 0.3s;
  overflow-y: auto;
  overflow-x: hidden;

  &.collapsed { width: 64px; }

  .logo {
    height: var(--header-height);
    display: flex;
    align-items: center;
    justify-content: center;
    border-bottom: 1px solid rgba(255,255,255,0.08);

    h2 {
      color: #fff; font-size: 18px; white-space: nowrap;
    }
    span {
      color: #fff; font-size: 20px; font-weight: 700;
    }
  }

  :deep(.el-menu) { border-right: none; }
}

.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: #f0f2f5;
}

.header {
  height: var(--header-height);
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.06);
  z-index: 10;

  .left { display: flex; align-items: center; gap: 16px; }
  .collapse-btn { font-size: 20px; cursor: pointer; color: #606266; }
  .right { display: flex; align-items: center; gap: 20px; }
  .icon-btn { cursor: pointer; font-size: 18px; color: #606266; }

  .user-info {
    display: flex; align-items: center; gap: 8px; cursor: pointer;
    .username { color: #303133; font-size: 14px; }
  }
}

.content {
  flex: 1;
  overflow-y: auto;
}
</style>
