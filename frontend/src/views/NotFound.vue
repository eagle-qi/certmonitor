<template>
  <div class="not-found">
    <div class="content">
      <h1>404</h1>
      <p>抱歉，您访问的页面不存在</p>
      <el-button type="primary" @click="$router.push('/')">返回首页</el-button>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.not-found {
  display: flex; align-items: center; justify-content: center;
  min-height: 100vh; background: #f0f2f5;
}
.content { text-align: center; }
h1 { font-size: 120px; color: #409eff; margin: 0; line-height: 1; }
p { color: #909399; margin: 16px 0 32px; font-size: 18px; }
</style>
