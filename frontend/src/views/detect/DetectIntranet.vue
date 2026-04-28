<template>
  <div class="page-container">
    <div class="page-header"><h2>内网网段探测</h2></div>
    <el-card shadow="never">
      <el-form :model="form" label-width="120px" style="max-width: 600px;">
        <el-form-item label="任务名称"><el-input v-model="form.taskName" placeholder="如：办公区内网扫描" /></el-form-item>
        <el-form-item label="IP/CIDR 网段" required><el-input v-model="form.ipSegment" placeholder="如：192.168.1.0/24" /></el-form-item>
        <el-form-item label="端口范围"><el-input v-model="form.portRange" placeholder="默认: 80,443,8080,8443" value="80,443,8080,8443" /></el-form-item>
        <el-form-item label="协议类型">
          <el-radio-group v-model="form.protocolType">
            <el-radio label="ALL">全部</el-radio><el-radio label="HTTP">HTTP</el-radio><el-radio label="HTTPS">HTTPS</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item><el-button type="primary" :loading="submitting" @click="handleSubmit">创建探测任务</el-button></el-form-item>
      </el-form>

      <el-divider />
      <h4>历史任务</h4>
      <el-table :data="tasks" stripe border size="small" v-loading="loading">
        <el-table-column prop="taskName" label="任务名称" />
        <el-table-column prop="ipSegment" label="网段" />
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
const form = reactive({ taskName: '', ipSegment: '', portRange: '80,443,8080,8443', protocolType: 'ALL' })

onMounted(() => loadTasks())

async function loadTasks() {
  loading.value = true
  try { const res = await detectApi.listIntranetTasks({}); tasks.value = res.data?.list || [] }
  catch (e) {} finally { loading.value = false }
}

async function handleSubmit() {
  if (!form.ipSegment) return ElMessage.warning('请输入 IP/CIDR 网段')
  submitting.value = true
  try { await detectApi.createIntranetTask(form); ElMessage.success('任务已提交'); loadTasks() }
  catch (e) {} finally { submitting.value = false }
}
</script>
