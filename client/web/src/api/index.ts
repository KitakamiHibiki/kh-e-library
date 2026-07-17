import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
})

export interface PageResult<T> {
  data: T[]
  total: number
  page: number
}

export default api