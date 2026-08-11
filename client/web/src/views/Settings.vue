<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { checkUpdate, getSettings, getSystemStatus, getUpdateStatus, startUpdate, updateSettings } from '@/api'
import type { UpdateCheckResult, UpdateStatus } from '@/types/book'
import { getSavedLocale, saveLocale, type SupportedLocale } from '@/locales'
import { useTheme } from '@/composables/useTheme'

const { t, locale } = useI18n()
const { loadTheme } = useTheme()

const settings = ref<Record<string, string>>({})
const savedSettings = ref<Record<string, string>>({})
const loading = ref(false)

// Software update state
const currentVersion = ref('')
const updateResult = ref<UpdateCheckResult | null>(null)
const checking = ref(false)
// True while the async download+install task is running or before it resolves.
const updating = ref(false)
const updateStatus = ref<UpdateStatus | null>(null)
let updatePollTimer: ReturnType<typeof setInterval> | null = null

const stopUpdatePolling = () => {
  if (updatePollTimer) {
    clearInterval(updatePollTimer)
    updatePollTimer = null
  }
}

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
    settings.value['update.github_repo'] = settings.value['update.github_repo'] || ''
  } catch {
    ElMessage.error(t('settings.loadFailed'))
  } finally {
    loading.value = false
  }
}

// ---- Software update handlers ----

const errMsg = (e: unknown, fallback: string) => {
  const msg = (e as { response?: { data?: { msg?: string } } })?.response?.data?.msg
  return msg || fallback
}

const formatFileSize = (bytes: number) => {
  if (!bytes) return '-'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
}

const formatDate = (iso: string) => {
  if (!iso) return '-'
  const d = new Date(iso)
  return d.toString() === 'Invalid Date' ? '-' : d.toLocaleDateString()
}

const fetchSystemStatus = async () => {
  try {
    const res = await getSystemStatus()
    currentVersion.value = res.data.data.version
  } catch {
    // Non-fatal: version just stays unknown.
  }
}

const handleCheckUpdate = async () => {
  const repo = (settings.value['update.github_repo'] || '').trim()
  if (!repo) {
    ElMessage.warning(t('settings.update.notConfigured'))
    return
  }
  checking.value = true
  updateResult.value = null
  try {
    const res = await checkUpdate(repo)
    updateResult.value = res.data.data
  } catch (e: unknown) {
    ElMessage.error(errMsg(e, t('settings.update.checkFailed')))
  } finally {
    checking.value = false
  }
}

