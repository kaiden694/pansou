<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { CopyDocument, Delete, Link, Lock, Refresh, Search, Star, StarFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getHealth, login, search as searchApi } from './api/client'
import type { HealthResponse, MergedLink, SearchResponse } from './types/api'

type Favorite = MergedLink & { type: string }

const keyword = ref('')
const loading = ref(false)
const forceRefresh = ref(false)
const response = ref<SearchResponse | null>(null)
const health = ref<HealthResponse | null>(null)
const activeType = ref('all')
const loginVisible = ref(false)
const username = ref('')
const password = ref('')
const loggingIn = ref(false)
function loadFavorites(): Favorite[] {
  try {
    const stored = JSON.parse(localStorage.getItem('pansou_favorites') || '[]')
    return Array.isArray(stored) ? stored : []
  } catch {
    localStorage.removeItem('pansou_favorites')
    return []
  }
}

const favorites = ref<Favorite[]>(loadFavorites())

const groupedLinks = computed(() => response.value?.merged_by_type || {})
const types = computed(() => Object.keys(groupedLinks.value))
const visibleGroups = computed(() => {
  if (activeType.value === 'all') return Object.entries(groupedLinks.value)
  return [[activeType.value, groupedLinks.value[activeType.value] || []]] as [string, MergedLink[]][]
})
const resultCount = computed(() => Object.values(groupedLinks.value).reduce((total, links) => total + links.length, 0))

function persistFavorites() {
  localStorage.setItem('pansou_favorites', JSON.stringify(favorites.value))
}

function isFavorite(type: string, item: MergedLink) {
  return favorites.value.some((favorite) => favorite.url === item.url && favorite.type === type)
}

function toggleFavorite(type: string, item: MergedLink) {
  const index = favorites.value.findIndex((favorite) => favorite.url === item.url && favorite.type === type)
  if (index >= 0) favorites.value.splice(index, 1)
  else favorites.value.unshift({ ...item, type })
  persistFavorites()
}

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败，请手动复制')
  }
}

async function runSearch() {
  const value = keyword.value.trim()
  if (!value) return ElMessage.warning('请输入搜索关键词')
  loading.value = true
  activeType.value = 'all'
  try {
    response.value = await searchApi(value, forceRefresh.value)
    if (!resultCount.value) ElMessage.info('没有找到可用资源')
  } catch (error: any) {
    if (error?.response?.status === 401) loginVisible.value = true
    ElMessage.error(error?.response?.data?.message || error?.message || '搜索失败')
  } finally {
    loading.value = false
  }
}

async function submitLogin() {
  if (!username.value || !password.value) return ElMessage.warning('请输入用户名和密码')
  loggingIn.value = true
  try {
    await login(username.value, password.value)
    loginVisible.value = false
    ElMessage.success('登录成功')
    await runSearch()
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.message || error?.message || '登录失败')
  } finally {
    loggingIn.value = false
  }
}

onMounted(async () => {
  try {
    health.value = await getHealth()
  } catch {
    health.value = null
  }
})
</script>

