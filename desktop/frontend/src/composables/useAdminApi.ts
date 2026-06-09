import { ref } from 'vue'
import { useStatus } from '@/composables/useStatus'
import { GetAdminAccessKey } from '@bindings/github.com/BenedictKing/ccx/desktop/desktopservice'

/**
 * 统一封装 Desktop 前端对本地后端 /api/* 管理接口的 HTTP 调用。
 * 复用 RuntimeCircuitBreakerCard.vue 中已验证的 fetch + GetAdminAccessKey + x-api-key 模式。
 */

export class AdminApiError extends Error {
  status: number
  body: unknown

  constructor(message: string, status: number, body?: unknown) {
    super(message)
    this.name = 'AdminApiError'
    this.status = status
    this.body = body
  }
}

const loading = ref(false)
const error = ref('')

function clearError() {
  error.value = ''
}

async function getAdminKey(): Promise<string> {
  return await GetAdminAccessKey()
}

function buildUrl(path: string): string | null {
  const { status } = useStatus()
  if (!status.value.running || !status.value.url) {
    return null
  }
  return `${status.value.url}${path}`
}

async function request<T = unknown>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const url = buildUrl(path)
  if (!url) {
    throw new AdminApiError('服务未运行，无法发送请求', 0)
  }

  const adminKey = await getAdminKey()

  const headers: Record<string, string> = {
    'x-api-key': adminKey,
  }

  const init: RequestInit = {
    method,
    headers,
  }

  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    init.body = JSON.stringify(body)
  }

  let resp: Response
  try {
    resp = await fetch(url, init)
  } catch (e) {
    throw new AdminApiError(
      '服务未运行或网络不可达，请检查后端是否已启动',
      0,
      e,
    )
  }

  if (!resp.ok) {
    let errBody: unknown
    try {
      errBody = await resp.json()
    } catch {
      errBody = null
    }
    const msg =
      (errBody as Record<string, string>)?.error ||
      (errBody as Record<string, string>)?.message ||
      `HTTP ${resp.status}`
    throw new AdminApiError(msg, resp.status, errBody)
  }

  // 204 No Content
  if (resp.status === 204) {
    return undefined as T
  }

  return (await resp.json()) as T
}

/**
 * 组合式 HTTP 调用接口，返回 loading/error 状态和便捷方法。
 * 调用方可在 try/catch 中使用 error ref 展示错误信息。
 */
export function useAdminApi() {
  return {
    loading,
    error,
    clearError,

    /** 发送 GET 请求 */
    get<T = unknown>(path: string): Promise<T> {
      return request<T>('GET', path)
    },

    /** 发送 POST 请求 */
    post<T = unknown>(path: string, body?: unknown): Promise<T> {
      return request<T>('POST', path, body)
    },

    /** 发送 PUT 请求 */
    put<T = unknown>(path: string, body?: unknown): Promise<T> {
      return request<T>('PUT', path, body)
    },

    /** 发送 PATCH 请求 */
    patch<T = unknown>(path: string, body?: unknown): Promise<T> {
      return request<T>('PATCH', path, body)
    },

    /** 发送 DELETE 请求 */
    del<T = unknown>(path: string): Promise<T> {
      return request<T>('DELETE', path)
    },

    /**
     * 通用请求（高级场景）
     * 支持自定义 method/body，返回 raw Response。
     */
    raw(method: string, path: string, body?: unknown): Promise<Response> {
      const url = buildUrl(path)
      if (!url) {
        throw new AdminApiError('服务未运行，无法发送请求', 0)
      }
      return getAdminKey().then((adminKey) => {
        const headers: Record<string, string> = {
          'x-api-key': adminKey,
        }
        const init: RequestInit = { method, headers }
        if (body !== undefined) {
          headers['Content-Type'] = 'application/json'
          init.body = JSON.stringify(body)
        }
        return fetch(url, init)
      })
    },
  }
}
