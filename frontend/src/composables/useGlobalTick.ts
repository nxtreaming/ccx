/**
 * useGlobalTick — 中心化 tick 源 + 可见性暂停
 *
 * 多组件共用同一频率的 setInterval，避免每个组件独立创建定时器。
 * visibilityState === 'hidden' 时自动暂停，恢复时若距上次 tick 超过 intervalMs 则立即触发一次。
 *
 * Usage:
 *   // 在 <script setup> 或 composable 中
 *   const tick = useGlobalTick(5000, 'MyComponent')
 *   tick.onTick(() => { refreshData(true) })
 *
 *   // 可选：手动暂停/恢复（多数场景不需要，visibility 自动处理）
 *   tick.pause()
 *   tick.resume()
 */

import { onUnmounted } from 'vue'

interface TimerEntry {
  intervalId: ReturnType<typeof setInterval> | null
  listeners: Set<() => void | Promise<void>>
  lastTickAt: number
  paused: boolean
}

// 相同 intervalMs 共享一个 TimerEntry（键：毫秒数）
const entries = new Map<number, TimerEntry>()

// 全局 visibility 监听（只注册一次）
let visibilityListenerAttached = false

function ensureVisibilityListener(): void {
  if (visibilityListenerAttached) return
  visibilityListenerAttached = true

  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') {
      // 暂停所有 entry
      for (const entry of entries.values()) {
        if (entry.paused) continue
        entry.paused = true
        if (entry.intervalId !== null) {
          clearInterval(entry.intervalId)
          entry.intervalId = null
        }
      }
    } else {
      // 恢复所有 entry，距上次 tick 超过 intervalMs 则立即触发一次
      const now = Date.now()
      for (const [intervalMs, entry] of entries.entries()) {
        if (!entry.paused) continue
        entry.paused = false
        if (now - entry.lastTickAt >= intervalMs) {
          entry.lastTickAt = now
          fireListeners(entry.listeners)
        }
        entry.intervalId = setInterval(() => {
          entry.lastTickAt = Date.now()
          fireListeners(entry.listeners)
        }, intervalMs)
      }
    }
  })
}

function fireListeners(listeners: Set<() => void | Promise<void>>): void {
  for (const fn of listeners) {
    try {
      const result = fn()
      // 如果返回 Promise，静默吞掉未捕获异常
      if (result && typeof (result as Promise<void>).catch === 'function') {
        (result as Promise<void>).catch(err => console.warn('[useGlobalTick] listener error:', err))
      }
    } catch (err) {
      console.warn('[useGlobalTick] listener error:', err)
    }
  }
}

function getOrCreateEntry(intervalMs: number): TimerEntry {
  const existing = entries.get(intervalMs)
  if (existing) return existing

  // 若当前 tab 已隐藏，创建时即标记 paused，不启动 setInterval
  // visible 分支会在下次恢复时补触发并启动定时器（P1 修复：避免后台 tab 首次注册时定时器继续运行）
  const isHiddenAtCreate = typeof document !== 'undefined' && document.visibilityState === 'hidden'

  const entry: TimerEntry = {
    intervalId: isHiddenAtCreate ? null : setInterval(() => {
      entry.lastTickAt = Date.now()
      fireListeners(entry.listeners)
    }, intervalMs),
    listeners: new Set(),
    lastTickAt: Date.now(),
    paused: isHiddenAtCreate,
  }
  entries.set(intervalMs, entry)
  return entry
}

function removeListener(intervalMs: number, listener: () => void | Promise<void>): void {
  const entry = entries.get(intervalMs)
  if (!entry) return
  entry.listeners.delete(listener)
  // 最后一个订阅者退订后，清理 entry
  if (entry.listeners.size === 0) {
    if (entry.intervalId !== null) clearInterval(entry.intervalId)
    entries.delete(intervalMs)
  }
}

export interface GlobalTickHandle {
  /** 注册一个 tick 回调（自动在组件 unmount 时退订） */
  onTick: (fn: () => void | Promise<void>) => void
  /** 手动暂停本组件的 tick（可选，visibility 自动管理） */
  pause: () => void
  /** 手动恢复本组件的 tick（可选，visibility 自动管理） */
  resume: () => void
}

/**
 * @param intervalMs tick 间隔（毫秒），相同间隔的组件共用同一个底层 setInterval
 * @param debugName 仅用于调试，无实际作用
 */
export function useGlobalTick(intervalMs: number, _debugName?: string): GlobalTickHandle {
  ensureVisibilityListener()

  let currentListener: (() => void | Promise<void>) | null = null
  let paused = false // 组件级暂停标志

  // 组件级 pause：暂停接收 tick（不影响其他订阅者）
  const pause = (): void => { paused = true }
  const resume = (): void => { paused = false }

  // 关键：在 setup 同步阶段就注册 onUnmounted（而不是在 onTick 里），
  // 这样即使用户在 onMounted 回调里才调 onTick，卸载清理依旧生效。
  // 用闭包捕获 currentListener 的最新值，而不是固定绑定某个 listener。
  try {
    onUnmounted(() => {
      if (currentListener) {
        removeListener(intervalMs, currentListener)
        currentListener = null
      }
    })
  } catch {
    // 非组件上下文（例如 store）调用时 onUnmounted 会告警，忽略
    // 这些调用者应使用 registerGlobalTick 并手动退订
  }

  const onTick = (fn: () => void | Promise<void>): void => {
    // 先移除旧回调（防止重复注册）
    if (currentListener) removeListener(intervalMs, currentListener)

    // 包装一层：检查组件级 pause；返回 fn() 的 Promise 以便 fireListeners 捕获异步拒绝
    const wrapped = () => {
      if (!paused) return fn()
    }
    currentListener = wrapped

    const entry = getOrCreateEntry(intervalMs)
    entry.listeners.add(wrapped)
  }

  return { onTick, pause, resume }
}

/**
 * store 级别的 tick 注册（不依赖组件 onMounted 上下文，需手动退订）
 *
 * 返回 unsubscribe 函数；退订后如果该 entry 无订阅者，会自动清理 setInterval。
 */
export function registerGlobalTick(intervalMs: number, fn: () => void | Promise<void>): () => void {
  ensureVisibilityListener()
  const entry = getOrCreateEntry(intervalMs)
  const wrapped = () => fn()
  entry.listeners.add(wrapped)
  return () => removeListener(intervalMs, wrapped)
}

/**
 * 测试专用：清理所有模块级状态（entries Map + 定时器）
 *
 * 仅在单元测试 afterEach 中调用，用于隔离测试间状态。
 * @internal
 */
export function __resetForTests__(): void {
  for (const entry of entries.values()) {
    if (entry.intervalId !== null) clearInterval(entry.intervalId)
  }
  entries.clear()
}
