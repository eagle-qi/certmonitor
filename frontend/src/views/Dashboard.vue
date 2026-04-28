<template>
  <div class="page-container">
    <div class="page-header">
      <h2>工作台</h2>
      <span class="time">{{ currentTime }}</span>
    </div>

    <el-row :gutter="20" class="stat-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: linear-gradient(135deg, #667eea, #764ba2);">
            <el-icon :size="28"><FolderOpened /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.totalAssets }}</div>
            <div class="stat-label">资产总数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: linear-gradient(135deg, #f093fb, #f5576c);">
            <el-icon :size="28"><Lock /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.totalCerts }}</div>
            <div class="stat-label">证书总数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: linear-gradient(135deg, #4facfe, #00f2fe);">
            <el-icon :size="28"><Warning /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.expiringSoon }}</div>
            <div class="stat-label">即将过期</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: linear-gradient(135deg, #43e97b, #38f9d7);">
            <el-icon :size="28"><Odometer /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.runningTasks }}</div>
            <div class="stat-label">运行中任务</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="16">
        <el-card header="最近探测任务">
          <el-table :data="recentTasks" stripe size="small">
            <el-table-column prop="name" label="任务名称" />
            <el-table-column prop="type" label="类型" width="100" />
            <el-table-column prop="status" label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="statusType(row.status)" size="small">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="createdAt" label="创建时间" width="180" />
          </el-table>
          <div v-if="!recentTasks.length" class="empty-tip">暂无任务数据</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card header="证书过期预警">
          <div v-for="item in expiringCerts" :key="item.domain" class="cert-item">
            <div class="cert-domain">{{ item.domain }}</div>
            <el-progress
              :percentage="item.percentage"
              :color="progressColor(item.daysLeft)"
              :show-text="false"
              style="margin: 4px 0;"
            />
            <div class="cert-days">剩余 {{ item.daysLeft }} 天</div>
          </div>
          <div v-if="!expiringCerts.length" class="empty-tip">暂无预警</div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { FolderOpened, Lock, Warning, Odometer } from '@element-plus/icons-vue'

const stats = reactive({ totalAssets: 0, totalCerts: 0, expiringSoon: 0, runningTasks: 0 })
const recentTasks = ref([])
const expiringCerts = ref([])
const currentTime = ref('')

let timer = null

onMounted(() => {
  updateTime()
  timer = setInterval(updateTime, 1000)
})

onUnmounted(() => { clearInterval(timer) })

function updateTime() { currentTime.value = new Date().toLocaleString('zh-CN') }

function statusType(status) {
  const map = { '运行中': 'success', '已完成': 'info', '失败': 'danger', '等待中': 'warning' }
  return map[status] || 'info'
}

function progressColor(days) { if (days <= 7) return '#f56c6c'; if (days <= 30) return '#e6a23c'; return '#67c23a' }
</script>

<style lang="scss" scoped>
.stat-row .stat-card {
  display: flex; align-items: center; padding: 4px 16px;
}
.stat-icon {
  width: 56px; height: 56px; border-radius: 12px;
  display: flex; align-items: center; justify-content: center; color: #fff;
  margin-right: 16px; flex-shrink: 0;
}
.stat-info { flex: 1; min-width: 0; }
.stat-value { font-size: 26px; font-weight: 700; color: #303133; line-height: 1.2; }
.stat-label { font-size: 13px; color: #909399; margin-top: 2px; }

.time { color: #909399; font-size: 14px; }
.empty-tip { text-align: center; padding: 30px; color: #909399; }

.cert-item {
  padding: 10px 0; border-bottom: 1px solid #ebeef5;
  &:last-child { border-bottom: none; }
}
.cert-domain { font-size: 13px; color: #303133; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cert-days { font-size: 12px; color: #909399; }
</style>
