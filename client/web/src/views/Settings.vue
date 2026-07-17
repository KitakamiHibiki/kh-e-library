<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'
import type { AxiosResponse } from 'axios'

const settings = ref<Record<string, string>>({})
const loading = ref(false)

const labels: Record<string, string> = {
  'ui.theme': '界面主题',
  'ui.page_size': '每页数量',
  'ui.sort_field': '排序字段',
  'ui.sort_order': '排序方式',
  'reader.font_size': '阅读字号',
}

const themeOptions = [
  { label: '浅色', value: 'light' },
  { label: '深色', value: 'dark' },
  { label: '跟随系统', value: 'auto' },
]

const sortFieldOptions = [
  { label: '更新时间', value: 'updated_at' },
  { label: '书名', value: 'title' },
  { label: '作者', value: 'author' },
  { label: '创建时间', value: 'created_at' },
]

const sortOrderOptions = [
  { label: '降序', value: 'desc' },
  { label: '升序', value: 'asc' },
]

const fetchSettings = async () => {
  loading.value = true
  try {
    const res: AxiosResponse<{ data: Record<string, string> }> = await api.get('/settings')
    settings.value = res.data.data
  } catch (e) {
    console.error('Failed to load settings', e)
  } finally {
    loading.value = false
  }
}

const save = async () => {
  try {
    await api.put('/settings', { settings: settings.value })
    ElMessage.success('设置已保存')
  } catch (e) {
    ElMessage.error('保存失败')
  }
}

onMounted(fetchSettings)
</script>

<template>
  <el-container>
    <el-header>
      <el-row align="middle" style="height: 100%">
        <el-col>
          <h1 style="margin: 0">设置</h1>
        </el-col>
        <el-col :span="4" style="text-align: right">
          <el-button @click="$router.push('/')">返回书架</el-button>
        </el-col>
      </el-row>
    </el-header>
    <el-main>
      <el-form label-width="120px" v-loading="loading">
        <el-divider content-position="left">界面</el-divider>

        <el-form-item :label="labels['ui.theme']">
          <el-select v-model="settings['ui.theme']">
            <el-option v-for="o in themeOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>

        <el-form-item :label="labels['ui.page_size']">
          <el-input-number v-model.number="settings['ui.page_size']" :min="5" :max="100" />
        </el-form-item>

        <el-form-item :label="labels['ui.sort_field']">
          <el-select v-model="settings['ui.sort_field']">
            <el-option v-for="o in sortFieldOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>

        <el-form-item :label="labels['ui.sort_order']">
          <el-select v-model="settings['ui.sort_order']">
            <el-option v-for="o in sortOrderOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>

        <el-divider content-position="left">阅读器</el-divider>

        <el-form-item :label="labels['reader.font_size']">
          <el-slider v-model.number="settings['reader.font_size']" :min="12" :max="32" show-input />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="save">保存</el-button>
        </el-form-item>
      </el-form>
    </el-main>
  </el-container>
</template>