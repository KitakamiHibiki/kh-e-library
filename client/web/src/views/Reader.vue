<script setup lang="ts">
import { useRoute } from 'vue-router'
const route = useRoute()
</script>

<template>
  <el-container>
    <el-header>
      <el-button @click="$router.back()">返回</el-button>
      <span style="margin-left: 16px;">阅读 #{{ route.params.id }}</span>
    </el-header>
    <el-main>
      <p>阅读器组件待实现</p>
    </el-main>
  </el-container>
</template>