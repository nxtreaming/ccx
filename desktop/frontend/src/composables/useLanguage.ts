import { ref, watch } from 'vue'
import { useI18n as vueUseI18n } from 'vue-i18n'
import { applyDocumentLanguage, resolveInitialLocale } from '@/i18n/core'
import type { SupportedLocale } from '@/i18n/messages'
import { defaultLocale, languageOptions } from '@/i18n/messages'
import {
  GetLanguagePreference,
  SaveLanguagePreference,
} from '@bindings/github.com/BenedictKing/ccx/desktop/desktopservice'

const locale = ref<SupportedLocale>(defaultLocale)
const languageReady = ref(false)
let initPromise: Promise<void> | null = null

export const useLanguage = () => {
  const vueI18n = vueUseI18n()

  const t = (key: string, params?: Record<string, string | number>) => {
    return vueI18n.t(key, params as Record<string, string>) as string
  }

  /**
   * 动态 key 翻译，带 Go 后端 fallback。
   * vue-i18n 查不到时（返回 key 本身）使用 fallback 参数。
   * 查找顺序：当前 locale → en → fallback 参数
   */
  const tf = (key: string, fallback: string, params?: Record<string, string>) => {
    const resolved = vueI18n.t(key, params as Record<string, string>) as string
    if (resolved !== key) return resolved
    // vue-i18n 中无此 key，使用 fallback（通常来自 Go 后端）
    if (!params) return fallback
    return Object.entries(params).reduce((acc, [k, v]) => acc.replaceAll(`{${k}}`, v), fallback)
  }

  // 同步 vue-i18n locale 和本地 ref
  watch(locale, (newLocale) => {
    vueI18n.locale.value = newLocale
  })

  const initializeLanguage = async () => {
    if (initPromise) {
      return initPromise
    }
    initPromise = (async () => {
      try {
        const preference = await GetLanguagePreference()
        locale.value = resolveInitialLocale(preference.locale, preference.systemLocale)
      } catch {
        locale.value = defaultLocale
      } finally {
        vueI18n.locale.value = locale.value
        applyDocumentLanguage(locale.value)
        languageReady.value = true
      }
    })()
    return initPromise
  }

  const setLanguage = async (next: SupportedLocale) => {
    locale.value = next
    vueI18n.locale.value = next
    applyDocumentLanguage(next)
    try {
      await SaveLanguagePreference(next)
    } catch {
      // Wails API 失败不阻断 UI 切换
    }
  }

  return {
    locale,
    languageReady,
    languageOptions,
    initializeLanguage,
    setLanguage,
    t,
    tf,
  }
}
