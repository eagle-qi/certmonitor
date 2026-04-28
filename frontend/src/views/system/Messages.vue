<template>
  <div class="page-container">
    <div class="page-header">
      <h2>消息中心</h2>
      <el-button type="primary" @click="markAllRead" :disabled="!unreadCount">全部已读</el-button>
    </div>

    <el-card shadow="never">
      <el-table :data="tableData" v-loading="loading" stripe>
        <el-table-column prop="title" label="标题" min-width="200">
          <template #default="{ row }"><span :style="{ fontWeight: row.read ? 'normal' : 'bold' }">{{ row.title }}</span></template>
        </el-table-column>
        <el-table-column prop="type" label="类型" width="100"><template #default="{ row }"><el-tag size="small">{{ row.type || '通知' }}</el-tag></template></el-table-column>
        <el-table-column prop="createdAt" label="时间" width="180" />
        <el-table-column label="操作" width="80"><template #default="{ row }"><el-button v-if="!row.read" link type="primary" @click="markRead(row)">标记已读</el-button></template></el-table-column>
      </el-table>

      <div class="pagination-wrap">
        <el-pagination v-model:current-page="page" :total="total" layout="prev, pager, next" @current-change="loadData" />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { messageApi } from '@/api/modules/message'

const loading = ref(false)
const tableData = ref([])
const total = ref(0)
const page = ref(1)
const unreadCount = ref(0)

async function loadData() {
  loading.value = true
  try { const res = await messageApi.list({ page: page.value }); tableData.value = res.data?.list || []; total.value = res.data?.total || 0 }
  catch (e) {} finally { loading.value = false }
}

async function markAllRead() { try { await messageApi.markAllRead(); loadData() } catch (e) {} }
async function markRead(row) { try { await messageApi.markRead(row.id); row.read = true; unreadCount.value-- } catch (e) {} }

onMounted(async () => {
  loadData()
  try { const res = await messageApi.unreadCount(); unreadCount.value = res.data?.count || 0 } catch (e) {}
})
</script>
<style scoped>.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; }</style>