<template>
  <div class="app-shell">
    <header class="topbar">
      <a class="brand" href="#"><span class="brand-mark">P</span><span>PanSou</span></a>
      <div class="top-actions">
        <span class="status" :class="{ offline: !health }"><i></i>{{ health ? '服务正常' : '服务未连接' }}</span>
        <el-button v-if="health?.auth_enabled" text :icon="Lock" @click="loginVisible = true">登录</el-button>
      </div>
    </header>

    <main>
      <section class="hero">
        <div class="eyebrow">聚合网盘搜索</div>
        <h1>一次搜索，发现更多资源</h1>
        <p>整合频道与插件搜索结果，快速查找常用网盘分享链接。</p>
        <form class="search-box" @submit.prevent="runSearch">
          <el-input v-model="keyword" size="large" clearable placeholder="输入影视、软件、课程等关键词" :prefix-icon="Search" />
          <el-button type="primary" size="large" :loading="loading" native-type="submit">搜索</el-button>
        </form>
        <label class="refresh-option"><el-switch v-model="forceRefresh" size="small" />忽略缓存并刷新结果</label>
      </section>

      <section v-if="response" class="results-section">
        <div class="results-head">
          <div><h2>“{{ keyword }}” 的搜索结果</h2><p>共整理出 {{ resultCount }} 个链接，来自 {{ types.length }} 种资源类型</p></div>
          <el-button :icon="Refresh" circle :loading="loading" @click="runSearch" />
        </div>

        <div class="filter-row">
          <button :class="{ active: activeType === 'all' }" @click="activeType = 'all'">全部 {{ resultCount }}</button>
          <button v-for="type in types" :key="type" :class="{ active: activeType === type }" @click="activeType = type">
            {{ type }} {{ groupedLinks[type].length }}
          </button>
        </div>

        <div v-if="!resultCount" class="empty-state"><el-empty description="没有找到可用资源" /></div>
        <div v-for="[type, links] in visibleGroups" :key="type" class="resource-group">
          <div class="group-title"><span>{{ type }}</span><small>{{ links.length }} 个链接</small></div>
          <article v-for="item in links" :key="item.url" class="result-card">
            <div class="result-main">
              <h3>{{ item.note || '未命名资源' }}</h3>
              <div class="meta"><span v-if="item.source">来源：{{ item.source }}</span><span v-if="item.datetime">{{ item.datetime }}</span></div>
              <a class="resource-link" :href="item.url" target="_blank" rel="noreferrer"><el-icon><Link /></el-icon><span>{{ item.url }}</span></a>
              <span v-if="item.password" class="password">提取码：<b>{{ item.password }}</b></span>
              <div v-if="item.images?.length" class="image-gallery">
                <el-image
                  v-for="(image, imageIndex) in item.images.slice(0, 4)"
                  :key="`${item.url}-${imageIndex}`"
                  :src="image"
                  :preview-src-list="item.images"
                  :initial-index="imageIndex"
                  fit="cover"
                  loading="lazy"
                  preview-teleported
                  hide-on-click-modal
                >
                  <template #error><div class="image-error">图片加载失败</div></template>
                </el-image>
                <span v-if="item.images.length > 4" class="image-more">+{{ item.images.length - 4 }}</span>
              </div>
            </div>
            <div class="card-actions">
              <el-tooltip content="复制链接"><el-button :icon="CopyDocument" circle @click="copyText(item.url)" /></el-tooltip>
              <el-tooltip :content="isFavorite(type, item) ? '取消收藏' : '收藏'">
                <el-button :icon="isFavorite(type, item) ? StarFilled : Star" circle @click="toggleFavorite(type, item)" />
              </el-tooltip>
              <el-button type="primary" tag="a" :href="item.url" target="_blank">打开</el-button>
            </div>
          </article>
        </div>
      </section>

      <section v-else class="intro-grid">
        <article><strong>{{ health?.channels_count ?? '—' }}</strong><span>搜索频道</span></article>
        <article><strong>{{ health?.plugin_count ?? '—' }}</strong><span>聚合插件</span></article>
        <article><strong>{{ favorites.length }}</strong><span>本地收藏</span></article>
      </section>

      <section v-if="favorites.length" class="favorites-section">
        <div class="results-head"><div><h2>我的收藏</h2><p>收藏仅保存在当前浏览器</p></div></div>
        <article v-for="item in favorites" :key="`${item.type}-${item.url}`" class="result-card compact">
          <div class="result-main"><h3>{{ item.note || '未命名资源' }} <em>{{ item.type }}</em></h3><a :href="item.url" target="_blank">{{ item.url }}</a></div>
          <el-button :icon="Delete" circle @click="toggleFavorite(item.type, item)" />
        </article>
      </section>
    </main>

    <footer>PanSou Web · 结果来自已配置的频道与插件，请自行判断资源有效性。</footer>

    <el-dialog v-model="loginVisible" title="登录 PanSou" width="min(420px, 92vw)">
      <el-form label-position="top" @submit.prevent="submitLogin">
        <el-form-item label="用户名"><el-input v-model="username" autocomplete="username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="password" type="password" show-password autocomplete="current-password" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="loginVisible = false">取消</el-button><el-button type="primary" :loading="loggingIn" @click="submitLogin">登录</el-button></template>
    </el-dialog>
  </div>
</template>
