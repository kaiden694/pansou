import axios from 'axios'
import type { HealthResponse, PluginHealthResponse, SearchResponse } from '../types/api'

interface ApiResponse<T> {
  code: number
  message: string
  data?: T
}

const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  timeout: 60000,
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('pansou_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

export async function getHealth() {
  const { data } = await client.get<HealthResponse>('/health')
  return data
}

export async function getPluginHealth() {
  const { data } = await client.get<PluginHealthResponse>('/health/plugins')
  return data
}

export async function search(keyword: string, forceRefresh = false) {
  const { data } = await client.post<ApiResponse<SearchResponse>>('/search', {
    kw: keyword,
    res: 'merge',
    src: 'all',
    refresh: forceRefresh,
  })
  if (data.code !== 0) throw new Error(data.message || '搜索失败')
  return data.data || { total: 0, results: [], merged_by_type: {} }
}

export async function login(username: string, password: string) {
  const { data } = await client.post<Record<string, unknown>>('/auth/login', { username, password })
  const token = String(data.token || data.access_token || '')
  if (!token) throw new Error('登录响应中未返回 Token')
  localStorage.setItem('pansou_token', token)
  return token
}