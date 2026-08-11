<script setup lang="ts">
import { ref, computed } from 'vue'
import type { UploadInstance, UploadRequestOptions } from 'element-plus'
import { initChunkUpload, uploadChunk } from '@/api'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  uploaded: []
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const uploadRef = ref<UploadInstance>()
const uploading = ref(false)
const uploadSuccess = ref(false)
const errorMsg = ref('')
const progress = ref(0)
const selectedFileName = ref('')
const selectedFileSize = ref(0)
const chunkText = ref('')
let successTimer: ReturnType<typeof setTimeout> | null = null

// Each upload chunk is capped at 50MB.
const CHUNK_SIZE = 50 * 1024 * 1024

const formatFileSize = (bytes: number) => {
  if (!bytes) return '-'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
}

const handleUpload = async (options: UploadRequestOptions) => {
  errorMsg.value = ''
  uploading.value = true
  progress.value = 0
  chunkText.value = ''
  selectedFileName.value = options.file.name
  selectedFileSize.value = options.file.size

  try {
    const file = options.file
    if (file.size === 0) {
      errorMsg.value = t('upload.emptyFile')
      uploadRef.value?.clearFiles()
      selectedFileName.value = ''
      selectedFileSize.value = 0
      return
    }

    const totalChunks = Math.ceil(file.size / CHUNK_SIZE)

    // Phase 1: create the upload session
    const initRes = await initChunkUpload({
      file_name: file.name,
      file_size: file.size,
      total_chunks: totalChunks,
      chunk_size: CHUNK_SIZE,
    })
    const uploadId = initRes.data.data.upload_id

    // Phase 2: upload chunks sequentially, aggregating progress across them
    let uploadedBytes = 0
    for (let i = 0; i < totalChunks; i++) {
      const start = i * CHUNK_SIZE
      const end = Math.min(start + CHUNK_SIZE, file.size)
      const chunk = file.slice(start, end)

      chunkText.value = t('upload.uploadingChunk', { current: i + 1, total: totalChunks })

      await uploadChunk(uploadId, i, totalChunks, chunk, (e) => {
        if (e.total) {
          progress.value = Math.round(((uploadedBytes + e.loaded) / file.size) * 100)
        }
      })

      uploadedBytes += end - start
    }

    progress.value = 100
    uploadSuccess.value = true
    successTimer = setTimeout(() => {
      visible.value = false
      emit('uploaded')
    }, 1500)
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { msg?: string } } })?.response?.data?.msg
    errorMsg.value = msg || t('upload.chunkFailed')
    uploadRef.value?.clearFiles()
    selectedFileName.value = ''
    selectedFileSize.value = 0
    chunkText.value = ''
  } finally {
    uploading.value = false
  }
}

const onClosed = () => {
  if (successTimer) {
    clearTimeout(successTimer)
    successTimer = null
  }
  uploading.value = false
  uploadSuccess.value = false
  errorMsg.value = ''
  progress.value = 0
  selectedFileName.value = ''
  selectedFileSize.value = 0
  chunkText.value = ''
  uploadRef.value?.clearFiles()
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="t('upload.dialogTitle')"
    width="480px"
    :close-on-click-modal="false"
    @closed="onClosed"
  >
    <!-- Error alert -->
    <el-alert
      v-if="errorMsg"
      type="error"
      show-icon
      :closable="false"
      class="upload-error"
    >
      {{ errorMsg }}
    </el-alert>

    <!-- Success result -->
    <div v-if="uploadSuccess" class="upload-success">
      <el-result
        icon="success"
        :title="t('upload.uploadSuccess')"
      />
    </div>

    <!-- Upload area (idle / uploading / error) -->
    <div v-if="!uploadSuccess" class="upload-body">
      <el-upload
        ref="uploadRef"
        drag
        :auto-upload="true"
        :show-file-list="false"
        :accept="'.epub,.pdf,.EPUB,.PDF'"
        :disabled="uploading"
        :http-request="handleUpload"
        class="upload-dropzone"
      >
        <div class="upload-dropzone-content">
          <span class="upload-hint">{{ t('upload.dragHint') }}</span>
          <el-button type="primary" plain :disabled="uploading">
            {{ t('upload.selectFile') }}
          </el-button>
          <span class="upload-formats">{{ t('upload.supportedFormats') }}</span>
        </div>
      </el-upload>

      <!-- Progress section -->
      <div v-if="uploading" class="upload-progress">
        <div class="upload-progress-file">
          <span class="upload-progress-name">{{ selectedFileName }}</span>
          <span class="upload-progress-size">{{ formatFileSize(selectedFileSize) }}</span>
        </div>
        <el-progress :percentage="progress" :status="progress === 100 ? 'success' : ''" />
        <div class="upload-progress-text">{{ chunkText || t('upload.uploading') }}</div>
      </div>
    </div>
  </el-dialog>
</template>

<style scoped lang="less">
@import '../styles/variables.less';

.upload-error {
  margin-bottom: @gap-md;
}

.upload-success {
  padding: calc(20 * @h) 0;
}

.upload-body {
  display: flex;
  flex-direction: column;
  gap: @gap-md;
}

.upload-dropzone {
  width: 100%;
}

.upload-dropzone-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: @gap-sm;
  padding: calc(20 * @h) calc(16 * @w);
}

.upload-hint {
  font-size: @font-md;
  color: var(--text-secondary, #909399);
}

.upload-formats {
  font-size: @font-xs;
  color: var(--text-secondary, #909399);
}

.upload-progress {
  padding: @gap-sm 0;
}

.upload-progress-file {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: @gap-sm;
}

.upload-progress-name {
  font-size: @font-sm;
  color: var(--text-primary, #303133);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
  margin-right: @gap-md;
}

.upload-progress-size {
  font-size: @font-xs;
  color: var(--text-secondary, #909399);
  flex-shrink: 0;
}

.upload-progress-text {
  margin-top: @gap-xs;
  font-size: @font-xs;
  color: var(--text-secondary, #909399);
  text-align: center;
}
</style>
