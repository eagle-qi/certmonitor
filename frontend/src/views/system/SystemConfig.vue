<template>
  <div class="page-container">
    <div class="page-header"><h2>系统配置</h2></div>
    <el-card shadow="never">
      <el-form :model="form" label-width="160px" style="max-width: 700px;" v-loading="loading">
        <el-divider content-position="left">基础配置</el-divider>
        <el-form-item label="系统名称"><el-input v-model="form.systemName" /></el-form-item>
        <el-form-item label="默认语言"><el-select v-model="form.defaultLang"><el-option value="zh-CN" /><el-option value="en-US" /></el-select></el-form-item>

        <el-divider content-position="left">探测配置</el-divider>
        <el-form-item label="默认超时(秒)"><el-input-number v-model="form.defaultTimeout" :min="5" :max="60" /></el-form-item>
        <el-form-item label="最大并发数"><el-input-number v-model="form.maxConcurrent" :min="10" :max="500" /></el-form-item>

        <el-divider content-position="left">证书告警</el-divider>
        <el-form-item label="过期预警天数"><el-input-number v-model="form.expiryWarnDays" :min="7" :max="90" /></el-form-item>
        <el-form-item label="告警通知邮箱"><el-input v-model="form.alertEmail" type="textarea" placeholder="多个邮箱用逗号分隔" /></el-form-item>

        <el-form-item><el-button type="primary" :loading="saving" @click="handleSave">保存配置</el-button></el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'

const loading = ref(false)
const saving = ref(false)
const form = reactive({ systemName: 'CertMonitor', defaultLang: 'zh-CN', defaultTimeout: 10, maxConcurrent: 100, expiryWarnDays: 30, alertEmail: '' })

function handleSave() {
  saving.value = true
  setTimeout(() => { saving.value = false; ElMessage.success('配置已保存') }, 500)
}
</script>
