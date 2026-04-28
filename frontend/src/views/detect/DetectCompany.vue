<template>
  <div class="page-container">
    <div class="page-header"><h2>公网备案探测</h2></div>
    <el-card shadow="never">
      <el-form :model="form" label-width="120px" style="max-width: 600px;">
        <el-form-item label="企业名称" required><el-input v-model="form.companyName" placeholder="输入需要探测的企业全称" /></el-form-item>
        <el-form-item><el-button type="primary" :loading="submitting" @click="handleSubmit">创建探测任务</el-button></el-form-item>
      </el-form>

      <el-divider />

      <h4>历史任务</h4>
      <el-table :data="tasks" stripe border size="small" v-loading="loading">
        <el-table-column prop="companyName" label="企业名称" />
        <el-table-column prop="status" label="状态" width="100"><template #default="{ row }"><el-tag size="small">{{ row.status }}</el-tag></template></el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { detectApi } from '@/api/modules/detect'

const loading = ref(false)
const submitting = ref(false)
const tasks = ref([])
const form = reactive({ companyName: '' })

onMounted(() => loadTasks())

async function loadTasks() {
  loading.value = true
  try { const res = await detectApi.listCompanyTasks({}); tasks.value = res.data?.list || [] }
  catch (e) {} finally { loading.value = false }
}

async function handleSubmit() {
  if (!form.companyName) return ElMessage.warning('请输入企业名称')
  submitting.value = true
  try { await detectApi.createCompanyTask(form); ElMessage.success('任务已提交'); form.companyName = ''; loadTasks() }
  catch (e) {} finally { submitting.value = false }
}
</script>
