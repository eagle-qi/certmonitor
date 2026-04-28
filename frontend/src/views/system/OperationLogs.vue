<template>
  <div class="page-container">
    <div class="page-header">
      <h2>操作日志</h2>
      <el-button @click="handleExport">导出</el-button>
    </div>

    <el-card shadow="never" style="margin-bottom: 16px;">
      <el-form :inline="true">
        <el-form-item label="操作人"><el-input v-model="query.operator" clearable /></el-form-item>
        <el-form-item label="操作类型"><el-select v-model="query.action" clearable placeholder="全部"><el-option label="登录" value="login" /><el-option label="创建" value="create" /><el-option label="修改" value="update" /><el-option label="删除" value="delete" /></el-select></el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker v-model="query.dateRange" type="daterange" range-separator="至" start-placeholder="开始" end-placeholder="结束" />
        </el-form-item>
        <el-form-item><el-button type="primary" @click="loadData">查询</el-button></el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <el-table :data="tableData" v-loading="loading" stripe border size="small">
        <el-table-column prop="operator" label="操作人" width="120" />
        <el-table-column prop="action" label="操作类型" width="100"><template #default="{ row }"><el-tag size="small">{{ row.action }}</el-tag></template></el-table-column>
        <el-table-column prop="target" label="目标对象" min-width="180" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP地址" width="140" />
        <el-table-column prop="createdAt" label="时间" width="170" />
      </el-table>

      <div class="pagination-wrap">
        <el-pagination v-model:current-page="page" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { systemApi } from '@/api/modules/system'

const loading = ref(false)
const tableData = ref([])
const total = ref(0)
const page = ref(1)
const query = reactive({ operator: '', action: '', dateRange: null })

async function loadData() {
  loading.value = true
  try { const res = await systemApi.listLogs({ ...query, page: page.value }); tableData.value = res.data?.list || []; total.value = res.data?.total || 0 }
  catch (e) {} finally { loading.value = false }
}

function handleExport() { ElMessage.info('导出功能开发中') }

// Import ElMessage for the stub
import { ElMessage } from 'element-plus'

loadData()
</script>

<style scoped>.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; }</style>
