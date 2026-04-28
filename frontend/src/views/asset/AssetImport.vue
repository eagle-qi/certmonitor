<template>
  <div class="page-container">
    <div class="page-header">
      <h2>批量导入</h2>
    </div>

    <el-card shadow="never">
      <el-alert title="支持 Excel 格式文件导入，每行包含：域名/IP、端口、协议类型" type="info" show-icon :closable="false" style="margin-bottom: 20px;" />

      <el-upload
        drag action="#" :auto-upload="false" accept=".xlsx,.xls,.csv"
        :on-change="handleFileChange"
      >
        <el-icon class="upload-icon"><UploadFilled /></el-icon>
        <div class="el-upload__text">将文件拖到此处，或<em>点击上传</em></div>
        <template #tip><div class="el-upload__tip">仅支持 .xlsx/.xls/.csv 文件，单次最大 10MB</div></template>
      </el-upload>

      <div v-if="file" style="margin-top: 20px;">
        <p>已选择文件：<strong>{{ file.name }}</strong>（{{ (file.size / 1024).toFixed(1) }} KB）</p>
        <el-button type="primary" :loading="importing" @click="handleImport">开始导入</el-button>
      </div>

      <el-divider />

      <h4>操作指引</h4>
      <ol>
        <li>下载 <el-button link type="primary" @click="downloadTemplate">导入模板</el-button></li>
        <li>按照模板格式填写资产信息</li>
        <li>上传填写完成的文件并点击"开始导入"</li>
        <li>在"资产列表 > 导入日志"中查看导入结果</li>
      </ol>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled } from '@element-plus/icons-vue'
import { assetApi } from '@/api/modules/asset'

const file = ref(null)
const importing = ref(false)

function handleFileChange(fileInfo) {
  file.value = fileInfo.raw
}

async function downloadTemplate() {
  try {
    const res = await assetApi.downloadTemplate()
    const url = window.URL.createObjectURL(new Blob([res]))
    const link = document.createElement('a')
    link.href = url; link.download = 'asset_import_template.xlsx'
    document.body.appendChild(link); link.click(); document.body.removeChild(link)
  } catch (e) { /* handled */ }
}

async function handleImport() {
  if (!file.value) return ElMessage.warning('请先选择文件')
  importing.value = true
  try {
    await assetApi.batchImport(file.value, () => {})
    ElMessage.success('导入任务已提交，请在导入日志中查看结果')
  } catch (e) { /* handled */ }
  finally { importing.value = false }
}
</script>
