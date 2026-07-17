<script setup lang="ts">
import { ref, onMounted } from "vue"
import { ElMessage, ElMessageBox } from "element-plus"
import { getBooks, uploadBook, deleteBook, getCoverUrl } from "@/api"
import type { Book } from "@/types/book"

const books = ref<Book[]>([])
const total = ref(0)
const page = ref(1)
const keyword = ref("")
const loading = ref(false)
const pageSize = 20

const fetchBooks = async () => {
  loading.value = true
  try {
    const res = await getBooks(page.value, pageSize, keyword.value)
    books.value = res.data.data
    total.value = res.data.total
  } finally {
    loading.value = false
  }
}

const handleUpload = async (options: any) => {
  try {
    await uploadBook(options.file)
    ElMessage.success("upload ok")
    fetchBooks()
  } catch {
    ElMessage.error("upload failed")
  }
}

const handleDelete = async (id: number) => {
  try {
    await ElMessageBox.confirm("delete this book?")
    await deleteBook(id)
    ElMessage.success("deleted")
    fetchBooks()
  } catch { /* cancel */ }
}

onMounted(fetchBooks)
</script>

<template>
  <el-container>
    <el-header>
      <el-row align="middle" style="height: 100%" :gutter="12">
        <el-col :span="6">
          <h1 style="margin: 0; font-size: 1.3rem">kh e-library</h1>
        </el-col>
        <el-col :span="8">
          <el-input v-model="keyword" placeholder="search by title or author..." clearable @clear="fetchBooks" @keyup.enter="fetchBooks" />
        </el-col>
        <el-col :span="4" style="text-align: right">
          <el-upload :show-file-list="false" :http-request="handleUpload" accept=".epub,.pdf">
            <el-button type="primary">upload EPUB</el-button>
          </el-upload>
        </el-col>
        <el-col :span="2" style="text-align: right">
          <el-button @click="$router.push('/settings')">settings</el-button>
        </el-col>
      </el-row>
    </el-header>
    <el-main>
      <div v-if="!loading && books.length === 0" style="margin-top: 80px">
        <el-empty description="no books yet. upload an EPUB to get started." />
      </div>
      <el-row :gutter="16" v-loading="loading">
        <el-col v-for="book in books" :key="book.id" :xs="12" :sm="8" :md="6" :lg="4" style="margin-bottom: 16px">
          <el-card :body-style="{ padding: "12px" }" shadow="hover" style="cursor: pointer" @click="$router.push('/read/' + book.id)">
            <div style="aspect-ratio: 3/4; background: #f5f5f5; border-radius: 4px; margin-bottom: 8px; overflow: hidden; display: flex; align-items: center; justify-content: center">
              <img v-if="book.cover_path" :src="getCoverUrl(book.id)" @error="(e:any)=>(e.target.src='')" style="width: 100%; height: 100%; object-fit: cover" />
              <span v-if="book.file_path?.toLowerCase().endsWith('.pdf')" style="color: #e74c3c; font-size: 0.85rem; font-weight: bold">PDF</span>
            <span v-else style="color: #999; font-size: 2rem">+</span>
            </div>
            <h4 style="margin: 0 0 4px; font-size: 0.85rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">{{ book.title }}</h4>
            <p style="margin: 0; color: #999; font-size: 0.75rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">{{ book.author || "unknown author" }}</p>
            <div style="margin-top: 8px; display: flex; gap: 4px">
              <el-button size="small" @click.stop="$router.push('/read/' + book.id)">read</el-button>
              <el-button size="small" type="danger" plain @click.stop="handleDelete(book.id)">delete</el-button>
            </div>
          </el-card>
        </el-col>
      </el-row>
      <div style="margin-top: 16px; text-align: center" v-if="total > pageSize">
        <el-pagination background layout="prev, pager, next" :total="total" :page-size="pageSize" v-model:current-page="page" @current-change="fetchBooks" />
      </div>
    </el-main>
  </el-container>
</template>