const handleStartUpdate = async () => {
  if (!updateResult.value?.download_url) {
    ElMessage.warning(t('settings.update.noDownloadUrl'))
    return
  }
  try {
    await ElMessageBox.confirm(t('settings.update.installConfirm'), t('settings.update.startUpdate'), {
      confirmButtonText: t('settings.update.startUpdate'),
      cancelButtonText: t('settings.update.cancel'),
      type: 'warning',
    })
  } catch {
    return // user cancelled
  }
  updating.value = true
  updateStatus.value = null
  try {
    await startUpdate(updateResult.value.download_url)
    // Poll the async task until it completes, fails, or the server restarts.
    updatePollTimer = setInterval(async () => {
      try {
        const res = await getUpdateStatus()
        updateStatus.value = res.data.data
        if (res.data.data.state === 'completed') {
          stopUpdatePolling()
          updating.value = false
          ElMessage.success(t('settings.update.updateCompleted'))
        } else if (res.data.data.state === 'failed') {
          stopUpdatePolling()
          updating.value = false
          ElMessage.error(res.data.data.message || t('settings.update.updateFailed'))
        }
      } catch {
        // Transient (server restarting / draining) — keep polling or give up.
      }
    }, 1000)
  } catch (e: unknown) {
    updating.value = false
    ElMessage.error(errMsg(e, t('settings.update.updateFailed')))
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

onMounted(() => {
  fetchSettings()
  fetchSystemStatus()
})

onBeforeUnmount(() => {
  stopUpdatePolling()
})
</script>

<template>
  <el-container class="settings-container">
    <el-header class="settings-header">
      <h1 class="settings-title">{{ t('settings.title') }}</h1>
      <el-button @click="$router.replace('/')">{{ t('settings.back') }}</el-button>
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

        <el-divider content-position="left">{{ t('settings.update.title') }}</el-divider>

        <el-form-item :label="t('settings.update.currentVersion')">
          <span class="update-version">{{ currentVersion || t('settings.update.unknown') }}</span>
        </el-form-item>

        <el-form-item :label="t('settings.update.githubRepo')">
          <el-input
            v-model="settings['update.github_repo']"
            :placeholder="t('settings.update.githubRepoHint')"
            class="settings-input"
            clearable
          />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" plain :loading="checking" @click="handleCheckUpdate">
            {{ checking ? t('settings.update.checking') : t('settings.update.checkUpdate') }}
          </el-button>
        </el-form-item>

        <div v-if="updateResult" class="update-result">
          <template v-if="updateResult.has_update">
            <el-alert type="success" :closable="false" show-icon>
              <template #title>{{ t('settings.update.newVersionFound') }} {{ updateResult.latest_version }}</template>
              <div v-if="updateResult.file_size || updateResult.published_at" class="update-meta">
                <span v-if="updateResult.file_size">{{ t('settings.update.fileSize') }}: {{ formatFileSize(updateResult.file_size) }}</span>
                <span v-if="updateResult.published_at">{{ t('settings.update.publishedAt') }}: {{ formatDate(updateResult.published_at) }}</span>
              </div>
            </el-alert>
            <div v-if="updateResult.release_notes" class="update-notes">
              <div class="update-notes-title">{{ t('settings.update.releaseNotes') }}</div>
              <pre>{{ updateResult.release_notes }}</pre>
            </div>
            <div class="update-actions">
              <el-button type="primary" :loading="updating" :disabled="!updateResult.download_url" @click="handleStartUpdate">
                {{ updating ? t('settings.update.updating') : t('settings.update.startUpdate') }}
              </el-button>
            </div>
            <div
              v-if="updateStatus && (updateStatus.state === 'downloading' || updateStatus.state === 'installing')"
              class="update-progress"
            >
              <el-progress
                :percentage="updateStatus.progress"
                :status="updateStatus.progress === 100 ? 'success' : ''"
              />
              <div class="update-progress-text">
                {{
                  updateStatus.state === 'downloading'
                    ? t('settings.update.downloadingStage')
                    : t('settings.update.installingStage')
                }}
              </div>
            </div>
            <el-alert
              v-if="updateStatus && updateStatus.state === 'failed'"
              class="update-error"
              type="error"
              :closable="false"
              show-icon
              :title="updateStatus.message || t('settings.update.updateFailed')"
            />
          </template>
          <template v-else>
            <el-alert type="info" :closable="false" show-icon :title="t('settings.update.upToDate')" />
          </template>
        </div>

        <p class="update-manual-hint">{{ t('settings.update.manualHint') }}</p>

        <el-form-item>
          <el-button type="primary" @click="save">{{ t('settings.save') }}</el-button>
        </el-form-item>
      </el-form>
    </el-main>
  </el-container>
</template>

<style scoped lang="less">
@import '../styles/variables.less';

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
  max-width: @settings-form-max;
}
.settings-select {
  width: @settings-select-w;
}
.settings-input {
  max-width: @settings-input-w;
}
.settings-input-number {
  width: @settings-number-w;
}
.settings-slider {
  max-width: @settings-slider-w;
}
.update-version {
  font-size: @font-md;
  font-weight: 600;
  color: var(--text-primary, #303133);
}
.update-result {
  margin-bottom: @gap-md;
}
.update-meta {
  display: flex;
  gap: @gap-lg;
  color: var(--text-secondary, #666);
  font-size: @font-xs;
  margin-top: @gap-xs;
}
.update-notes {
  margin-top: @gap-md;
  border: 1px solid var(--border-color, #e0e0e0);
  border-radius: @radius-sm;
  padding: @gap-sm @gap-md;
  background: var(--bg-secondary, #f5f5f5);
}
.update-notes-title {
  font-weight: 600;
  margin-bottom: @gap-xs;
}
.update-notes pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: calc(160 * @h);
  overflow-y: auto;
  font-family: inherit;
  font-size: @font-sm;
  color: var(--text-primary, #303133);
}
.update-actions {
  margin-top: @gap-md;
  display: flex;
  gap: @gap-sm;
}
.update-progress {
  margin-top: @gap-md;
}
.update-progress-text {
  margin-top: @gap-xs;
  font-size: @font-xs;
  color: var(--text-secondary, #999);
}
.update-error {
  margin-top: @gap-md;
}
.update-manual-hint {
  color: var(--text-secondary, #999);
  font-size: @font-xs;
  margin: 0 0 @gap-lg;
}
</style>
