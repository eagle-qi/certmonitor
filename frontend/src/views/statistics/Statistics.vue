<template>
  <div class="page-container">
    <div class="page-header"><h2>统计报表</h2></div>

    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stat-row">
      <el-col :span="6" v-for="(stat, i) in statCards" :key="i">
        <el-card shadow="hover" class="mini-stat">
          <div class="value">{{ stat.value }}</div>
          <div class="label">{{ stat.label }}</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 图表区域 -->
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card header="资产协议分布"><div ref="protocolChartRef" style="height: 300px;"></div></el-card>
      </el-col>
      <el-col :span="12">
        <el-card header="证书过期趋势"><div ref="expiryChartRef" style="height: 300px;"></div></el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="24"><el-card header="探测任务统计"><div ref="taskChartRef" style="height: 300px;"></div></el-card></el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'

const protocolChartRef = ref(null)
const expiryChartRef = ref(null)
const taskChartRef = ref(null)

const statCards = reactive([
  { label: '总资产数', value: '-' },
  { label: '有效证书', value: '-' },
  { label: '即将过期', value: '-' },
  { label: '本月新增', value: '-' },
])

onMounted(() => {
  // 图表初始化（实际使用 ECharts 渲染）
})
</script>

<style scoped>
.stat-row .mini-stat { text-align: center; padding: 24px; }
.stat-row .value { font-size: 32px; font-weight: 700; color: #303133; }
.stat-row .label { color: #909399; margin-top: 8px; font-size: 14px; }
</style>
