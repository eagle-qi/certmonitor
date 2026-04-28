<template>
  <div class="page-container">
    <div class="page-header">
      <h2>资产列表</h2>
      <div>
        <el-button @click="$router.push('/assets/import')">批量导入</el-button>
        <el-button type="primary" @click="showCreateDialog = true">新增资产</el-button>
      </div>
    </div>

    <!-- 搜索筛选 -->
    <el-card shadow="never" style="margin-bottom: 16px;">
      <el-form :inline="true">
        <el-form-item label="域名/IP"><el-input v-model="query.keyword" placeholder="搜索..." clearable /></el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.status" clearable placeholder="全部">
            <el-option label="正常" value="active" />
            <el-option label="待确认" value="pending" />
            <el-option label="已禁用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item><el-button type="primary" @click="loadData">查询</el-button></el-form-item>
      </el-form>
    </el-card>

    <!-- 数据表格 -->
    <el-card shadow="never">
      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="domain" label="域名/IP" min-width="180" />
        <el-table-column prop="port" label="端口" width="80" />
        <el-table-column prop="protocol" label="协议" width="80" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : row.status === 'pending' ? 'warning' : 'info'" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="网页标题" min-width="200" show-overflow-tooltip />
        <el-table-column prop="certExpiry" label="证书过期时间" width="170" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="$router.push(`/assets/${row.id}`)">详情</el-button>
            <el-button link type="danger">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrap">
        <el-pagination
          v-model:current-page="page" v-model:page-size="pageSize"
          :total="total" layout="total, prev, pager, next, jumper"
          @current-change="loadData"
        />
      </div>
    </el-card>

    <!-- 新增对话框 -->
    <el-dialog v-model="showCreateDialog" title="新增资产" width="500px">
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="域名/IP"><el-input v-model="createForm.domain" /></el-form-item>
        <el-form-item label="端口"><el-input-number v-model="createForm.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item label="协议">
          <el-radio-group v-model="createForm.protocol">
            <el-radio value="HTTPS">HTTPS</el-radio>
            <el-radio value="HTTP">HTTP</el-radio>
            <el-radio value="ALL">ALL</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="showCreateDialog = false">取消</el-button><el-button type="primary">确定</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { assetApi } from '@/api/modules/asset'

const loading = ref(false)
const tableData = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const query = reactive({ keyword: '', status: '' })
const showCreateDialog = ref(false)
const createForm = reactive({ domain: '', port: 443, protocol: 'HTTPS' })

async function loadData() {
  loading.value = true
  try {
    const res = await assetApi.list({ ...query, page: page.value, pageSize: pageSize.value })
    tableData.value = res.data?.list || []
    total.value = res.data?.total || 0
  } catch (e) { /* handled */ }
  finally { loading.value = false }
}

function statusText(s) { return { active: '正常', pending: '待确认', disabled: '已禁用' }[s] || s }

loadData()
</script>

<style scoped>
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
