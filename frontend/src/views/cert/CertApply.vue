<template>
  <div class="page-container">
    <div class="page-header"><h2>申请证书</h2></div>
    <el-card shadow="never">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="140px" style="max-width: 650px;">
        <el-form-item label="申请类型" prop="applyType">
          <el-radio-group v-model="form.applyType">
            <el-radio :value="1">公网域名 (ACME)</el-radio>
            <el-radio :value="2">内网IP (自签名)</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="域名/IP" prop="applyAddr"><el-input v-model="form.applyAddr" placeholder="如：example.com 或 192.168.1.100" /></el-form-item>
        <el-form-item label="SAN 额外地址"><el-input v-model="form.sanAddrs" placeholder="逗号分隔，如：www.example.com,api.example.com" /></el-form-item>
        <el-form-item label="验证方式" prop="verifyMethod" v-if="form.applyType === 1">
          <el-radio-group v-model="form.verifyMethod">
            <el-radio :value="1">DNS验证</el-radio><el-radio :value="2">HTTP文件验证</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="加密算法">
          <el-select v-model="form.encryptAlgorithm"><el-option value="RSA" /><el-option value="ECDSA" /></el-select>
        </el-form-item>
        <el-form-item label="密钥长度" v-if="form.encryptAlgorithm === 'RSA'">
          <el-select v-model="form.keySize"><el-option :value="2048" /><el-option :value="4096" /></el-select>
        </el-form-item>
        <el-form-item label="有效期（天）"><el-input-number v-model="form.validDays" :min="30" :max="3650" /></el-form-item>
        <el-form-item><el-button type="primary" :loading="submitting" @click="handleSubmit">提交申请</el-button></el-form-item>
      </el-form>

      <el-divider />
      <div class="tips">
        <p><strong>公网域名：</strong>通过 ACME 协议自动向 Let's Encrypt 申请免费 SSL 证书，需要域名已解析到本服务器。</p>
        <p><strong>内网 IP：</strong>自动生成自签名 CA 和证书，适用于内部系统或测试环境。</p>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { certApplyApi } from '@/api/modules/certificate'

const formRef = ref(null)
const submitting = ref(false)
const form = reactive({ applyType: 1, applyAddr: '', sanAddrs: '', verifyMethod: 1, encryptAlgorithm: 'RSA', keySize: 2048, validDays: 365 })
const rules = { applyType: [{ required: true }], applyAddr: [{ required: true, message: '请输入域名或IP', trigger: 'blur' }] }

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try { await certApplyApi.submit(form); ElMessage.success('申请已提交') }
  catch (e) {} finally { submitting.value = false }
}
</script>

<style scoped>.tips p { color: #909399; font-size: 13px; line-height: 2; margin: 0; }</style>
