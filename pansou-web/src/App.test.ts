// @vitest-environment jsdom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  getHealth: vi.fn(),
  login: vi.fn(),
  search: vi.fn(),
  messageInfo: vi.fn(),
  messageError: vi.fn(),
  messageWarning: vi.fn(),
  messageSuccess: vi.fn(),
}))

vi.mock('./api/client', () => ({
  getHealth: mocks.getHealth,
  login: mocks.login,
  search: mocks.search,
}))

vi.mock('element-plus', async (importOriginal) => {
  const actual = await importOriginal<typeof import('element-plus')>()
  return {
    ...actual,
    ElMessage: {
      info: mocks.messageInfo,
      error: mocks.messageError,
      warning: mocks.messageWarning,
      success: mocks.messageSuccess,
    },
  }
})

import App from './App.vue'

const searchResponse = {
  total: 2,
  results: [],
  merged_by_type: {
    quark: [{
      url: 'https://example.test/quark',
      password: '1234',
      note: '夸克测试资源',
      datetime: '2026-09-19T00:00:00Z',
      source: 'test',
    }],
    baidu: [{
      url: 'https://example.test/baidu',
      note: '百度测试资源',
      source: 'test',
    }],
  },
}

type AppVm = {
  keyword: string
  forceRefresh: boolean
  activeType: string
  username: string
  password: string
  loginVisible: boolean
  runSearch: () => Promise<void>
  submitLogin: () => Promise<void>
  toggleFavorite: (type: string, item: (typeof searchResponse.merged_by_type.quark)[number]) => void
  copyText: (text: string) => Promise<void>
}

function mountApp() {
  return shallowMount(App)
}

describe('App workflows', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
    mocks.getHealth.mockResolvedValue({ status: 'ok', channels_count: 1, plugin_count: 1, auth_enabled: true })
  })

  it('renders merged links returned by the unwrapped search response', async () => {
    mocks.search.mockResolvedValue(searchResponse)
    const wrapper = mountApp()
    await flushPromises()
    const vm = wrapper.vm as unknown as AppVm

    vm.keyword = '组件回归'
    await vm.runSearch()
    await flushPromises()

    expect(mocks.search).toHaveBeenCalledWith('组件回归', false)
    expect(wrapper.text()).toContain('共整理出 2 个链接')
    expect(wrapper.text()).toContain('夸克测试资源')
    expect(wrapper.text()).toContain('百度测试资源')
  })

  it('passes the force refresh selection to the search client', async () => {
    mocks.search.mockResolvedValue(searchResponse)
    const wrapper = mountApp()
    const vm = wrapper.vm as unknown as AppVm
    vm.keyword = '刷新测试'
    vm.forceRefresh = true

    await vm.runSearch()

    expect(mocks.search).toHaveBeenCalledWith('刷新测试', true)
  })

  it('filters displayed resource groups by type', async () => {
    mocks.search.mockResolvedValue(searchResponse)
    const wrapper = mountApp()
    const vm = wrapper.vm as unknown as AppVm
    vm.keyword = '筛选测试'
    await vm.runSearch()
    await flushPromises()

    vm.activeType = 'baidu'
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('百度测试资源')
    expect(wrapper.text()).not.toContain('夸克测试资源')
  })

  it('adds and removes a favorite while persisting local storage', async () => {
    const wrapper = mountApp()
    const vm = wrapper.vm as unknown as AppVm
    const item = searchResponse.merged_by_type.quark[0]

    vm.toggleFavorite('quark', item)
    await wrapper.vm.$nextTick()
    expect(JSON.parse(localStorage.getItem('pansou_favorites') || '[]')).toEqual([{ ...item, type: 'quark' }])
    expect(wrapper.text()).toContain('我的收藏')

    vm.toggleFavorite('quark', item)
    await wrapper.vm.$nextTick()
    expect(JSON.parse(localStorage.getItem('pansou_favorites') || '[]')).toEqual([])
  })

  it('recovers from malformed favorites stored in local storage', async () => {
    localStorage.setItem('pansou_favorites', '{invalid-json')

    const wrapper = mountApp()
    await wrapper.vm.$nextTick()

    expect(localStorage.getItem('pansou_favorites')).toBeNull()
    expect(wrapper.text()).not.toContain('我的收藏')
  })

  it('reports a clipboard failure without rejecting', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('clipboard unavailable'))
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    })
    const wrapper = mountApp()
    const vm = wrapper.vm as unknown as AppVm

    await expect(vm.copyText('https://example.test/failed-copy')).resolves.toBeUndefined()

    expect(writeText).toHaveBeenCalledWith('https://example.test/failed-copy')
    expect(mocks.messageError).toHaveBeenCalledWith('复制失败，请手动复制')
  })
  it('copies a resource link and reports success', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    })
    const wrapper = mountApp()
    const vm = wrapper.vm as unknown as AppVm

    await vm.copyText('https://example.test/copied')

    expect(writeText).toHaveBeenCalledWith('https://example.test/copied')
    expect(mocks.messageSuccess).toHaveBeenCalledWith('已复制')
  })
  it('opens login after a 401 search failure', async () => {
    mocks.search.mockRejectedValue({ response: { status: 401, data: { message: '需要登录' } } })
    const wrapper = mountApp()
    const vm = wrapper.vm as unknown as AppVm
    vm.keyword = '认证测试'

    await vm.runSearch()
    await wrapper.vm.$nextTick()

    expect(vm.loginVisible).toBe(true)
    expect(mocks.messageError).toHaveBeenCalledWith('需要登录')
  })

  it('logs in and retries the current search', async () => {
    mocks.login.mockResolvedValue('token')
    mocks.search.mockResolvedValue(searchResponse)
    const wrapper = mountApp()
    const vm = wrapper.vm as unknown as AppVm
    vm.keyword = '登录后搜索'
    vm.username = 'tester'
    vm.password = 'secret'
    vm.loginVisible = true

    await vm.submitLogin()

    expect(mocks.login).toHaveBeenCalledWith('tester', 'secret')
    expect(mocks.search).toHaveBeenCalledWith('登录后搜索', false)
    expect(vm.loginVisible).toBe(false)
    expect(mocks.messageSuccess).toHaveBeenCalledWith('登录成功')
  })
})
