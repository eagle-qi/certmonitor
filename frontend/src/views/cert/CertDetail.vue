<template>
  <div class="page-container">
    <div class="page-header">
      <h2>证书详情</h2>
      <el-button @click="$router.back()">返回列表</el-button>
    </div>

    <el-card shadow="never" v-loading="loading">
      <el-descriptions :column="2" border v-if="cert.id">
        <el-descriptions-item label="域名">{{ cert.domain }}</el-descriptions-item>
        <el-descriptions-item label="颁发者">{{ cert.issuer }}</el-descriptions-item>
        <el-descriptions-item label="有效期起">{{ cert.notBefore }}</el-descriptions-item>
        <el-descriptions-item label="有效期至"><span :style="{ color: (cert.daysLeft || 999) <= 30 ? '#f56c6c' : '#67c23a' }">{{ cert.notAfter }}（剩余 {{ cert.daysLeft != null ? cert.daysLeft + '天' : '-' }}）</span></el-descriptions-item>
        <el-descriptions-item label="序列号">{{ cert.serialNumber || '-' }}</el-descriptions-item>
        <el-descriptions-item label="签名算法">{{ cert.signatureAlgorithm || '-' }}</el-descriptions-item>
        <el-descriptions-item label="公钥类型">{{ cert.publicKeyType || '-' }}</el-descriptions-item>
        <el-descriptions-item label="公钥长度">{{ cert.publicKeySize ? `${cert.publicKeySize} bit` : '-' }}</el-descriptions-item>
        <el-descriptions-item label="SHA-256 指纹" :span="2">{{ cert.fingerprintSha256 || '-' }}</el-descriptions-item>
        <el-descriptions-item label="SAN 域名" :span="2">
          <el-tag v-for="san in (cert.sanDomains || [])" :key="san" size="small" style="margin-right: 4px; margin-bottom: 4px;">{{ san }}</el-tag>
          <span v-if="!cert.sanDomains?.length">无</span>
        </el-descriptions-item>
      </el-descriptions>

      <div style="margin-top: 20px; display: flex; gap: 12px;">
        <el-button type="primary">重新采集</el-button>
        <el-button type="success">下载证书</el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { certApi } from '@/api/modules/certificate'

const route = useRoute()
const loading = ref(false)
const cert = ref({})

onMounted(async () => {
  loading.value = true
  try { const res = await certApi.detail(route.params.id); cert.value = res.data || {} }
  catch (e) {} finally { loading.value = false }
})
</script>
