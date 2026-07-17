<script setup lang="ts">
</script>

<template>
  <el-container>
    <el-header>
      <el-row align="middle" style="height: 100%">
        <el-col>
          <h1 style="margin: 0">kh e-library</h1>
        </el-col>
        <el-col :span="4" style="text-align: right">
          <el-button @click="$router.push('/settings')">设置</el-button>
        </el-col>
      </el-row>
    </el-header>
    <el-main>
      <el-empty description="暂无书籍" />
    </el-main>
  </el-container>
</template>