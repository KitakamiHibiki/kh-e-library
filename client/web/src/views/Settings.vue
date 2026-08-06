<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { getSettings, updateSettings } from '@/api'
import { getSavedLocale, saveLocale, type SupportedLocale } from '@/locales'
import { useTheme } from '@/composables/useTheme'

const { t, locale } = useI18n()
const { loadTheme } = useTheme()

const settings = ref<Record<string, string>>({})
const savedSettings = ref<Record<string, string>>({})
const loading = ref(false)

const themeOptions = computed(() => [
  { label: t('settings.themeOptions.light'), value: 'light' },
  { label: t('settings.themeOptions.dark'), value: 'dark' },
  { label: t('settings.themeOptions.auto'), value: 'auto' },
])

const sortFieldOptions = computed(() => [
  { label: t('settings.sortFieldOptions.updated_at'), value: 'updated_at' },
  { label: t('settings.sortFieldOptions.title'), value: 'title' },
  { label: t('settings.sortFieldOptions.author'), value: 'author' },
  { label: t('settings.sortFieldOptions.created_at'), value: 'created_at' },
])

const sortOrderOptions = computed(() => [
  { label: t('settings.sortOrderOptions.desc'), value: 'desc' },
  { label: t('settings.sortOrderOptions.asc'), value: 'asc' },
])

const languageOptions = computed(() => [
  { label: t('settings.langZhCN'), value: 'zh-CN' },
  { label: t('settings.langEn'), value: 'en' },
])

const currentLanguage = computed({
  get: () => getSavedLocale(),
  set: (val: SupportedLocale) => {
    saveLocale(val)
    locale.value = val
  },
})

// Computed number accessors for settings stored as strings
// (backend expects string values per spec §4.6, but el-input-number/el-slider need numbers)
const pageSizeNum = computed({
  get: () => parseInt(settings.value['ui.page_size']) || 20,
  set: (val: number) => { settings.value['ui.page_size'] = String(val) },
})
const fontSizeNum = computed({
  get: () => parseInt(settings.value['reader.font_size']) || 16,
  set: (val: number) => { settings.value['reader.font_size'] = String(val) },
})

const fetchSettings = async () => {
  loading.value = true
  try {
    const res = await getSettings()
    settings.value = res.data.data
    savedSettings.value = { ...res.data.data }
  } catch {
    ElMessage.error(t('settings.loadFailed'))
  } finally {
    loading.value = false
  }
}

const save = async () => {
  try {
    await updateSettings(settings.value)
    // Update saved snapshot on success
    savedSettings.value = { ...settings.value }
    // Apply theme change immediately if ui.theme was updated
    if (settings.value['ui.theme'] !== undefined) {
      loadTheme()
    }
    ElMessage.success(t('settings.saveSuccess'))
  } catch {
    // Rollback to last successfully saved state on failure
    settings.value = { ...savedSettings.value }
    ElMessage.error(t('settings.saveFailed'))
  }
}

onMounted(fetchSettings)
</script>

<template>
  <el-container class="settings-container">
    <el-header class="settings-header">
      <h1 class="settings-title">{{ t('settings.title') }}</h1>
      <el-button @click="$router.push('/')">{{ t('settings.back') }}</el-button>
    </el-header>
    <el-main>
      <el-form label-width="7.5rem" v-loading="loading" class="settings-form">
        <el-divider content-position="left">{{ t('settings.interface') }}</el-divider>

        <el-form-item :label="t('settings.language')">
          <el-select v-model="currentLanguage" class="settings-select">
            <el-option v-for="o in languageOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>

        <el-form-item :label="t('settings.labels.ui.theme')">
          <el-select v-model="settings['ui.theme']" class="settings-select">
            <el-option v-for="o in themeOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>

        <el-form-item :label="t('settings.labels.ui.sortField')">
          <el-select v-model="settings['ui.sort_field']" class="settings-select">
            <el-option v-for="o in sortFieldOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>

        <el-form-item :label="t('settings.labels.ui.sortOrder')">
          <el-select v-model="settings['ui.sort_order']" class="settings-select">
            <el-option v-for="o in sortOrderOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>

        <el-form-item :label="t('settings.labels.ui.pageSize')">
          <el-input-number v-model="pageSizeNum" :min="5" :max="100" class="settings-input-number" />
        </el-form-item>

        <el-divider content-position="left">{{ t('settings.reader') }}</el-divider>

        <el-form-item :label="t('settings.labels.reader.fontSize')">
          <el-slider v-model="fontSizeNum" :min="12" :max="32" show-input class="settings-slider" />
        </el-form-item>

        <el-divider content-position="left">{{ t('settings.storage') }}</el-divider>

        <el-form-item :label="t('settings.labels.storage.driver')">
          <el-select v-model="settings['storage.driver']" class="settings-select" disabled>
            <el-option :label="t('settings.storageOptions.local')" value="local" />
          </el-select>
        </el-form-item>

        <el-form-item :label="t('settings.labels.storage.local.booksDir')">
          <el-input v-model="settings['storage.local.books_dir']" class="settings-input" />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="save">{{ t('settings.save') }}</el-button>
        </el-form-item>
      </el-form>
    </el-main>
  </el-container>
</template>

<style scoped>
.settings-container {
  min-height: 100vh;
  background: var(--bg-primary, #fff);
  color: var(--text-primary, #303133);
}
.settings-header {
  display: flex;
  align-items: center;
  border-bottom: 1px solid var(--border-color, #e0e0e0);
}
.settings-title {
  margin: 0;
  flex: 1;
}
.settings-form {
  max-width: 35rem;
}
.settings-select {
  width: 12.5rem;
}
.settings-input {
  max-width: 18.75rem;
}
.settings-input-number {
  width: 8.75rem;
}
.settings-slider {
  max-width: 17.5rem;
}
</style>
