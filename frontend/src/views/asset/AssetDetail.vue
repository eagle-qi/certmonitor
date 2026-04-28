<template>
  <div class="page-container">
    <div class="page-header">
      <h2>资产详情</h2>
      <el-button @click="$router.back()">返回列表</el-button>
    </div>

    <el-card shadow="never" v-loading="loading">
      <el-descriptions :column="3" border v-if="detail.id">
        <el-descriptions-item label="域名/IP">{{ detail.domain }}</el-descriptions-item>
        <el-descriptions-item label="端口">{{ detail.port }}</el-descriptions-item>
        <el-descriptions-item label="协议">{{ detail.protocol }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag>{{ detail.status === 'active' ? '正常' : detail.status }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="网页标题">{{ detail.title || '-' }}</el-descriptions-item>
        <el-descriptions-item label="发现时间">{{ detail.createdAt || '-' }}</el-descriptions-item>
        <el-descriptions-item label="证书颁发者" :span="3">{{ detail.certIssuer || '无' }}</el-descriptions-item>
        <el-descriptions-item label="证书有效期" :span="2">
          {{ detail.certNotBefore || '-' }} ~ {{ detail.certNotAfter || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="剩余天数">
          <span :style="{ color: (detail.daysLeft || 999) <= 30 ? '#f56c6c' : '#67c23a', fontWeight: 'bold' }">
            {{ detail.daysLeft != null ? `${detail.daysLeft} 天` : '-' }}
          </span>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { assetApi } from '@/api/modules/asset'

const route = useRoute()
const loading = ref(false)
const detail = ref({})

onMounted(async () => {
  loading.value = true
  try {
    const res = await assetApi.detail(route.params.id)
    detail.value = res.data || {}
  } catch (e) { /* handled */ }
  finally { loading.value = false }
})
</script>
