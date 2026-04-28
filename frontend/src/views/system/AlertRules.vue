<template>
  <div class="page-container">
    <div class="page-header"><h2>告警规则</h2><el-button type="primary" @click="showDialog = true">新建规则</el-button></div>
    <el-card shadow="never">
      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="name" label="规则名称" min-width="180" />
        <el-table-column prop="type" label="类型" width="120"><template #default="{ row }"><el-tag size="small">{{ row.type }}</el-tag></template></el-table-column>
        <el-table-column prop="condition" label="触发条件" min-width="200" show-overflow-tooltip />
        <el-table-column prop="enabled" label="状态" width="80">
          <template #default="{ row }">
            <el-switch :model-value="row.enabled" @change="(val) => toggleRule(row.id, val)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160">
          <template #default><el-button link type="primary">编辑</el-button><el-button link type="danger">删除</el-button></template>
        </el-table-column>
      </el-table>

      <el-dialog v-model="showDialog" title="新建告警规则" width="500px">
        <el-form :model="form" label-width="100px">
          <el-form-item label="规则名称"><el-input v-model="form.name" /></el-form-item>
          <el-form-item label="规则类型"><el-select v-model="form.type"><el-option value="cert_expiry" /><el-option value="asset_change" /></el-select></el-form-item>
          <el-form-item label="触发条件"><el-input v-model="form.condition" placeholder="如：证书剩余天数 <= 30" /></el-form-item>
          <el-form-item label="通知方式">
            <el-checkbox-group v-model="form.notifyChannels"><el-checkbox value="email">邮件</el-checkbox><el-checkbox value="webhook">Webhook</el-checkbox></el-checkbox-group>
          </el-form-item>
        </el-form>
        <template #footer><el-button @click="showDialog = false">取消</el-button><el-button type="primary">确定</el-button></template>
      </el-dialog>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { systemApi } from '@/api/modules/system'

const loading = ref(false)
const tableData = ref([])
const showDialog = ref(false)
const form = reactive({ name: '', type: 'cert_expiry', condition: '', notifyChannels: [] })

onMounted(async () => {
  loading.value = true
  try { const res = await systemApi.listAlerts(); tableData.value = res.data || [] }
  catch (e) {} finally { loading.value = false }
})

async function toggleRule(id, enabled) {
  try { await systemApi.toggleAlert(id) }
  catch (e) {}
}
</script>
