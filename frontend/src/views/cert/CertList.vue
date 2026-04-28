<template>
  <div class="page-container">
    <div class="page-header">
      <h2>证书监控</h2>
      <el-button type="primary" @click="$router.push('/certificates/apply')">申请证书</el-button>
    </div>

    <el-card shadow="never" style="margin-bottom: 16px;">
      <el-form :inline="true">
        <el-form-item label="域名"><el-input v-model="query.domain" placeholder="搜索域名..." clearable /></el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.status" clearable placeholder="全部">
            <el-option label="有效" value="valid" /><el-option label="即将过期" value="expiring" /><el-option label="已过期" value="expired" />
          </el-select>
        </el-form-item>
        <el-form-item><el-button type="primary" @click="loadData">查询</el-button></el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="domain" label="域名" min-width="180" />
        <el-table-column prop="issuer" label="颁发者" min-width="200" show-overflow-tooltip />
        <el-table-column prop="notAfter" label="过期时间" width="170" />
        <el-table-column prop="daysLeft" label="剩余天数" width="100">
          <template #default="{ row }">
            <span :style="{ color: (row.daysLeft || 999) <= 30 ? '#f56c6c' : '#67c23a', fontWeight: 'bold' }">{{ row.daysLeft != null ? `${row.daysLeft}天` : '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="90"><template #default="{ row }"><el-tag :type="row.status === 'valid' ? 'success' : row.status === 'expiring' ? 'warning' : 'danger'" size="small">{{ row.status }}</el-tag></template></el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="$router.push(`/certificates/${row.id}`)">详情</el-button>
            <el-button link type="success">采集</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap">
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total" layout="total, prev, pager, next, jumper" @current-change="loadData" />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { certApi } from '@/api/modules/certificate'

const loading = ref(false)
const tableData = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const query = reactive({ domain: '', status: '' })

async function loadData() {
  loading.value = true
  try { const res = await certApi.list({ ...query, page: page.value, pageSize: pageSize.value }); tableData.value = res.data?.list || []; total.value = res.data?.total || 0 }
  catch (e) {} finally { loading.value = false }
}
loadData()
</script>
<style scoped>.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; }</style>
