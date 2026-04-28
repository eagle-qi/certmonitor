<template>
  <div class="page-container">
    <div class="page-header"><h2>角色管理</h2><el-button type="primary" @click="showDialog = true">新建角色</el-button></div>
    <el-card shadow="never">
      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="name" label="角色名称" width="160" />
        <el-table-column prop="code" label="角色编码" width="140" />
        <el-table-column prop="description" label="描述" />
        <el-table-column prop="userCount" label="用户数" width="80" align="center" />
        <el-table-column label="操作" width="200">
          <template #default><el-button link type="primary">编辑</el-button><el-button link type="danger">删除</el-button></template>
        </el-table-column>
      </el-table>

      <el-dialog v-model="showDialog" title="新建角色" width="450px">
        <el-form :model="form" label-width="80px">
          <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
          <el-form-item label="编码"><el-input v-model="form.code" /></el-form-item>
          <el-form-item label="描述"><el-input type="textarea" v-model="form.description" /></el-form-item>
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
const form = reactive({ name: '', code: '', description: '' })

onMounted(async () => {
  loading.value = true
  try { const res = await systemApi.listRoles(); tableData.value = res.data || [] }
  catch (e) {} finally { loading.value = false }
})
</script>
