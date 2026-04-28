<template>
  <div class="page-container">
    <div class="page-header"><h2>用户管理</h2><el-button type="primary" @click="showDialog = true">新建用户</el-button></div>
    <el-card shadow="never">
      <el-table :data="tableData" v-loading="loading" stripe border>
        <el-table-column prop="username" label="用户名" width="140" />
        <el-table-column prop="realName" label="真实姓名" width="120" />
        <el-table-column prop="email" label="邮箱" min-width="180" />
        <el-table-column prop="status" label="状态" width="80"><template #default="{ row }"><el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">{{ row.status === 'active' ? '启用' : '禁用' }}</el-tag></template></el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="170" />
        <el-table-column label="操作" width="200">
          <template #default><el-button link type="primary">编辑</el-button><el-button link type="warning">重置密码</el-button><el-button link type="danger">删除</el-button></template>
        </el-table-column>
      </el-table>

      <el-dialog v-model="showDialog" title="新建用户" width="450px"><el-form :model="form" label-width="80px"><el-form-item label="用户名"><el-input v-model="form.username" /></el-form-item><el-form-item label="邮箱"><el-input v-model="form.email" /></el-form-item><el-form-item label="角色"><el-select v-model="form.roleId" multiple style="width: 100%;" /></el-form-item></el-form><template #footer><el-button @click="showDialog = false">取消</el-button><el-button type="primary">确定</el-button></template></el-dialog>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { systemApi } from '@/api/modules/system'

const loading = ref(false)
const tableData = ref([])
const showDialog = ref(false)
const form = reactive({ username: '', email: '', roleId: [] })

onMounted(async () => {
  loading.value = true
  try { const res = await systemApi.listUsers({}); tableData.value = res.data?.list || [] }
  catch (e) {} finally { loading.value = false }
})
</script>
