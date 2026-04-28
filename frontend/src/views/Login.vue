<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-header">
        <h1>CertMonitor</h1>
        <p>URL探测与证书生命周期管理系统</p>
      </div>

      <el-form ref="formRef" :model="form" :rules="rules" size="large">
        <el-form-item prop="username">
          <el-input v-model="form.username" placeholder="用户名" prefix-icon="User" />
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="form.password" type="password" placeholder="密码" prefix-icon="Lock"
                    show-password @keyup.enter="handleLogin" />
        </el-form-item>

        <div class="login-options">
          <el-checkbox v-model="rememberMe">记住登录状态</el-checkbox>
          <a href="#">忘记密码？</a>
        </div>

        <el-button type="primary" :loading="loading" style="width: 100%" @click="handleLogin">
          登 录
        </el-button>
      </el-form>

      <div class="login-footer">
        还没有账号？<router-link to="/register">立即注册</router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/store/user'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const formRef = ref(null)
const loading = ref(false)
const rememberMe = ref(true)

const form = reactive({ username: '', password: '' })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function handleLogin() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    await userStore.login(form)
    ElMessage.success('登录成功')
    router.push(route.query.redirect || '/dashboard')
  } catch (e) {
    // error handled by interceptor
  } finally {
    loading.value = false
  }
}
</script>

<style lang="scss" scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
  width: 420px; padding: 40px;
  background: #fff; border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0,0,0,0.15);
}

.login-header { text-align: center; margin-bottom: 32px; }
.login-header h1 {
  font-size: 28px; font-weight: 700;
  color: #303133; margin-bottom: 8px;
}
.login-header p { color: #909399; font-size: 14px; }

.login-options {
  display: flex; justify-content: space-between;
  margin-bottom: 20px; color: #606266;
  a { color: #409eff; text-decoration: none; }
}

.login-footer {
  margin-top: 20px; text-align: center; color: #909399; font-size: 14px;
  a { color: #409eff; text-decoration: none; }
}
</style>
