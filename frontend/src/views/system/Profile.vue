<template>
  <div class="page-container">
    <div class="page-header"><h2>个人中心</h2></div>

    <el-row :gutter="20">
      <el-col :span="8">
        <el-card shadow="never" class="profile-card">
          <div class="avatar-wrap">
            <el-avatar :size="80">{{ userStore.username?.charAt(0)?.toUpperCase() }}</el-avatar>
            <h3>{{ userStore.username }}</h3>
            <p>{{ userStore.email || '未设置邮箱' }}</p>
          </div>
          <div class="role-tags">
            <el-tag v-for="role in (userStore.roles || [])" :key="role" size="small" style="margin-right: 4px;">{{ role }}</el-tag>
          </div>
        </el-card>
      </el-col>

      <el-col :span="16">
        <el-card shadow="never">
          <el-tabs v-model="activeTab">
            <el-tab-pane label="基本信息" name="basic">
              <el-form :model="form" label-width="100px" style="max-width: 500px;">
                <el-form-item label="用户名"><el-input :model-value="userStore.username" disabled /></el-form-item>
                <el-form-item label="真实姓名"><el-input v-model="form.realName" placeholder="请输入真实姓名" /></el-form-item>
                <el-form-item label="邮箱"><el-input v-model="form.email" /></el-form-item>
                <el-form-item><el-button type="primary">保存修改</el-button></el-form-item>
              </el-form>
            </el-tab-pane>

            <el-tab-pane label="修改密码" name="password">
              <el-form :model="pwdForm" :rules="pwdRules" ref="pwdRef" label-width="100px" style="max-width: 400px;">
                <el-form-item label="当前密码" prop="oldPassword"><el-input v-model="pwdForm.oldPassword" type="password" show-password /></el-form-item>
                <el-form-item label="新密码" prop="newPassword"><el-input v-model="pwdForm.newPassword" type="password" show-password /></el-form-item>
                <el-form-item label="确认新密码" prop="confirmPassword">
                  <el-input v-model="pwdForm.confirmPassword" type="password" show-password />
                </el-form-item>
                <el-form-item><el-button type="primary">修改密码</el-button></el-form-item>
              </el-form>
            </el-tab-pane>
          </el-tabs>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
const activeTab = ref('basic')
const pwdRef = ref(null)

const form = reactive({ realName: '', email: '' })
const pwdForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })
const pwdRules = {
  oldPassword: [{ required: true, message: '请输入当前密码', trigger: 'blur' }],
  newPassword: [{ required: true, message: '请输入新密码', trigger: 'blur' }, { min: 6, message: '至少6个字符', trigger: 'blur' }],
  confirmPassword: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    { validator: (rule, value, callback) => { if (value !== pwdForm.newPassword) callback(new Error('两次密码不一致')); else callback() }, trigger: 'blur' }
  ],
}
</script>

<style scoped>
.profile-card { text-align: center; }
.avatar-wrap {
  padding: 20px 0;
  h3 { margin-top: 12px; color: #303133; }
  p { color: #909399; font-size: 13px; margin-top: 4px; }
}
.role-tags { padding-top: 16px; border-top: 1px solid #ebeef5; }
</style>
