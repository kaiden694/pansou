// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  requestUse: vi.fn(),
}))

vi.mock('axios', () => ({
  default: {
    create: vi.fn(() => ({
      get: mocks.get,
      post: mocks.post,
      interceptors: { request: { use: mocks.requestUse } },
    })),
  },
}))

import { login, search } from './client'

describe('search API client', () => {
  beforeEach(() => {
    localStorage.clear()
    mocks.post.mockReset()
  })

  it('stores the token returned by a successful login', async () => {
    mocks.post.mockResolvedValue({ data: { token: 'test-token' } })

    await expect(login('tester', 'password')).resolves.toBe('test-token')

    expect(mocks.post).toHaveBeenCalledWith('/auth/login', {
      username: 'tester',
      password: 'password',
    })
    expect(localStorage.getItem('pansou_token')).toBe('test-token')
  })

  it('rejects a login response without a token', async () => {
    mocks.post.mockResolvedValue({ data: {} })

    await expect(login('tester', 'password')).rejects.toThrow('登录响应中未返回 Token')
    expect(localStorage.getItem('pansou_token')).toBeNull()
  })
  it('unwraps a successful API response', async () => {
    const expected = {
      total: 1,
      results: [],
      merged_by_type: {
        quark: [{ url: 'https://example.test/resource', password: '1234' }],
      },
    }
    mocks.post.mockResolvedValue({ data: { code: 0, message: 'success', data: expected } })

    await expect(search('软件')).resolves.toEqual(expected)
    expect(mocks.post).toHaveBeenCalledWith('/search', {
      kw: '软件',
      res: 'merge',
      src: 'all',
      refresh: false,
    })
  })

  it('returns a complete empty result when data is omitted', async () => {
    mocks.post.mockResolvedValue({ data: { code: 0, message: 'success' } })

    await expect(search('空结果')).resolves.toEqual({
      total: 0,
      results: [],
      merged_by_type: {},
    })
  })

  it('throws the backend message for a business error', async () => {
    mocks.post.mockResolvedValue({ data: { code: 500, message: '搜索失败' } })

    await expect(search('错误')).rejects.toThrow('搜索失败')
  })

  it('passes the force refresh flag', async () => {
    mocks.post.mockResolvedValue({ data: { code: 0, message: 'success', data: { total: 0 } } })

    await search('刷新', true)
    expect(mocks.post).toHaveBeenCalledWith('/search', expect.objectContaining({ refresh: true }))
  })
})
